package app

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/di"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	web "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/util/vslice"
)

type testFullSpec struct {
	Application
	EventerEnabled
	TaskerEnabled
}

type _TestHTTPListener struct {
	closed chan struct{}
	addr   *net.TCPAddr
}

func newTestHTTPListener(port int) *_TestHTTPListener {
	return &_TestHTTPListener{
		closed: make(chan struct{}),
		addr:   &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port},
	}
}

func useTestHTTPListener(t *testing.T, port int) {
	t.Helper()
	previous := newHTTPListener
	newHTTPListener = func(network string, address string) (net.Listener, error) {
		assert.Equal(t, "tcp", network)
		assert.Equal(t, defaultHTTPListenAddr, address)
		return newTestHTTPListener(port), nil
	}
	t.Cleanup(func() {
		newHTTPListener = previous
	})
}

func (l *_TestHTTPListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *_TestHTTPListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}

func (l *_TestHTTPListener) Addr() net.Addr {
	return l.addr
}

func (*testFullSpec) Name() string {
	return "test.full"
}

func (*testFullSpec) EventerBind(*di.Binder) {}

func (*testFullSpec) EventerInitListeners(addListener ListenerTypeAdder) {
	addListener(T[*testEventerListenerImpl]())
}

func (*testFullSpec) EventerInitFilters(addFilter TypeAdder) {
	addFilter(T[*testEventerFilter]())
}

func (*testFullSpec) TaskerBind(*di.Binder) {}

func (*testFullSpec) TaskerInitRunners(addRunner RunnerTypeAdder) {
	addRunner(T[*testTaskerRunnerImpl]())
}

func (*testFullSpec) TaskerInitFilters(addFilter TypeAdder) {
	addFilter(T[*testTaskerFilter]())
}

type testListenAddrSpec struct {
	Application
}

func (*testListenAddrSpec) Name() string {
	return "test.listen"
}

func (s *testListenAddrSpec) DIInit() {
	if s.AppFlag.ListenAddr == "" {
		s.AppFlag.ListenAddr = ":18089"
	}
}

type testHookSpec struct {
	Application
	componentTypes []reflect.Type
	moduleTypes    []reflect.Type
}

func (*testHookSpec) Name() string {
	return "test.hook"
}

func (s *testHookSpec) InitComponents(addComponent TypeAdder) {
	for _, componentType := range s.componentTypes {
		addComponent(componentType)
	}
}

func (s *testHookSpec) InitModules(addModule TypeAdder) {
	for _, moduleType := range s.moduleTypes {
		addModule(moduleType)
	}
}

type testHookModule struct {
	BaseModule
	Log *testLifecycleLog `inject:""`
}

func (p *testHookModule) DIInit() {
	*p.Log.Events = append(*p.Log.Events, "di-init")
}

func (p *testHookModule) BeforeAppStart() error {
	*p.Log.Events = append(*p.Log.Events, "before-start")
	return nil
}

func (p *testHookModule) AfterAppStart() {
	*p.Log.Events = append(*p.Log.Events, "after-start")
}

func (p *testHookModule) BeforeAppStop() {
	*p.Log.Events = append(*p.Log.Events, "before-stop")
}

func (p *testHookModule) AfterAppStop() {
	*p.Log.Events = append(*p.Log.Events, "after-stop")
}

type testHookErrorSpec struct {
	Application
	moduleTypes []reflect.Type
}

func (*testHookErrorSpec) Name() string {
	return "test.hook.error"
}

func (s *testHookErrorSpec) InitModules(addModule TypeAdder) {
	for _, moduleType := range s.moduleTypes {
		addModule(moduleType)
	}
}

type testInjectedModuleFlag struct {
	FlagModel
	Value string
}

type testLifecycleLog struct {
	FlagModel
	Events *[]string
}

type testHookErrorState struct {
	FlagModel
	Err error
}

type testInjectedModule struct {
	BaseModule
	Flag *testInjectedModuleFlag `inject:""`
}

type testInjectedModuleSpec struct {
	Application
}

func (*testInjectedModuleSpec) Name() string {
	return "test.injected.module"
}

func (*testInjectedModuleSpec) InitModules(addModule TypeAdder) {
	addModule(T[*testInjectedModule]())
}

type testHTTPRouteModuleSpec struct {
	Application
}

