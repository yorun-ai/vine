// Package webtimeout defines the WebGW timeout policy shared by Hub and Portal.
package webtimeout

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Config contains an optional default and path overrides ordered only to break
// specificity ties. An empty config clears the policy; unmatched paths then
// retain legacy timeout behavior.
type Config struct {
	Default *Policy `json:"default,omitempty" yaml:"default"`
	Paths   []Path  `json:"paths,omitempty" yaml:"paths"`
}

// Policy sets the total execution budget.
type Policy struct {
	Total string `json:"total" yaml:"total"`
}

// Path overrides the default for a matching request path.
type Path struct {
	Path  string `json:"path" yaml:"path"`
	Total string `json:"total" yaml:"total"`
}

// Compiled is an immutable policy snapshot safe to share between requests.
type Compiled struct {
	defaultTotal time.Duration
	paths        []_Path
}
type _Path struct {
	pattern  *regexp.Regexp
	total    time.Duration
	exact    bool
	segments []string
}

// Compile validates a configuration and builds an immutable matching snapshot.
func Compile(config *Config) (*Compiled, error) {
	compiled := new(Compiled)
	if config == nil {
		return compiled, nil
	}
	if config.Default != nil {
		total, err := parseTotal(config.Default.Total)
		if err != nil {
			return nil, fmt.Errorf("webTimeout.default: %w", err)
		}
		compiled.defaultTotal = total
	}
	for i, rule := range config.Paths {
		total, err := parseTotal(rule.Total)
		if err != nil {
			return nil, fmt.Errorf("webTimeout.paths[%d]: %w", i, err)
		}
		pattern, err := compilePath(rule.Path)
		if err != nil {
			return nil, fmt.Errorf("webTimeout.paths[%d]: %w", i, err)
		}
		compiled.paths = append(compiled.paths, _Path{
			pattern:  pattern,
			total:    total,
			exact:    !strings.Contains(rule.Path, "*"),
			segments: strings.Split(rule.Path[1:], "/"),
		})
	}
	slices.SortStableFunc(compiled.paths, compareSpecificity)
	return compiled, nil
}

// compareSpecificity orders exact paths first, then compares segments from left
// to right. Ending a pattern is more specific than a trailing ** that can match
// zero segments, but less specific than another literal or single-segment glob.
func compareSpecificity(left _Path, right _Path) int {
	if left.exact != right.exact {
		if left.exact {
			return -1
		}
		return 1
	}
	for i := 0; i < max(len(left.segments), len(right.segments)); i++ {
		leftKind, leftFixed := segmentSpecificity(left.segments, i)
		rightKind, rightFixed := segmentSpecificity(right.segments, i)
		if order := cmp.Compare(rightKind, leftKind); order != 0 {
			return order
		}
		if order := cmp.Compare(rightFixed, leftFixed); order != 0 {
			return order
		}
	}
	return 0
}

func segmentSpecificity(segments []string, index int) (int, int) {
	if index >= len(segments) {
		return 1, 0
	}
	segment := segments[index]
	if segment == "**" {
		return 0, 0
	}
	if strings.Contains(segment, "*") {
		return 2, utf8.RuneCountInString(strings.ReplaceAll(segment, "*", ""))
	}
	return 3, 0
}

func parseTotal(value string) (time.Duration, error) {
	total, err := time.ParseDuration(value)
	if err != nil || total <= 0 {
		return 0, fmt.Errorf("total must be a positive duration")
	}
	return total, nil
}

func compilePath(value string) (*regexp.Regexp, error) {
	invalid := func() (*regexp.Regexp, error) {
		return nil, fmt.Errorf("invalid path pattern %q: use an absolute path with * within a segment or ** as a whole segment", value)
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#[]\\") || strings.ContainsFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) {
		return invalid()
	}
	segments := strings.Split(value[1:], "/")
	var expression strings.Builder
	expression.WriteString("^")
	for i, segment := range segments {
		if segment == "." || segment == ".." || (segment == "" && i < len(segments)-1) {
			return invalid()
		}
		if segment == "**" {
			if i == len(segments)-1 {
				if i == 0 {
					expression.WriteString("/.*")
				} else {
					expression.WriteString("(?:/.*)?")
				}
			} else {
				expression.WriteString("(?:/[^/]+)*")
			}
		} else {
			if strings.Contains(segment, "**") {
				return invalid()
			}
			expression.WriteByte('/')
			expression.WriteString(strings.ReplaceAll(regexp.QuoteMeta(segment), `\*`, "[^/]*"))
		}
	}
	expression.WriteString("$")
	return regexp.Compile(expression.String())
}

// Total selects the most specific match for the decoded, rewritten URL path.
// Equal specificity preserves configuration order. Zero means no explicit policy.
func (c *Compiled) Total(path string) time.Duration {
	if c == nil {
		return 0
	}
	for _, rule := range c.paths {
		if rule.pattern.MatchString(path) {
			return rule.total
		}
	}
	return c.defaultTotal
}
