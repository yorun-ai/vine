package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	coreapp "go.yorun.ai/vine/internal/core/app"

	"go.yorun.ai/vine/core/skel"
	"go.yorun.ai/vine/internal/core/conf"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/link"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/server"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/core/runtime"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

// unregisterTimeout must exceed Link-side unregister drain waiting, otherwise app shutdown
// may fail early on the client side while Link is still draining inflight work.
const unregisterTimeout = time.Minute

type _AppImpl struct {
	spec   ApplicationSpec
	info   runtime.App
	ctx    context.Context
	cancel context.CancelFunc
	flags  _Flags

	listenAddr string
	linker     link.Linker
	reader     conf.Reader

	inprocFlag *InternalInprocFlag

	injector                  di.PlainInjector
	frameworkComponentMinders []FrameworkComponentMinder
	components                []Component
	componentLifecycles       []ComponentLifecycle
	modules                   []Module

	httpServer *http.Server
	httpWG     sync.WaitGroup
	httpHost   string
	httpPort   int
	doneSignal chan struct{}

	lifecycleState _AppLifecycleState

	webber   *_Webber
	servicer *_Servicer
	eventer  *_Eventer
	tasker   *_Tasker

	consoleServer *server.Server
	routes        []_ServerRoute
}

type _AppLifecycleState int

const (
	appLifecycleStateNew _AppLifecycleState = iota
	appLifecycleStateStarted
	appLifecycleStateStopping
	appLifecycleStateStopped
)

var detectHostIP = vnet.DetectHostIP

func newApp(spec ApplicationSpec, flags _Flags) *_AppImpl {
	ctx, cancel := context.WithCancel(flags.Context())
	app := &_AppImpl{
		spec:   spec,
		ctx:    ctx,
		cancel: cancel,
		flags:  flags,
	}

	app.init()
	return app
}

func (a *_AppImpl) init() {
	vpre.CheckNotEmpty(a.spec.Name(), "application name must not be empty")
	vpre.Check(!strings.Contains(a.spec.Name(), "@"), "application name must not contain @")

	a.inprocFlag = a.flags.InprocFlag()
	if !a.initByInternalAttrs() {
		a.info = a.newAppInfo()
		a.inprocFlag.HostPath = coreapp.InprocHostPath(a.info.InstanceId())
	}

	a.listenAddr = a.flags.ListenAddr()
	a.doneSignal = make(chan struct{})
}

func (a *_AppImpl) isInternalApplication() bool {
	_, ok := a.spec.(InternalApplicationSpec)
	return ok
}

func (a *_AppImpl) initByInternalAttrs() bool {
	if !a.isInternalApplication() {
		return false
	}

	internalAttrs := a.spec.(InternalApplicationSpec).internalAttrs()
	vpre.CheckNotNil(internalAttrs.Info, "internal application info must not be nil")
	vpre.CheckNotNil(internalAttrs.Linker, "internal application linker must not be nil")
	a.info = internalAttrs.Info
	a.linker = internalAttrs.Linker
	a.inprocFlag.HostPath = internalAttrs.InprocHostPath

	return true
}

func (a *_AppImpl) newAppInfo() runtime.App {
	runtimeApp := runtime.Application()
	if !a.inprocFlag.Enabled && a.spec.Name() == runtimeApp.Name() {
		return runtimeApp
	}
	return meta.MustNewAppWithRandomId(a.spec.Name()+"@"+runtimeApp.Name(), runtimeApp.Version())
}

func (a *_AppImpl) initLinking() {
	if !a.isInternalApplication() {
		a.linker = link.NewLinker(a.info, a.inprocFlag.Enabled, a.flags.LinkEndpoint())
	}
	a.reader = conf.NewReader(a.linker)
}

func (a *_AppImpl) Name() string {
	return a.spec.Name()
}

func (a *_AppImpl) Start() {
	vpre.Check(a.lifecycleState != appLifecycleStateStarted, "application already started")
	vpre.Check(a.lifecycleState == appLifecycleStateNew, "application already stopped")
	a.lifecycleState = appLifecycleStateStarted

	a.initLinking()
	a.initInjector()
	a.initComponents()
	a.initModules()
	a.initServers()

	err := a.beforeAppStart()
	vpre.Check(err == nil, "application before start failed: %v", err)
	a.startServers()

	if a.servicer != nil {
		a.servicer.start()
	}
	if a.eventer != nil {
		a.eventer.start()
	}
	if a.tasker != nil {
		a.tasker.start()
	}

	a.registerApp()
	a.afterAppStart()
}

func (a *_AppImpl) StopGracefully() {
	vpre.Check(a.lifecycleState != appLifecycleStateNew, "application is not started")
	vpre.Check(a.lifecycleState == appLifecycleStateStarted, "application already stopped")
	a.lifecycleState = appLifecycleStateStopping

	go func() {
		a.beforeAppStop()
		a.unregisterApp()
		a.stopServers()
		a.cancel()
		a.afterAppStop()
		close(a.doneSignal)
	}()
	<-a.doneSignal

	a.lifecycleState = appLifecycleStateStopped
}

func (a *_AppImpl) StartAndWait() {
	a.Start()
	WaitExitSignal()
	a.StopGracefully()
}

func (a *_AppImpl) startServers() {
	if a.inprocFlag.Enabled {
		a.startInprocServer()
		return
	}
	if a.shouldEnableHTTPServer() {
		a.startHTTPServer()
	}
}

