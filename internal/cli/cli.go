package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
)

const (
	commandVine = "vine"

	exitCodeSuccess = 0
	exitCodeError   = 1
)

type _RunResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func Main() {
	result := run(os.Args[1:])
	if result.stdout != "" {
		_, _ = fmt.Fprint(os.Stdout, result.stdout)
	}
	if result.stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, result.stderr)
		if !strings.HasSuffix(result.stderr, "\n") {
			_, _ = fmt.Fprintln(os.Stderr)
		}
	}
	os.Exit(result.exitCode)
}

func run(args []string) _RunResult {
	return runCLICommand(newVineCommand(), append([]string{"vine"}, args...))
}

func runCLICommand(command *ucli.Command, args []string) (result _RunResult) {
	defer recoverAsErrorResult(&result)

	var stdout strings.Builder
	var stderr strings.Builder

	command.Writer = &stdout
	command.ErrWriter = &stderr
	command.ExitErrHandler = func(_ context.Context, _ *ucli.Command, _ error) {}

	err := command.Run(context.Background(), args)
	if err != nil {
		if stderr.Len() > 0 {
			return _RunResult{
				exitCode: exitCodeError,
				stdout:   stdout.String(),
				stderr:   stderr.String(),
			}
		}
		return _RunResult{exitCode: exitCodeError, stdout: stdout.String(), stderr: err.Error()}
	}
	if stderr.Len() > 0 {
		return _RunResult{exitCode: exitCodeError, stdout: stdout.String(), stderr: stderr.String()}
	}
	return _RunResult{exitCode: exitCodeSuccess, stdout: stdout.String(), stderr: stderr.String()}
}

func newVineCommand() *ucli.Command {
	return &ucli.Command{
		Name:            commandVine,
		Usage:           "vine runtime tools",
		Suggest:         true,
		HideHelpCommand: true,
		Commands: []*ucli.Command{
			newVersionCommand(),
			newHubCommand(),
			newLinkCommand(),
			newPortalCommand(),
		},
	}
}

func newVersionCommand() *ucli.Command {
	return &ucli.Command{
		Name: commandVersion,
		Flags: []ucli.Flag{
			&ucli.BoolFlag{Name: flagVersionJSON, Usage: "output version info as JSON"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			args := commandPositionalArgs(cmd)
			if len(args) > 0 {
				return fmt.Errorf("too many arguments")
			}
			info := versionInfo()
			if cmd.Bool(flagVersionJSON) {
				_, _ = fmt.Fprintln(cmd.Root().Writer, info.JSONString())
				return nil
			}
			_, _ = fmt.Fprintln(cmd.Root().Writer, info.TextString())
			return nil
		},
	}
}