func (*testHTTPRouteModuleSpec) Name() string {
	return "test.http.route.module"
}

func (*testHTTPRouteModuleSpec) InitModules(addModule TypeAdder) {
	addModule(T[*testHTTPRouteModule]())
}

type testHookErrorModule struct {
	BaseModule
	State *testHookErrorState `inject:""`
}

func (p *testHookErrorModule) BeforeAppStart() error {
	return p.State.Err
}

type testHTTPRouteModule struct {
	BaseModule
}

func (*testHTTPRouteModule) InitPathPrefixRoute(add PathPrefixRouteAdder) {
	add(strings.TrimSuffix("/rpc/proxy/out", "/out"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}), nil)
}

type testServicerSpec struct {
	Application
	ServicerEnabled
}

func (*testServicerSpec) Name() string {
	return "test.servicer"
}

func (*testServicerSpec) ServicerInitHandlers(addHandler TypeAdder) {
	addHandler(T[*ConsoleServiceServerImpl]())
}

type testUniqueWebberRegisterSpec struct {
	InternalApplication
	WebberEnabled
}

func (*testUniqueWebberRegisterSpec) Name() string {
	return "test.unique.webber.register"
}

func (*testUniqueWebberRegisterSpec) WebberBind(*di.Binder) {}

func (*testUniqueWebberRegisterSpec) WebberInitFilters(TypeAdder) {}

func (*testUniqueWebberRegisterSpec) WebberInitHandlers(addHandler TypeAdder) {
	addHandler(T[*testUniqueWebberRegisterHandler]())
}

type testWebberRegisterSpec struct {
	Application
	WebberEnabled
}

func (*testWebberRegisterSpec) Name() string {
	return "test.webber.register"
}

func (*testWebberRegisterSpec) WebberBind(*di.Binder) {}

func (*testWebberRegisterSpec) WebberInitFilters(TypeAdder) {}

func (*testWebberRegisterSpec) WebberInitHandlers(addHandler TypeAdder) {
	addHandler(T[*testUniqueWebberRegisterHandler]())
}

type testUniqueWebberRegisterHandler struct {
	defaultTestUniqueWebberRegisterWebServer
	GinCtx *gin.Context `inject:""`
}

func (h *testUniqueWebberRegisterHandler) Routes(r *web.Router) {
	r.GET("/ping", h.Ping)
}

func (h *testUniqueWebberRegisterHandler) Ping() {
	h.GinCtx.Status(http.StatusNoContent)
}

type testUniqueWebberRegisterWebServer interface {
	web.Handler

	mustBeTestUniqueWebberRegisterWebServer()
}

type defaultTestUniqueWebberRegisterWebServer struct {
}

func (*defaultTestUniqueWebberRegisterWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*defaultTestUniqueWebberRegisterWebServer) mustBeTestUniqueWebberRegisterWebServer() {}

