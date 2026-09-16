package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/app"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

const (
	commandLink      = "link"
	commandLinkServe = "serve"

	FlagLinkAPIListen     = "api-listen"
	FlagLinkIngressListen = "ingress-listen"
	FlagLinkHubEndpoint   = "hub-endpoint"

	EnvLinkAPIListen     = "VINE_API_LISTEN"
	EnvLinkIngressListen = "VINE_INGRESS_LISTEN"
	EnvLinkHubEndpoint   = "VINE_HUB_ENDPOINT"

	// Linked has no command to scope these, so they carry the prefix.
	FlagLinkedHubEndpoint   = "link-hub-endpoint"
	FlagLinkedIngressListen = "link-ingress-listen"
	FlagLinkedMTLSCAFile    = "link-mtls-ca-file"
	FlagLinkedMTLSCertFile  = "link-mtls-cert-file"
	FlagLinkedMTLSKeyFile   = "link-mtls-key-file"

	EnvLinkedHubEndpoint   = "VINE_LINK_HUB_ENDPOINT"
	EnvLinkedIngressListen = "VINE_LINK_INGRESS_LISTEN"
	EnvLinkedMTLSCAFile    = "VINE_LINK_MTLS_CA_FILE"
	EnvLinkedMTLSCertFile  = "VINE_LINK_MTLS_CERT_FILE"
	EnvLinkedMTLSKeyFile   = "VINE_LINK_MTLS_KEY_FILE"
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
	}, mtlsFlags()...)
}

func newLinkServeCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandLinkServe,
		Usage: "start the link service",
		Flags: newLinkServeFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
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