func (a *_AppImpl) stopServers() {
	if a.inprocFlag.Enabled {
		a.stopInprocServer()
		return
	}
	if a.shouldEnableHTTPServer() {
		a.stopHTTPServer()
	}
}

func (a *_AppImpl) shouldEnableHTTPServer() bool {
	if a.inprocFlag.Enabled {
		return false
	}
	if !a.isInternalApplication() {
		return true
	}
	return !a.spec.(InternalApplicationSpec).internalAttrs().DisableHTTPServer
}

func (a *_AppImpl) shouldRegisterApp() bool {
	if a.isInternalApplication() {
		return false
	}
	return a.servicer != nil || a.webber != nil || a.eventer != nil || a.tasker != nil
}

func (a *_AppImpl) shouldEnableConsole() bool {
	if !a.isInternalApplication() {
		return true
	}
	return !a.spec.(InternalApplicationSpec).internalAttrs().DisableConsole
}

func (a *_AppImpl) beforeAppStart() error {
	for _, componentHook := range a.componentLifecycles {
		if err := componentHook.BeforeAppStart(); err != nil {
			return err
		}
	}
	for _, module := range a.modules {
		if err := module.BeforeAppStart(); err != nil {
			return err
		}
	}
	return nil
}

func (a *_AppImpl) afterAppStart() {
	for _, componentHook := range a.componentLifecycles {
		componentHook.AfterAppStart()
	}
	for _, module := range a.modules {
		module.AfterAppStart()
	}
}

func (a *_AppImpl) beforeAppStop() {
	for i := range a.modules {
		a.modules[len(a.modules)-1-i].BeforeAppStop()
	}
	for i := range a.componentLifecycles {
		a.componentLifecycles[len(a.componentLifecycles)-1-i].BeforeAppStop()
	}
}

func (a *_AppImpl) afterAppStop() {
	for i := range a.modules {
		a.modules[len(a.modules)-1-i].AfterAppStop()
	}
	for i := range a.componentLifecycles {
		a.componentLifecycles[len(a.componentLifecycles)-1-i].AfterAppStop()
	}
}

func (a *_AppImpl) registerApp() {
	if !a.shouldRegisterApp() {
		return
	}

	serviceHandlers := []linkskeled.ServiceHandlerRegistration{}
	if a.servicer != nil {
		serviceHandlers = a.servicer.serviceHandlerRegistrations()
	}

	webHandlers := a.webHandlerRegistrations()

	eventListeners := []linkskeled.EventListenerRegistration{}
	if a.eventer != nil {
		eventListeners = a.eventer.eventListenerRegistrations()
	}

	taskRunners := []linkskeled.TaskRunnerRegistration{}
	if a.tasker != nil {
		taskRunners = a.tasker.taskRunnerRegistrations()
	}

	a.linker.RegistryClient().Register(linkskeled.AppRegistration{
		ConsoleEndpoint:   a.rpcEndpoint(coreapp.PathConsole),
		ServiceEndpoint:   a.rpcEndpoint(coreapp.PathRpcInvoke),
		WebEndpointPrefix: a.webEndpoint(coreapp.PathWebAccess),
		EventEndpoint:     a.rpcEndpoint(coreapp.PathEvent),
		TaskEndpoint:      a.rpcEndpoint(coreapp.PathTask),
		ServiceHandlers:   serviceHandlers,
		WebHandlers:       webHandlers,
		EventListeners:    eventListeners,
		TaskRunners:       taskRunners,
		DomainSchemas:     a.domainSchemas(),
	})
}

func (a *_AppImpl) unregisterApp() {
	if !a.shouldRegisterApp() {
		return
	}
	a.linker.RegistryClient().Unregister(rpcclient.WithTimeout(unregisterTimeout))
}

func (a *_AppImpl) webHandlerRegistrations() []linkskeled.WebHandlerRegistration {
	if a.webber == nil {
		return []linkskeled.WebHandlerRegistration{}
	}
	return vslice.Map(a.webber.server.WebInfos(), func(webInfo webspec.WebInfo) linkskeled.WebHandlerRegistration {
		return linkskeled.WebHandlerRegistration{
			WebSkelName: webInfo.SkelName(),
			SchemaHash:  webInfo.Hash(),
		}
	})
}

func (a *_AppImpl) domainSchemas() []skel.JSON {
	if a.linker.SkipDomainSchemas() {
		return []skel.JSON{}
	}
	schemas := skel.RegisteredDomainSchemas()
	return vslice.Map(schemas, func(schema *skel.DomainSchema) skel.JSON {
		return skel.JSON(vcode.MustMarshalJsonS(schema))
	})
}

func (a *_AppImpl) rpcEndpoint(path string) string {
	if a.inprocFlag.Enabled {
		return rpcinproc.Endpoint(a.inprocFlag.HostPath, path)
	}
	return a.httpEndpoint(path)
}

func (a *_AppImpl) webEndpoint(path string) string {
	if a.inprocFlag.Enabled {
		return webinproc.Endpoint(a.inprocFlag.HostPath, path)
	}
	return a.httpEndpoint(path)
}

func (a *_AppImpl) httpEndpoint(paths ...string) string {
	endpoint := fmt.Sprintf("http://%s:%d", a.endpointHost(), a.httpPort)
	for _, path := range paths {
		endpoint += path
	}
	return endpoint
}

func (a *_AppImpl) endpointHost() string {
	if a.httpHost == "0.0.0.0" {
		if loopbackHost, ok := a.linker.CheckLoopback(); ok {
			return loopbackHost
		}
		return detectHostIP()
	}
	return a.httpHost
}

func WaitExitSignal() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