func init() {
	web.Register(&web.WebSpec{
		Name:              "TestUniqueWebberRegisterWeb",
		SkelName:          "demo.user.TestUniqueWebberRegisterWeb",
		ServerType:        reflect.TypeOf((*testUniqueWebberRegisterWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultTestUniqueWebberRegisterWebServer](),
	})
}

type testInternalServicerSpec struct {
	InternalApplication
	ServicerEnabled
}

func (*testInternalServicerSpec) Name() string {
	return "test.internal.servicer"
}

func (*testInternalServicerSpec) ServicerInitHandlers(addHandler TypeAdder) {
	addHandler(T[*ConsoleServiceServerImpl]())
}

type TestLifecycleFrameworkComponent struct {
	BaseFrameworkComponent[*TestLifecycleFrameworkComponent]
	BaseFrameworkComponentMinder
	Log *testLifecycleLog `inject:""`

	component FrameworkComponent
}

func (c *TestLifecycleFrameworkComponent) InitComponent(component FrameworkComponent) {
	c.component = component
}

func (c *TestLifecycleFrameworkComponent) Component() FrameworkComponent {
	return c.component
}

func (c *TestLifecycleFrameworkComponent) BeforeAppStart() error {
	*c.Log.Events = append(*c.Log.Events, "fx-component-before-start")
	return nil
}

func (c *TestLifecycleFrameworkComponent) AfterAppStart() {
	*c.Log.Events = append(*c.Log.Events, "fx-component-after-start")
}

func (c *TestLifecycleFrameworkComponent) BeforeAppStop() {
	*c.Log.Events = append(*c.Log.Events, "fx-component-before-stop")
}

func (c *TestLifecycleFrameworkComponent) AfterAppStop() {
	*c.Log.Events = append(*c.Log.Events, "fx-component-after-stop")
}

type testLifecycleComponent struct {
	TestLifecycleFrameworkComponent
}

func (c *testLifecycleComponent) BeforeAppStart() error {
	*c.Log.Events = append(*c.Log.Events, "component-before-start")
	return nil
}

func (c *testLifecycleComponent) AfterAppStart() {
	*c.Log.Events = append(*c.Log.Events, "component-after-start")
}

func (c *testLifecycleComponent) BeforeAppStop() {
	*c.Log.Events = append(*c.Log.Events, "component-before-stop")
}

func (c *testLifecycleComponent) AfterAppStop() {
	*c.Log.Events = append(*c.Log.Events, "component-after-stop")
}

type testLifecycleModule struct {
	BaseModule
	Log *testLifecycleLog `inject:""`
}

type testLifecycleSimpleComponent struct {
	BaseComponent
	Log *testLifecycleLog `inject:""`
}

func (c *testLifecycleSimpleComponent) BeforeAppStart() error {
	*c.Log.Events = append(*c.Log.Events, "simple-before-start")
	return nil
}

func (c *testLifecycleSimpleComponent) AfterAppStart() {
	*c.Log.Events = append(*c.Log.Events, "simple-after-start")
}

func (c *testLifecycleSimpleComponent) BeforeAppStop() {
	*c.Log.Events = append(*c.Log.Events, "simple-before-stop")
}

func (c *testLifecycleSimpleComponent) AfterAppStop() {
	*c.Log.Events = append(*c.Log.Events, "simple-after-stop")
}

func (p *testLifecycleModule) BeforeAppStart() error {
	*p.Log.Events = append(*p.Log.Events, "module-before-start")
	return nil
}

func (p *testLifecycleModule) AfterAppStart() {
	*p.Log.Events = append(*p.Log.Events, "module-after-start")
}

func (p *testLifecycleModule) BeforeAppStop() {
	*p.Log.Events = append(*p.Log.Events, "module-before-stop")
}

func (p *testLifecycleModule) AfterAppStop() {
	*p.Log.Events = append(*p.Log.Events, "module-after-stop")
}

func TestHTTPRouteServeHTTPRewritesPathAndRequestURI(t *testing.T) {
	var path string
	var requestURI string

	route := _ServerRoute{
		Prefix: coreapp.PathRpcInvoke,
		HttpHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path = r.URL.Path
			requestURI = r.RequestURI
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, coreapp.PathRpcInvoke+"/demo/ping", nil)
	recorder := httptest.NewRecorder()
	route.serveHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, "/demo/ping", path)
	assert.Equal(t, "/demo/ping", requestURI)
}

func TestAppImplInitConsoleServerCreatesHandler(t *testing.T) {
	app := newTestAppImpl()

	assert.NotNil(t, app.consoleServer)
	assert.NotNil(t, app.consoleServer.HTTPHandler())
	assert.True(t, app.shouldEnableConsole())
}

type testInternalOnlyAppSpec struct {
	InternalApplication
}

func (*testInternalOnlyAppSpec) Name() string {
	return "test.internal.only"
}

type testUnnamedInternalOnlyAppSpec struct {
	InternalApplication
}

type testEmptyNameInternalOnlyAppSpec struct {
	InternalApplication
}

func (*testEmptyNameInternalOnlyAppSpec) Name() string {
	return ""
}

func TestAppImplInitConsoleServerSkipsWhenDisabledByInternalAttrs(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)

	app := newApp(&testInternalOnlyAppSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:         &testLinker{},
				DisableConsole: true,
			},
		},
	}, flags)
	app.initInjector()
	app.initServers()

	assert.Nil(t, app.consoleServer)
	assert.False(t, app.shouldEnableConsole())
}

func TestAppImplInitConsoleServerCreatesHandlerWhenInprocEnabled(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testHelperAppSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)
	app.initInjector()
	app.initServers()

	assert.NotNil(t, app.consoleServer)
	assert.NotNil(t, app.consoleServer.HTTPHandler())
	assert.NotNil(t, app.consoleServer.RpcHandler())
	assert.True(t, app.shouldEnableConsole())
}

