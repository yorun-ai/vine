package appcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/internal/core/runtime"
)

var argsStdout io.Writer = os.Stdout
var argsStderr io.Writer = os.Stderr
var argsExit = os.Exit

var errIgnoreArgs = errors.New("ignore app args")
var helpFlagMu sync.Mutex

const (
	flagLogLevel = "log-level"
	flagLogRule  = "log-rule"
	envLogLevel  = "VINE_LOG_LEVEL"
	envLogRules  = "VINE_LOG_RULES"
)

// Handle parses common application arguments together with flags.
func Handle(flags ...ucli.Flag) {
	shouldExit, err := parseArgs(os.Args, flags...)
	if err != nil {
		if errors.Is(err, errIgnoreArgs) {
			return
		}
		_, _ = fmt.Fprint(argsStderr, err.Error())
		argsExit(1)
		return
	}
	if shouldExit {
		argsExit(0)
	}
}

func parseArgs(args []string, flags ...ucli.Flag) (bool, error) {
	shouldExit := false

	command := newArgsCommand(args, func() {
		shouldExit = true
	}, flags...)
	command.Writer = argsStdout
	command.ErrWriter = argsStderr
	command.ExitErrHandler = func(_ context.Context, _ *ucli.Command, _ error) {}
	command.OnUsageError = func(_ context.Context, _ *ucli.Command, err error, _ bool) error {
		if isIgnorableArgsError(err) {
			return errIgnoreArgs
		}
		return err
	}

	err := runArgsCommand(command, args)
	return shouldExit, err
}

func runArgsCommand(command *ucli.Command, args []string) error {
	helpFlagMu.Lock()
	prevHelpFlag := ucli.HelpFlag
	ucli.HelpFlag = nil
	defer func() {
		ucli.HelpFlag = prevHelpFlag
		helpFlagMu.Unlock()
	}()

	return command.Run(context.Background(), args)
}

func newArgsCommand(args []string, setShouldExit func(), flags ...ucli.Flag) *ucli.Command {
	commandName := "app"
	if len(args) > 0 {
		commandName = filepath.Base(args[0])
	}

	var logLevel string
	var logRules []string
	flags = append([]ucli.Flag{
		new(ucli.StringFlag{
			Name:        flagLogLevel,
			Sources:     ucli.EnvVars(envLogLevel),
			Usage:       "log level: DEBUG, INFO, WARN, ERROR",
			Destination: &logLevel,
		}),
		new(ucli.StringSliceFlag{
			Name:        flagLogRule,
			Sources:     ucli.EnvVars(envLogRules),
			Usage:       "named log rule: pattern=LEVEL",
			Destination: &logRules,
		}),
	}, flags...)

	return &ucli.Command{
		Name:            commandName,
		Usage:           "application runtime options",
		HideHelp:        true,
		HideHelpCommand: true,
		Flags:           flags,
		Action: func(_ context.Context, cmd *ucli.Command) error {
			levels, hasLevels, err := parseRules(logRules)
			if err != nil {
				return err
			}
			var parsedLogLevel logger.Level
			if logLevel != "" {
				parsedLogLevel = logger.Level(logLevel)
				if !logger.IsValidLevel(parsedLogLevel) {
					return fmt.Errorf("invalid log level %q", logLevel)
				}
			}
			if hasLevels {
				for pattern, level := range levels {
					logger.SetLevel(pattern, level)
				}
			}
			if logLevel != "" {
				logger.SetGlobalLevel(parsedLogLevel)
			}

			arg := cmd.Args().First()
			if arg == "" {
				return nil
			}

			switch arg {
			case "version":
				setShouldExit()
				_, _ = fmt.Fprint(cmd.Root().Writer, runtime.Inspect())
				return nil
			case "help":
				setShouldExit()
				return ucli.ShowSubcommandHelp(cmd)
			default:
				return nil
			}
		},
	}
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
	separator := strings.LastIndexByte(rule, '=')
	if separator <= 0 || separator == len(rule)-1 {
		return "", "", fmt.Errorf("invalid log rule %q", rule)
	}
	pattern := rule[:separator]
	level := logger.Level(rule[separator+1:])
	if !isValidRulePattern(pattern) || !logger.IsValidLevel(level) {
		return "", "", fmt.Errorf("invalid log rule %q", rule)
	}
	return pattern, level, nil
}

func isValidRulePattern(pattern string) bool {
	if pattern == "" || pattern == "*" || pattern == "**" {
		return false
	}
	for _, segment := range strings.Split(pattern, ":") {
		if segment == "" || (strings.Contains(segment, "*") && segment != "*" && segment != "**") {
			return false
		}
	}
	return true
}

func isIgnorableArgsError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "flag provided but not defined:")
}
