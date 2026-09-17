package appcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/buildinfo"
)

var argsStdout io.Writer = os.Stdout
var argsStderr io.Writer = os.Stderr
var argsExit = os.Exit

var helpFlagMu sync.Mutex

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

	err := runArgsCommand(command, dropUnknownArgs(args, flags...))
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

	logFlags := new(_LogFlags)
	flags = append(logFlags.flags(), flags...)

	return &ucli.Command{
		Name:            commandName,
		Usage:           "application runtime options",
		HideHelp:        true,
		HideHelpCommand: true,
		Flags:           flags,
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if err := logFlags.apply(); err != nil {
				return err
			}

			arg := cmd.Args().First()
			if arg == "" {
				return nil
			}

			switch arg {
			case "version":
				setShouldExit()
				_, _ = fmt.Fprint(cmd.Root().Writer, buildinfo.Inspect())
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