func TestNewAppPanicsWhenApplicationNameIsEmpty(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)

	assert.PanicsWithError(t, "application name must not be empty", func() {
		_ = newApp(&testEmptyNameInternalOnlyAppSpec{
			InternalApplication: InternalApplication{
				Application: Application{AppFlag: &RunFlag{}},
				InternalAttrs: InternalAttributes{
					Info: testRuntimeApp{
						name:       "test.app",
						version:    "1.2.3",
						instanceID: "00000000-0000-0000-0000-000000000123",
					},
					Linker: &testLinker{},
				},
			},
		}, flags)
	})
}

func TestAppImplInitSpecInitializesEnabledModules(t *testing.T) {
	ensureTaskerTaskRegistered()
	ensureEventerEventRegistered()

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testFullSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)
	app.initServers()

	assert.NotNil(t, app.eventer)
	assert.NotNil(t, app.tasker)
}

func TestNewAppUsesRunFlagListenAddr(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	flags[T[*RunFlag]()] = &RunFlag{ListenAddr: ":18089"}
	app := newApp(&testListenAddrSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)

	assert.Equal(t, ":18089", app.listenAddr)
}

func TestNewAppDisablesInprocModeByDefault(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)

	app := newApp(&testListenAddrSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)

	assert.NotNil(t, app.inprocFlag)
	assert.False(t, app.inprocFlag.Enabled)
}

func TestNewAppEnablesInprocModeWhenFlagProvided(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)

	app := newApp(&testListenAddrSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)

	assert.NotNil(t, app.inprocFlag)
	assert.True(t, app.inprocFlag.Enabled)
	assert.Equal(t, coreapp.InprocHostPath(app.info.InstanceId()), app.inprocFlag.HostPath)
}

func TestNewAppUsesRunFlagListenAddrMutatedInDIInit(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	spec := newSpec(T[*testListenAddrSpec](), flags).(*testListenAddrSpec)
	app := newApp(spec, flags)

	assert.Equal(t, ":18089", app.listenAddr)
}

func TestAppImplStartIsNonBlocking(t *testing.T) {
	app := newTestAppImpl()
	useTestHTTPListener(t, 18081)

	started := make(chan struct{})
	go func() {
		app.Start()
		close(started)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Start should return without blocking")
	}

	app.StopGracefully()
}

func TestAppImplStartPanicsWhenAlreadyStarted(t *testing.T) {
	app := newTestAppImpl()
	useTestHTTPListener(t, 18081)
	app.Start()
	defer app.StopGracefully()

	assert.PanicsWithError(t, "application already started", func() {
		app.Start()
	})
}

func TestAppImplStopGracefullyPanicsWhenNotStarted(t *testing.T) {
	app := newTestAppImpl()

	assert.PanicsWithError(t, "application is not started", func() {
		app.StopGracefully()
	})
}

func TestAppImplStopGracefullyPanicsWhenAlreadyStopped(t *testing.T) {
	app := newTestAppImpl()
	useTestHTTPListener(t, 18081)
	app.Start()
	app.StopGracefully()

	assert.PanicsWithError(t, "application already stopped", func() {
		app.StopGracefully()
	})
}

func TestAppImplStartPanicsAfterStopped(t *testing.T) {
	app := newTestAppImpl()
	useTestHTTPListener(t, 18081)
	app.Start()
	app.StopGracefully()

	assert.PanicsWithError(t, "application already stopped", func() {
		app.Start()
	})
}

func TestAppImplStartAndStopInvokeLifecycleHooks(t *testing.T) {
	events := []string{}
	spec := &testHookSpec{
		Application: Application{AppFlag: &RunFlag{}},
		moduleTypes: []reflect.Type{
			T[*testHookModule](),
		},
	}
	flags := _Flags{}
	flags.Apply(With(&testLifecycleLog{
		Events: &events,
	}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	app := newApp(spec, flags)

	app.Start()
	app.StopGracefully()

	assert.True(t, vslice.Equal([]string{
		"di-init",
		"before-start",
		"after-start",
		"before-stop",
		"after-stop",
	}, events))
}

func TestStartHTTPServerDefaultsToLoopbackEphemeralAddress(t *testing.T) {
	app := newTestAppImpl()
	useTestHTTPListener(t, 18080)

	app.startHTTPServer()
	defer app.stopHTTPServer()

	expectedHost, _, err := net.SplitHostPort(defaultHTTPListenAddr)
	if err != nil {
		t.Fatalf("split default host port failed: %v", err)
	}
	assert.Equal(t, expectedHost, app.httpHost)
	assert.NotZero(t, app.httpPort)
}

func TestAppImplStartDoesNotRegisterInternalApp(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	linker := &testLinker{}
	app := newApp(&testInternalServicerSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:            linker,
				DisableHTTPServer: true,
			},
		},
	}, flags)

	app.Start()
	app.StopGracefully()

	assert.Equal(t, "", linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 0, linker.UnregisterCalls)
}

