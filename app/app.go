package app

import (
	ucli "github.com/urfave/cli/v3"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
)

// Option configures an application that connects to an independently running Link.
type Option struct {
	// LinkEndpoint is the Link API endpoint used by the application.
	LinkEndpoint string
}

// New constructs an application that connects to a Link configured by flags or environment.
func New[S ApplicationSpec](appliers ...FlagApplier) App {
	return NewWithOption[S](Option{}, appliers...)
}

// NewWithOption constructs an application that connects to the Link in option.
func NewWithOption[S ApplicationSpec](option Option, appliers ...FlagApplier) App {
	return internalapp.New[S](appAppliersWithCli(option, appliers)...)
}

// Helpers

const (
	flagAppLinkEndpoint = "link-endpoint"
	envAppLinkEndpoint  = "VINE_LINK_ENDPOINT"
)

func appAppliersWithCli(option Option, appliers []FlagApplier) []FlagApplier {
	cliOption := Option{}
	appcli.Handle(linkEndpointFlag(&cliOption.LinkEndpoint))
	applyOption(&cliOption, option)

	appliers = append([]FlagApplier(nil), appliers...)
	if cliOption.LinkEndpoint != "" {
		appliers = append(appliers, internalapp.WithLinkEndpoint(cliOption.LinkEndpoint))
	}
	return appliers
}

func applyOption(cliOption *Option, option Option) {
	if option.LinkEndpoint != "" {
		cliOption.LinkEndpoint = option.LinkEndpoint
	}
}

func linkEndpointFlag(destination *string) ucli.Flag {
	return &ucli.StringFlag{
		Name:        flagAppLinkEndpoint,
		Sources:     ucli.EnvVars(envAppLinkEndpoint),
		Usage:       "Link API endpoint",
		Destination: destination,
	}
}
