package linked

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/app"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	vinecli "go.yorun.ai/vine/internal/cli"
	"go.yorun.ai/vine/internal/core/logger"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

type _App struct {
	option Option

	link app.App
	apps []app.App
}

// Option configures the Hub connection and ingress listener for linked mode.
type Option struct {
	// HubEndpoint is the API endpoint of the external Hub.
	HubEndpoint string
	// IngressListen is the address on which the in-process Link accepts application traffic.
	IngressListen string
}

func (o Option) isZero() bool {
	return o.HubEndpoint == "" && o.IngressListen == ""
}

// New constructs an application and an in-process Link connected to an external Hub.
func New[S app.ApplicationSpec](appliers ...app.FlagApplier) app.App {
	return NewWithOption[S](Option{}, appliers...)
}

// NewWithOption constructs a linked application using option.
func NewWithOption[S app.ApplicationSpec](option Option, appliers ...app.FlagApplier) app.App {
	return &_App{
		apps:   []app.App{internalapp.NewInproc[S](appliers...)},
		option: option,
	}
}

// NewBundled combines linked applications so they share one in-process Link.
func NewBundled(apps ...app.App) app.App {
	return NewBundledWithOption(Option{}, apps...)
}

// NewBundledWithOption combines linked applications and configures their shared Link.
func NewBundledWithOption(option Option, apps ...app.App) app.App {
	vpre.Check(len(apps) > 0, "linked app expected")

	bundle := &_App{option: option}
	for _, application := range apps {
		linkedApp, ok := application.(*_App)
		vpre.Check(ok, "linked app expected")
		vpre.Check(linkedApp.option.isZero(), "bundled linked app must not have option")
		bundle.apps = append(bundle.apps, linkedApp.apps...)
	}
	return bundle
}

func (*_App) Name() string {
	return ""
}

func (a *_App) Start() {
	a.link = startLink(a.option)
	for _, application := range a.apps {
		application.Start()
		logger.Info("linked app started", "name", application.Name())
	}
}

func (a *_App) StopGracefully() {
	for i := range a.apps {
		application := a.apps[len(a.apps)-1-i]
		application.StopGracefully()
		logger.Info("linked app stopped", "name", application.Name())
	}
	a.link.StopGracefully()
	logger.Info("linked link stopped")
}

func (a *_App) StartAndWait() {
	a.Start()
	internalapp.WaitExitSignal()
	a.StopGracefully()
}

const (
	flagHubEndpoint   = vinecli.FlagLinkHubEndpoint
	flagIngressListen = vinecli.FlagLinkIngressListen
	envHubEndpoint    = vinecli.EnvLinkHubEndpoint
	envIngressListen  = vinecli.EnvLinkIngressListen
)

func startLink(option Option) app.App {
	flag := &linkflag.Flag{}
	appcli.Handle(flags(flag)...)
	applyOption(flag, option)

	link := internalapp.NewInternalInproc[*linkapp.LinkApp](internalapp.With(flag))
	link.Start()
	return link
}

func flags(flag *linkflag.Flag) []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{
			Name:        flagHubEndpoint,
			Sources:     ucli.EnvVars(envHubEndpoint),
			Usage:       "Hub API endpoint",
			Destination: &flag.HubEndpoint,
		},
		&ucli.StringFlag{
			Name:        flagIngressListen,
			Sources:     ucli.EnvVars(envIngressListen),
			Usage:       "link ingress listen address",
			Destination: &flag.IngressListen,
		},
	}
}

func applyOption(flag *linkflag.Flag, option Option) {
	if option.HubEndpoint != "" {
		flag.HubEndpoint = option.HubEndpoint
	}
	if option.IngressListen != "" {
		flag.IngressListen = option.IngressListen
	}
}