func TestAppImplStartInprocModeSkipsHTTPServerAndLinkerRegistration(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	linker := &testLinker{}
	app := newApp(&testInternalServicerSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:         linker,
				InprocHostPath: "app/test-start-inproc",
			},
		},
	}, flags)

	app.Start()
	app.StopGracefully()

	assert.Nil(t, app.httpServer)
	assert.Equal(t, "", app.httpHost)
	assert.Equal(t, 0, app.httpPort)
	assert.Equal(t, "", linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 0, linker.UnregisterCalls)
}

func TestAppImplStartWithDisableHTTPServerDoesNotRegisterInternalApp(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	linker := &testLinker{}
	app := newApp(&testInternalServicerSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:            linker,
				DisableHTTPServer: true,
			},
		},
	}, flags)

	app.Start()
	app.StopGracefully()

	assert.Nil(t, app.httpServer)
	assert.Equal(t, "", app.httpHost)
	assert.Equal(t, 0, app.httpPort)
	assert.Equal(t, "", linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 0, linker.UnregisterCalls)
}

func TestAppImplStartInprocModeStillUsesInprocServerWhenHTTPServerDisabled(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	linker := &testLinker{}
	app := newApp(&testInternalServicerSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:            linker,
				DisableHTTPServer: true,
				InprocHostPath:    "app/test-start-inproc-disable-http",
			},
		},
	}, flags)

	app.Start()
	app.StopGracefully()

	assert.Nil(t, app.httpServer)
	assert.Equal(t, "", app.httpHost)
	assert.Equal(t, 0, app.httpPort)
	assert.Equal(t, "", linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 0, linker.UnregisterCalls)
}

func TestAppImplStartDoesNotRegisterInternalUniqueWebberApp(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	linker := &testLinker{}
	app := newApp(&testUniqueWebberRegisterSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:            linker,
				DisableHTTPServer: true,
			},
		},
	}, flags)

	app.Start()
	app.StopGracefully()

	assert.Equal(t, "", linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 0, linker.UnregisterCalls)
}

func TestAppImplStartRegistersWebberOnlyAppWithNonNilEmptyCapabilities(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	linker := &testLinker{}
	useTestLinker(t, linker)
	app := newApp(&testWebberRegisterSpec{
		Application:   Application{AppFlag: &RunFlag{}},
		WebberEnabled: WebberEnabled{},
	}, flags)
	app.info = testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}
	app.inprocFlag.HostPath = coreapp.InprocHostPath(app.info.InstanceId())

	app.Start()
	app.StopGracefully()

	assert.NotNil(t, linker.RegisterServiceHandlers)
	assert.NotNil(t, linker.RegisterWebHandlers)
	assert.NotNil(t, linker.RegisterEventListeners)
	assert.NotNil(t, linker.RegisterTaskRunners)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Len(t, linker.RegisterWebHandlers, 1)
	assert.Empty(t, linker.RegisterEventListeners)
	assert.Empty(t, linker.RegisterTaskRunners)
	assert.Equal(t, 1, linker.UnregisterCalls)
}

