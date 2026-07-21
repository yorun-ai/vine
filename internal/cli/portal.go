package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

const (
	commandPortal      = "portal"
	commandPortalServe = "serve"

	flagPortalHubEndpoint = "hub-endpoint"

	envPortalHubEndpoint = "VINE_HUB_ENDPOINT"
)

// startPortalApp is overridden in tests to assert parsed flags without starting the real app.
var startPortalApp = func(flags portalflag.Flag) {
	app.NewInternal[*portalapp.PortalApp](
		app.With(&flags),
	).StartAndWait()
}

func newPortalCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandPortal,
		Usage:              "application gateway",
		Suggest:            true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newPortalServeCommand(),
		},
	}
}

func newPortalServeFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagPortalHubEndpoint, Sources: ucli.EnvVars(envPortalHubEndpoint), Usage: "hub API endpoint"},
	}
}

func newPortalServeCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandPortalServe,
		Usage: "start the portal service",
		Flags: newPortalServeFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandPortalServe)
			}

			flags := portalflag.Flag{
				HubEndpoint: cmd.String(flagPortalHubEndpoint),
			}
			startPortalApp(flags)
			return nil
		},
	}
}
