package logger

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"go.yorun.ai/vine/util/vpre"
)

type _Rule struct {
	pattern  string
	segments []string
	level    Level
}

func newRule(pattern string, level Level) (_Rule, error) {
	segments, err := parseRulePattern(pattern)
	if err != nil {
		return _Rule{}, err
	}
	if !IsValidLevel(level) {
		return _Rule{}, fmt.Errorf("%+v is not a valid LogLevel", level)
	}
	return _Rule{pattern: pattern, segments: segments, level: level}, nil
}

func parseRulePattern(pattern string) ([]string, error) {
	if pattern == "" || pattern == "*" {
		return nil, fmt.Errorf("%q is not a valid logger rule pattern", pattern)
	}

	segments := strings.Split(pattern, ":")
	for _, segment := range segments {
		if segment == "" || (strings.Contains(segment, "*") && segment != "*" && segment != "**") {
			return nil, fmt.Errorf("%q is not a valid logger rule pattern", pattern)
		}
	}
	return segments, nil
}

func (r _Rule) matches(name []string) bool {
	var match func(patternIndex int, nameIndex int) bool
	match = func(patternIndex int, nameIndex int) bool {
		if patternIndex == len(r.segments) {
			return true
		}

		switch r.segments[patternIndex] {
		case "**":
			for next := nameIndex; next <= len(name); next++ {
				if match(patternIndex+1, next) {
					return true
				}
			}
			return false
		case "*":
			if nameIndex == len(name) {
				return false
			}
			return match(patternIndex+1, nameIndex+1)
		default:
			if nameIndex == len(name) {
				return false
			}
			return r.segments[patternIndex] == name[nameIndex] &&
				match(patternIndex+1, nameIndex+1)
		}
	}
	return match(0, 0)
}

func (r _Rule) moreSpecificThan(other _Rule) bool {
	weight := func(segments []string, index int) int {
		if index >= len(segments) {
			return 0
		}
		switch segments[index] {
		case "**":
			return 1
		case "*":
			return 2
		default:
			return 3
		}
	}

	for index := range max(len(r.segments), len(other.segments)) {
		left, right := weight(r.segments, index), weight(other.segments, index)
		if left != right {
			return left > right
		}
	}
	return r.pattern > other.pattern
}

type _Rules struct {
	byPattern map[string]Level
	ordered   []_Rule
}

func newRules(byPattern map[string]Level) (*_Rules, error) {
	rules := &_Rules{
		byPattern: make(map[string]Level, len(byPattern)),
		ordered:   make([]_Rule, 0, len(byPattern)),
	}
	for pattern, level := range byPattern {
		rule, err := newRule(pattern, level)
		if err != nil {
			return nil, err
		}
		rules.byPattern[pattern] = level
		rules.ordered = append(rules.ordered, rule)
	}
	sort.Slice(rules.ordered, func(i, j int) bool {
		return rules.ordered[i].moreSpecificThan(rules.ordered[j])
	})
	return rules, nil
}

func (r *_Rules) resolve(name []string) (Level, bool) {
	for _, rule := range r.ordered {
		if rule.matches(name) {
			return rule.level, true
		}
	}
	return "", false
}

var rules = func() *atomic.Pointer[_Rules] {
	store := new(atomic.Pointer[_Rules])
	initial, err := newRules(map[string]Level{"**": LevelInfo})
	if err != nil {
		panic(err)
	}
	store.Store(initial)
	return store
}()

func SetLevel(pattern string, level Level) {
	_, err := newRule(pattern, level)
	vpre.Check(err == nil, "%v", err)
	updateRule(pattern, &level)
}

func ClearLevel(pattern string) {
	vpre.Check(pattern != "**", "the default logger level cannot be cleared")
	_, err := parseRulePattern(pattern)
	vpre.Check(err == nil, "%v", err)
	updateRule(pattern, nil)
}

func Levels() map[string]Level {
	current := rules.Load()
	levels := make(map[string]Level, len(current.byPattern))
	for pattern, level := range current.byPattern {
		levels[pattern] = level
	}
	return levels
}

func updateRule(pattern string, level *Level) {
	for {
		current := rules.Load()
		byPattern := make(map[string]Level, len(current.byPattern)+1)
		for existingPattern, existingLevel := range current.byPattern {
			byPattern[existingPattern] = existingLevel
		}
		if level == nil {
			delete(byPattern, pattern)
		} else {
			byPattern[pattern] = *level
		}
		next, err := newRules(byPattern)
		vpre.Check(err == nil, "%v", err)
		if rules.CompareAndSwap(current, next) {
			return
		}
	}
}