func TestAppImplStartRegistersEventerAndTasker(t *testing.T) {
	ensureTaskerTaskRegistered()
	ensureEventerEventRegistered()

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	linker := &testLinker{}
	useTestLinker(t, linker)
	app := newApp(&testFullSpec{
		Application:    Application{AppFlag: &RunFlag{}},
		EventerEnabled: EventerEnabled{},
		TaskerEnabled:  TaskerEnabled{},
	}, flags)
	app.info = testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}
	app.inprocFlag.HostPath = coreapp.InprocHostPath(app.info.InstanceId())

	app.Start()
	app.StopGracefully()

	assert.Equal(t, "rpc+inproc://"+coreapp.InprocHostPath(app.info.InstanceId())+coreapp.PathRpcInvoke, linker.RegisterServiceEndpoint)
	assert.Empty(t, linker.RegisterServiceHandlers)
	assert.Empty(t, linker.RegisterWebHandlers)
	assert.Equal(t, []linkskeled.EventListenerRegistration{{
		EventSkelName: "test.eventer.TestEventerEvent",
		TimeoutMs:     30000,
		Concurrency:   10,
		NoRetry:       false,
	}}, linker.RegisterEventListeners)
	assert.Equal(t, []linkskeled.TaskRunnerRegistration{{
		TaskSkelName:   "test.tasker.TaskerTestTask",
		TimeoutMs:      30000,
		Concurrency:    10,
		NoRetry:        false,
		CronSchedulers: []linkskeled.TaskRunnerCronScheduler{},
	}}, linker.RegisterTaskRunners)
	assert.Equal(t, 1, linker.UnregisterCalls)
}

func TestAppImplStartSkipsDomainSchemasWhenSkipDomainSchemasEnabled(t *testing.T) {
	ensureTaskerTaskRegistered()
	ensureEventerEventRegistered()

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	linker := &testLinker{SkipDomainSchemasValue: true}
	useTestLinker(t, linker)
	app := newApp(&testFullSpec{
		Application:    Application{AppFlag: &RunFlag{}},
		EventerEnabled: EventerEnabled{},
		TaskerEnabled:  TaskerEnabled{},
	}, flags)
	app.info = testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}

	app.Start()
	app.StopGracefully()

	assert.Nil(t, linker.RegisterDomainSchemas)
}

func TestAppImplStopGracefullyStopsInprocMode(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	app := newApp(&testInternalServicerSpec{
		InternalApplication: InternalApplication{
			Application: Application{AppFlag: &RunFlag{}},
			InternalAttrs: InternalAttributes{
				Info: testRuntimeApp{
					name:       "test.app",
					version:    "1.2.3",
					instanceID: "00000000-0000-0000-0000-000000000123",
				},
				Linker:         &testLinker{},
				InprocHostPath: "app/test-wait-inproc",
			},
		},
	}, flags)

	app.Start()

	stopDone := make(chan struct{})
	go func() {
		app.StopGracefully()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("StopGracefully should complete in inproc mode")
	}
}

func TestAppImplStopUnregistersInprocRoutes(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	newInprocApp := func() *_AppImpl {
		return newApp(&testInternalServicerSpec{
			InternalApplication: InternalApplication{
				Application: Application{AppFlag: &RunFlag{}},
				InternalAttrs: InternalAttributes{
					Info: testRuntimeApp{
						name:       "test.app",
						version:    "1.2.3",
						instanceID: "00000000-0000-0000-0000-000000000123",
					},
					Linker:         &testLinker{},
					InprocHostPath: "app/test-stop-inproc",
				},
			},
		}, flags)
	}

	first := newInprocApp()
	first.Start()
	first.StopGracefully()

	second := newInprocApp()
	second.Start()
	second.StopGracefully()
}

