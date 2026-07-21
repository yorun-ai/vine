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
	envLogLevel  = "VINE_LOG_LEVEL"
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
	flags = append([]ucli.Flag{
		&ucli.StringFlag{
			Name:        flagLogLevel,
			Sources:     ucli.EnvVars(envLogLevel),
			Usage:       "log level: DEBUG, INFO, WARN, ERROR",
			Destination: &logLevel,
		},
	}, flags...)

	return &ucli.Command{
		Name:            commandName,
		Usage:           "application runtime options",
		HideHelp:        true,
		HideHelpCommand: true,
		Flags:           flags,
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if logLevel != "" {
				logger.SetGlobalLevel(logger.Level(logLevel))
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

func isIgnorableArgsError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "flag provided but not defined:")
}
