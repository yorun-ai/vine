package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

const (
	commandPortal      = "portal"
	commandPortalServe = "serve"

	flagPortalHubEndpoint = "hub-endpoint"

	envPortalHubEndpoint = "VINE_PORTAL_HUB_ENDPOINT"
	EnvPortalLogLevel    = "VINE_PORTAL_LOG_LEVEL"
	EnvPortalLogRules    = "VINE_PORTAL_LOG_RULES"
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
	return append([]ucli.Flag{
		&ucli.StringFlag{Name: flagPortalHubEndpoint, Sources: ucli.EnvVars(envPortalHubEndpoint), Usage: "hub API endpoint"},
	}, mtlsFlags(commandPortal)...)
}

func newPortalServeCommand() *ucli.Command {
	logFlags, applyLog := appcli.LoggingFlags(EnvPortalLogLevel, EnvPortalLogRules)
	return &ucli.Command{
		Name:  commandPortalServe,
		Usage: "start the portal service",
		Flags: append(newPortalServeFlags(), logFlags...),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if err := applyLog(); err != nil {
				return err
			}
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected args for %s", commandPortalServe)
			}

			flags := portalflag.Flag{
				HubEndpoint: cmd.String(flagPortalHubEndpoint),
				MTLS:        mtlsFiles(cmd),
			}
			startPortalApp(flags)
			return nil
		},
	}
}