func TestAppImplStartAndStopInvokeComponentAndModuleLifecycleHooksInOrder(t *testing.T) {
	events := []string{}
	spec := &testHookSpec{
		Application: Application{AppFlag: &RunFlag{}},
		componentTypes: []reflect.Type{
			T[*testLifecycleComponent](),
			T[*testLifecycleSimpleComponent](),
		},
		moduleTypes: []reflect.Type{
			T[*testLifecycleModule](),
		},
	}
	flags := _Flags{}
	flags.Apply(With(&testLifecycleLog{
		Events: &events,
	}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	app := newApp(spec, flags)

	app.Start()
	app.StopGracefully()

	assert.True(t, vslice.Equal([]string{
		"fx-component-before-start",
		"simple-before-start",
		"module-before-start",
		"fx-component-after-start",
		"simple-after-start",
		"module-after-start",
		"module-before-stop",
		"simple-before-stop",
		"fx-component-before-stop",
		"module-after-stop",
		"simple-after-stop",
		"fx-component-after-stop",
	}, events))
}

func TestAppImplStartPanicsWhenBeforeStartFails(t *testing.T) {
	spec := &testHookErrorSpec{
		Application: Application{AppFlag: &RunFlag{}},
		moduleTypes: []reflect.Type{
			T[*testHookErrorModule](),
		},
	}
	flags := _Flags{}
	flags.Apply(With(&testHookErrorState{Err: errors.New("boom")}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(spec, flags)

	assert.PanicsWithError(t, "application before start failed: boom", func() {
		app.Start()
	})
}

func TestInitModulesInjectsProvidedFlag(t *testing.T) {
	spec := &testInjectedModuleSpec{Application: Application{AppFlag: &RunFlag{}}}
	flags := _Flags{}
	flags.Apply(With(&testInjectedModuleFlag{Value: "demo"}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(spec, flags)
	app.initInjector()
	app.initModules()

	if assert.Len(t, app.modules, 1) {
		module, ok := app.modules[0].(*testInjectedModule)
		if assert.True(t, ok) {
			assert.NotNil(t, module.Flag)
			assert.Equal(t, "demo", module.Flag.Value)
		}
	}
}

func TestStartModulesPanicsOnDuplicateModuleTypes(t *testing.T) {
	spec := &testHookSpec{
		Application: Application{AppFlag: &RunFlag{}},
		moduleTypes: []reflect.Type{
			T[*testHookModule](),
			T[*testHookModule](),
		},
	}

	assert.PanicsWithError(t, "module type *app.testHookModule already declared", func() {
		flags := _Flags{}
		flags.EnsureRunFlag()
		flags.InitInprocFlag(false)
		app := newApp(spec, flags)
		app.initInjector()
		app.initModules()
	})
}

func TestAppImplBuiltinRouteServesConsolePrefix(t *testing.T) {
	app := newTestAppImpl()

	req := httptest.NewRequest(http.MethodPost, "/console/vine.app.ConsoleService/ping", nil)
	recorder := httptest.NewRecorder()

	assert.True(t, app.serveHTTPRoutes(recorder, req))

	resp := recorder.Result()
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestAppImplMountsHTTPRouteModulePrefix(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testHTTPRouteModuleSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)
	app.initInjector()
	app.initModules()
	app.initServers()

	req := httptest.NewRequest(http.MethodGet, "/rpc/proxy/out/demo.Service/method", nil)
	recorder := httptest.NewRecorder()

	assert.True(t, app.serveHTTPRoutes(recorder, req))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "/out/demo.Service/method", recorder.Body.String())
}

func TestAppImplHTTPHandlerReturns404WhenPathIsNotBuiltin(t *testing.T) {
	app := newTestAppImpl()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	app.httpHandler().ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAppImplHTTPHandlerReturns404ByDefaultForNonBuiltinPath(t *testing.T) {
	app := newTestAppImpl()

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()

	app.httpHandler().ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAppImplHTTPHandlerReturns404ForRPCPathWhenRPCServerIsDisabled(t *testing.T) {
	app := newTestAppImpl()

	req := httptest.NewRequest(http.MethodGet, coreapp.PathRpcInvoke+"/demo/ping", nil)
	recorder := httptest.NewRecorder()

	app.httpHandler().ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAppImplHTTPEndpointUsesLoopbackEndpointForExplicitHost(t *testing.T) {
	app := newTestAppImpl()
	app.httpHost = "127.0.0.1"
	app.httpPort = 18080

	assert.Equal(t, "http://127.0.0.1:18080", app.httpEndpoint())
}

func TestAppImplHTTPEndpointUsesLoopbackHostForWildcardListenAddr(t *testing.T) {
	app := newTestAppImpl()
	app.httpHost = "0.0.0.0"
	app.httpPort = 18080
	app.linker = &testLinker{
		LoopbackHostValue: "localhost",
		HasLoopbackValue:  true,
	}

	assert.Equal(t, "http://localhost:18080", app.httpEndpoint())
}

func TestAppImplHTTPEndpointUsesDetectedHostIPForWildcardListenAddr(t *testing.T) {
	app := newTestAppImpl()
	app.httpHost = "0.0.0.0"
	app.httpPort = 18080
	app.linker = &testLinker{}

	prevDetectHostIP := detectHostIP
	detectHostIP = func() string {
		return "10.0.0.8"
	}
	t.Cleanup(func() {
		detectHostIP = prevDetectHostIP
	})

	assert.Equal(t, "http://10.0.0.8:18080", app.httpEndpoint())
}
