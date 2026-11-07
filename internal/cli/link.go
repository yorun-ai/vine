package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

const (
	commandLink      = "link"
	commandLinkServe = "serve"

	FlagLinkAPIListen     = "api-listen"
	FlagLinkIngressListen = "ingress-listen"
	FlagLinkHubEndpoint   = "hub-endpoint"

	EnvLinkAPIListen     = "VINE_LINK_API_LISTEN"
	EnvLinkIngressListen = "VINE_LINK_INGRESS_LISTEN"
	EnvLinkHubEndpoint   = "VINE_LINK_HUB_ENDPOINT"
	EnvLinkLogLevel      = "VINE_LINK_LOG_LEVEL"
	EnvLinkLogRules      = "VINE_LINK_LOG_RULES"
)

// startLinkApp is overridden in tests to assert parsed flags without starting the real app.
var startLinkApp = func(flags linkflag.Flag) {
	app.NewInternal[*linkapp.LinkApp](
		app.With(&flags),
	).StartAndWait()
}

func newLinkCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandLink,
		Usage:              "sidecar mesh",
		Suggest:            true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newLinkServeCommand(),
		},
	}
}

func newLinkServeFlags() []ucli.Flag {
	return append([]ucli.Flag{
		&ucli.StringFlag{Name: FlagLinkAPIListen, Sources: ucli.EnvVars(EnvLinkAPIListen), Value: linkflag.LinkDefaultAPIListen, Usage: "link API listen address"},
		&ucli.StringFlag{Name: FlagLinkIngressListen, Sources: ucli.EnvVars(EnvLinkIngressListen), Value: linkflag.LinkDefaultIngressListen, Usage: "link ingress listen address"},
		&ucli.StringFlag{Name: FlagLinkHubEndpoint, Sources: ucli.EnvVars(EnvLinkHubEndpoint), Usage: "hub API endpoint"},
	}, mtlsFlags(commandLink)...)
}

func newLinkServeCommand() *ucli.Command {
	logFlags, applyLog := appcli.LoggingFlags(EnvLinkLogLevel, EnvLinkLogRules)
	return &ucli.Command{
		Name:  commandLinkServe,
		Usage: "start the link service",
		Flags: append(newLinkServeFlags(), logFlags...),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if err := applyLog(); err != nil {
				return err
			}
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandLinkServe)
			}

			flags := linkflag.Flag{
				APIListen:     cmd.String(FlagLinkAPIListen),
				IngressListen: cmd.String(FlagLinkIngressListen),
				HubEndpoint:   cmd.String(FlagLinkHubEndpoint),
				MTLS:          mtlsFiles(cmd),
			}
			startLinkApp(flags)
			return nil
		},
	}
}
