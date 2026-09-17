package standalone

import (
	"go.yorun.ai/vine/app"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/appcli"
	"go.yorun.ai/vine/internal/core/logger"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/src/server/app"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkapp "go.yorun.ai/vine/internal/daemon/link/src/server/app"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	portalapp "go.yorun.ai/vine/internal/daemon/portal/src/server/app"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

type _App struct {
	option Option

	hub    app.App
	portal app.App
	link   app.App
	apps   []app.App
}

// New constructs an application with an in-process Hub, Portal, and Link.
func New[S app.ApplicationSpec](appliers ...app.FlagApplier) app.App {
	return NewWithOption[S](Option{}, appliers...)
}

// NewWithOption constructs a standalone application using option.
func NewWithOption[S app.ApplicationSpec](option Option, appliers ...app.FlagApplier) app.App {
	return &_App{
		option: option,
		apps:   []app.App{internalapp.NewInproc[S](appliers...)},
	}
}

// NewBundled combines standalone applications into one in-process runtime.
func NewBundled(apps ...app.App) app.App {
	return NewBundledWithOption(Option{}, apps...)
}

// NewBundledWithOption combines standalone applications and configures their shared runtime.
func NewBundledWithOption(option Option, apps ...app.App) app.App {
	vpre.Check(len(apps) > 0, "standalone app expected")

	bundle := &_App{option: option}
	for _, application := range apps {
		standaloneApp, ok := application.(*_App)
		vpre.Check(ok, "standalone app expected")
		vpre.Check(standaloneApp.option.isZero(), "bundled standalone app must not have option")
		bundle.apps = append(bundle.apps, standaloneApp.apps...)
	}
	return bundle
}

// Name reports the mode this bundle starts: the applications it holds keep their
// own names.
func (*_App) Name() string {
	return "standalone"
}

func (a *_App) Start() {
	a.initInfra()
	a.startInfra()
	for _, application := range a.apps {
		application.Start()
		logger.Info("standalone app started", "name", application.Name())
	}
}

func (a *_App) StopGracefully() {
	for i := range a.apps {
		application := a.apps[len(a.apps)-1-i]
		application.StopGracefully()
		logger.Info("standalone app stopped", "name", application.Name())
	}
	a.stopInfraGracefully()
}

func (a *_App) StartAndWait() {
	a.Start()
	internalapp.WaitExitSignal()
	a.StopGracefully()
}

func (a *_App) initInfra() {
	flag := &hubflag.Flag{}
	appcli.Handle(flags(flag, a.option.IgnoredFlags...)...)
	applyOption(flag, a.option)

	a.hub = internalapp.NewInternalInproc[*hubapp.HubApp](internalapp.With(flag))
	a.link = internalapp.NewInternalInproc[*linkapp.LinkApp](internalapp.With(&linkflag.Flag{
		HubInprocMode: true,
	}))
	a.portal = internalapp.NewInternalInproc[*portalapp.PortalApp](internalapp.With(&portalflag.Flag{
		HubInprocMode: true,
	}))
}

func (a *_App) startInfra() {
	a.hub.Start()
	logger.Info("standalone hub started")
	a.portal.Start()
	logger.Info("standalone portal started")
	a.link.Start()
	logger.Info("standalone link started")
}

func (a *_App) stopInfraGracefully() {
	a.link.StopGracefully()
	logger.Info("standalone link stopped")
	a.portal.StopGracefully()
	logger.Info("standalone portal stopped")
	a.hub.StopGracefully()
	logger.Info("standalone hub stopped")
}
