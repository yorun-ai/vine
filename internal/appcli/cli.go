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
	flagLogLevel          = "log-level"
	flagRpcServerLogLevel = "rpc-server-log-level"
	flagTaskLogLevel      = "task-log-level"
	flagEventLogLevel     = "event-log-level"
	flagAppLogLevel       = "app-log-level"
	flagAppScopeLogLevel  = "app-scope-log-level"
	envLogLevel           = "VINE_LOG_LEVEL"
	envRpcServerLogLevel  = "VINE_RPC_SERVER_LOG_LEVEL"
	envTaskLogLevel       = "VINE_TASK_LOG_LEVEL"
	envEventLogLevel      = "VINE_EVENT_LOG_LEVEL"
	envAppLogLevels       = "VINE_APP_LOG_LEVELS"
	envAppScopeLogLevels  = "VINE_APP_SCOPE_LOG_LEVELS"
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
	var rpcServerLogLevel string
	var taskLogLevel string
	var eventLogLevel string
	var appLogLevels []string
	var appScopeLogLevels []string
	flags = append([]ucli.Flag{
		new(ucli.StringFlag{
			Name:        flagLogLevel,
			Sources:     ucli.EnvVars(envLogLevel),
			Usage:       "log level: DEBUG, INFO, WARN, ERROR",
			Destination: &logLevel,
		}),
		new(ucli.StringFlag{
			Name:        flagRpcServerLogLevel,
			Sources:     ucli.EnvVars(envRpcServerLogLevel),
			Usage:       "Rpc server lifecycle log level",
			Destination: &rpcServerLogLevel,
		}),
		new(ucli.StringFlag{
			Name:        flagTaskLogLevel,
			Sources:     ucli.EnvVars(envTaskLogLevel),
			Usage:       "Task lifecycle log level",
			Destination: &taskLogLevel,
		}),
		new(ucli.StringFlag{
			Name:        flagEventLogLevel,
			Sources:     ucli.EnvVars(envEventLogLevel),
			Usage:       "Event lifecycle log level",
			Destination: &eventLogLevel,
		}),
		new(ucli.StringSliceFlag{
			Name:        flagAppLogLevel,
			Sources:     ucli.EnvVars(envAppLogLevels),
			Usage:       "App log level override: app=LEVEL",
			Destination: &appLogLevels,
		}),
		new(ucli.StringSliceFlag{
			Name:        flagAppScopeLogLevel,
			Sources:     ucli.EnvVars(envAppScopeLogLevels),
			Usage:       "App subsystem log level override: app:subsystem=LEVEL",
			Destination: &appScopeLogLevels,
		}),
	}, flags...)

	return &ucli.Command{
		Name:            commandName,
		Usage:           "application runtime options",
		HideHelp:        true,
		HideHelpCommand: true,
		Flags:           flags,
		Action: func(_ context.Context, cmd *ucli.Command) error {
			overrides, hasOverrides, err := parseLogLevelOverrides(
				rpcServerLogLevel, taskLogLevel, eventLogLevel, appLogLevels, appScopeLogLevels)
			if err != nil {
				return err
			}
			if logLevel != "" {
				level := logger.Level(logLevel)
				if !logger.IsValidLevel(level) {
					return fmt.Errorf("invalid log level %q", logLevel)
				}
				logger.SetGlobalLevel(level)
			}
			if hasOverrides {
				logger.ReplaceLevelOverrides(overrides)
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

func parseLogLevelOverrides(
	rpcServerLevel string,
	taskLevel string,
	eventLevel string,
	appRules []string,
	appScopeRules []string,
) (logger.LevelOverrides, bool, error) {
	overrides := logger.LevelOverrides{
		Subsystems: map[logger.Subsystem]logger.Level{},
		Apps:       map[string]logger.Level{},
	}
	hasOverrides := false
	for subsystem, rawLevel := range map[logger.Subsystem]string{
		logger.SubsystemRpcServer: rpcServerLevel,
		logger.SubsystemTask:      taskLevel,
		logger.SubsystemEvent:     eventLevel,
	} {
		if rawLevel == "" {
			continue
		}
		level := logger.Level(rawLevel)
		if !logger.IsValidLevel(level) {
			return logger.LevelOverrides{}, false, fmt.Errorf("invalid %s log level %q", subsystem, rawLevel)
		}
		overrides.Subsystems[subsystem] = level
		hasOverrides = true
	}
	for _, rule := range appRules {
		appName, level, err := parseAppLevelRule(rule)
		if err != nil {
			return logger.LevelOverrides{}, false, err
		}
		overrides.Apps[appName] = level
		hasOverrides = true
	}
	appScopeIndex := map[string]int{}
	for _, rule := range appScopeRules {
		appName, subsystem, level, err := parseAppScopeLevelRule(rule)
		if err != nil {
			return logger.LevelOverrides{}, false, err
		}
		key := appName + "\x00" + string(subsystem)
		entry := logger.AppSubsystemLevel{AppName: appName, Subsystem: subsystem, Level: level}
		if index, exists := appScopeIndex[key]; exists {
			overrides.AppSubsystem[index] = entry
		} else {
			appScopeIndex[key] = len(overrides.AppSubsystem)
			overrides.AppSubsystem = append(overrides.AppSubsystem, entry)
		}
		hasOverrides = true
	}
	return overrides, hasOverrides, nil
}

func parseAppLevelRule(rule string) (string, logger.Level, error) {
	separator := strings.LastIndexByte(rule, '=')
	if separator <= 0 || separator == len(rule)-1 {
		return "", "", fmt.Errorf("invalid app log level rule %q", rule)
	}
	appName := rule[:separator]
	level := logger.Level(rule[separator+1:])
	if !logger.IsValidLevel(level) {
		return "", "", fmt.Errorf("invalid app log level rule %q", rule)
	}
	return appName, level, nil
}

func parseAppScopeLevelRule(rule string) (string, logger.Subsystem, logger.Level, error) {
	separator := strings.LastIndexByte(rule, '=')
	if separator <= 0 || separator == len(rule)-1 {
		return "", "", "", fmt.Errorf("invalid app scope log level rule %q", rule)
	}
	selector := rule[:separator]
	scopeSeparator := strings.LastIndexByte(selector, ':')
	if scopeSeparator <= 0 || scopeSeparator == len(selector)-1 {
		return "", "", "", fmt.Errorf("invalid app scope log level rule %q", rule)
	}
	appName := selector[:scopeSeparator]
	subsystem := logger.Subsystem(selector[scopeSeparator+1:])
	level := logger.Level(rule[separator+1:])
	if !logger.IsValidSubsystem(subsystem) || !logger.IsValidLevel(level) {
		return "", "", "", fmt.Errorf("invalid app scope log level rule %q", rule)
	}
	return appName, subsystem, level, nil
}

func isIgnorableArgsError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "flag provided but not defined:")
}
