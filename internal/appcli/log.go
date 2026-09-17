package appcli

import (
	"fmt"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/core/logger"
)

const (
	flagLogLevel = "log-level"
	flagLogRule  = "log-rule"
	envLogLevel  = "VINE_LOG_LEVEL"
	envLogRules  = "VINE_LOG_RULES"
)

// _LogFlags carries the logging flags every application accepts and applies the
// parsed values to the process logger.
type _LogFlags struct {
	level string
	rules []string
}

func (l *_LogFlags) flags() []ucli.Flag {
	return []ucli.Flag{
		new(ucli.StringFlag{
			Name:        flagLogLevel,
			Sources:     ucli.EnvVars(envLogLevel),
			Usage:       "log level: DEBUG, INFO, WARN, ERROR",
			Destination: &l.level,
		}),
		new(ucli.StringSliceFlag{
			Name:        flagLogRule,
			Sources:     ucli.EnvVars(envLogRules),
			Usage:       "named log rule: pattern=LEVEL",
			Destination: &l.rules,
		}),
	}
}

func (l *_LogFlags) apply() error {
	levels, hasLevels, err := parseRules(l.rules)
	if err != nil {
		return err
	}

	var parsedLevel logger.Level
	if l.level != "" {
		parsedLevel = logger.Level(l.level)
		if !logger.IsValidLevel(parsedLevel) {
			return fmt.Errorf("invalid log level %q", l.level)
		}
	}
	if hasLevels {
		for pattern, level := range levels {
			logger.SetLevel(pattern, level)
		}
	}
	if l.level != "" {
		logger.SetGlobalLevel(parsedLevel)
	}
	return nil
}

func parseRules(rules []string) (map[string]logger.Level, bool, error) {
	levels := make(map[string]logger.Level, len(rules))
	for _, rule := range rules {
		pattern, level, err := parseRule(rule)
		if err != nil {
			return nil, false, err
		}
		levels[pattern] = level
	}
	return levels, len(rules) > 0, nil
}

func parseRule(rule string) (string, logger.Level, error) {
	pattern, levelValue, ok := strings.CutLast(rule, "=")
	if !ok || pattern == "" || levelValue == "" {
		return "", "", fmt.Errorf("invalid log rule %q", rule)
	}
	level := logger.Level(levelValue)
	if !isValidRulePattern(pattern) || !logger.IsValidLevel(level) {
		return "", "", fmt.Errorf("invalid log rule %q", rule)
	}
	return pattern, level, nil
}

func isValidRulePattern(pattern string) bool {
	if pattern == "" || pattern == "*" || pattern == "**" {
		return false
	}
	for segment := range strings.SplitSeq(pattern, ":") {
		if segment == "" || (strings.Contains(segment, "*") && segment != "*" && segment != "**") {
			return false
		}
	}
	return true
}
