package app

import (
	ucli "github.com/urfave/cli/v3"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	"go.yorun.ai/vine/util/vpre"
)

type _App struct {
	apps []App
}

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
	return &_App{
		apps: []App{internalapp.New[S](appAppliersWithCli(option, appliers)...)},
	}
}

// NewBundled combines applications created by New or NewWithOption into one
// lifecycle. It does not start a Link; each application retains its configured
// external Link endpoint.
func NewBundled(apps ...App) App {
	vpre.Check(len(apps) > 0, "app.New app expected")

	bundle := &_App{}
	for _, application := range apps {
		plainApp, ok := application.(*_App)
		vpre.Check(ok, "app.New app expected")
		bundle.apps = append(bundle.apps, plainApp.apps...)
	}
	return bundle
}

func (a *_App) Name() string {
	if len(a.apps) == 1 {
		return a.apps[0].Name()
	}
	return ""
}

func (a *_App) Start() {
	for _, application := range a.apps {
		application.Start()
	}
}

func (a *_App) StopGracefully() {
	for i := range a.apps {
		a.apps[len(a.apps)-1-i].StopGracefully()
	}
}

func (a *_App) StartAndWait() {
	a.Start()
	internalapp.WaitExitSignal()
	a.StopGracefully()
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
