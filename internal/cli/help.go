package cli

import (
	"fmt"

	ucli "github.com/urfave/cli/v3"
)

const groupCommandHelpTemplate = `NAME:
   {{template "helpNameTemplate" .}}

USAGE:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}}{{if .VisibleCommands}} [command [command options]]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Arguments}} [arguments...]{{end}}{{end}}{{end}}{{if .Category}}

CATEGORY:
   {{.Category}}{{end}}{{if .Description}}

DESCRIPTION:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{template "visibleCommandTemplate" .}}{{end}}
{{- range .VisibleCommands}}{{if .VisibleFlags}}

{{.Name}} OPTIONS:{{template "visibleFlagTemplate" .}}{{end}}{{end}}
{{- if .VisibleFlags}}

OPTIONS:{{template "visibleFlagTemplate" .}}{{end}}
`

func init() {
	if helpFlag, ok := ucli.HelpFlag.(*ucli.BoolFlag); ok {
		helpFlag.Hidden = true
	}

	ucli.ShowSubcommandHelp = func(cmd *ucli.Command) error {
		tpl := cmd.CustomHelpTemplate
		if tpl == "" {
			tpl = ucli.SubcommandHelpTemplate
		}
		ucli.HelpPrinter(cmd.Root().Writer, tpl, cmd)
		return nil
	}
}

func errorResult(format string, args ...any) _RunResult {
	return _RunResult{
		exitCode: exitCodeError,
		stderr:   fmt.Sprintf(format, args...),
	}
}

func recoverAsErrorResult(result *_RunResult) {
	if recovered := recover(); recovered != nil {
		*result = errorResult("Error: %s", recovered)
	}
}

func commandPositionalArgs(cmd *ucli.Command) []string {
	args := make([]string, 0, cmd.Args().Len())
	for index := 0; index < cmd.Args().Len(); index++ {
		args = append(args, cmd.Args().Get(index))
	}
	return args
}
