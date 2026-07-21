package app

import (
	coreapp "go.yorun.ai/vine/internal/core/app"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/runtime"
)

type testAppSpec struct {
	Application
}

func (*testAppSpec) Name() string {
	return "test.app"
}

type testInjectedAppFlag struct {
	FlagModel
	Value string
}

type testInjectedAppSpec struct {
	Application
	Flag *testInjectedAppFlag `inject:""`
}

func (*testInjectedAppSpec) Name() string {
	return "test.injected.app"
}

type testRuntimeNamedAppSpec struct {
	Application
}

func (*testRuntimeNamedAppSpec) Name() string {
	return runtime.Application().Name()
}

type testNamedAppSpec struct {
	Application
}

func (*testNamedAppSpec) Name() string {
	return "demo.worker"
}

type testDuplicateNamedAppSpec struct {
	Application
}

func (*testDuplicateNamedAppSpec) Name() string {
	return "demo.worker"
}

type testInvalidNamedAppSpec struct {
	Application
}

func (*testInvalidNamedAppSpec) Name() string {
	return "demo@worker"
}

type testInternalAppSpec struct {
	InternalApplication
	Flag *testInjectedAppFlag `inject:""`
}

func (*testInternalAppSpec) Name() string {
	return "internal.http"
}

func (s *testInternalAppSpec) DIInit() {
	info := testRuntimeApp{
		name:       "internal." + s.Flag.Value,
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000321",
	}
	s.InternalAttrs.Info = info
	s.InternalAttrs.Linker = link.NewRedirectedInternalLinker(info, "http://"+s.Flag.Value+".local:7071")
}

func TestNewPanicsWhenAppAlreadyCreated(t *testing.T) {
	restoreAppRegistry(t)

	specType := T[*testAppSpec]()
	cached := &stubApp{}
	appsByType[specType] = cached

	assert.PanicsWithError(t, "application *app.testAppSpec already created", func() {
		New[*testAppSpec]()
	})
}

func TestNewDoesNotRequireRegistration(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testAppSpec]()

	assert.Equal(t, "test.app", app.Name())
}

func TestNewPanicsWhenApplicationNameContainsAt(t *testing.T) {
	restoreAppRegistry(t)

	assert.PanicsWithError(t, "application name must not contain @", func() {
		New[*testInvalidNamedAppSpec]()
	})
}

func TestNewPanicsWhenApplicationNameAlreadyCreated(t *testing.T) {
	restoreAppRegistry(t)

	New[*testNamedAppSpec]()

	assert.PanicsWithError(t, "application name demo.worker already created", func() {
		New[*testDuplicateNamedAppSpec]()
	})
}

func TestNewWithFlagPanicsForSameType(t *testing.T) {
	restoreAppRegistry(t)

	New[*testAppSpec](With(&RunFlag{ListenAddr: ":18080"}))

	assert.PanicsWithError(t, "application *app.testAppSpec already created", func() {
		New[*testAppSpec](With(&RunFlag{ListenAddr: ":18081"}))
	})
}

func TestNewWithFlagUsesProvidedListenAddr(t *testing.T) {
	restoreAppRegistry(t)

	app1 := New[*testAppSpec](With(&RunFlag{ListenAddr: ":18080"})).(*_AppImpl)

	assert.Equal(t, ":18080", app1.listenAddr)
}

func TestNewInprocEnablesInprocMode(t *testing.T) {
	restoreAppRegistry(t)

	app1 := NewInproc[*testAppSpec]().(*_AppImpl)

	assert.NotNil(t, app1.inprocFlag)
	assert.True(t, app1.inprocFlag.Enabled)
	assert.Equal(t, coreapp.InprocHostPath(app1.info.InstanceId()), app1.inprocFlag.HostPath)
}

func TestNewInjectsProvidedFlagIntoAppSpec(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testInjectedAppSpec](With(&testInjectedAppFlag{Value: "demo"})).(*_AppImpl)
	spec := app.spec.(*testInjectedAppSpec)

	assert.NotNil(t, spec.Flag)
	assert.Equal(t, "demo", spec.Flag.Value)
}

func TestNewUsesRuntimeApplicationInfoWhenSpecNameMatchesRuntime(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testRuntimeNamedAppSpec]().(*_AppImpl)
	runtimeApp := runtime.Application()

	assert.Same(t, runtimeApp, app.info)
}

func TestNewInprocDerivesDedicatedAppInfoWhenSpecNameMatchesRuntime(t *testing.T) {
	restoreAppRegistry(t)

	app := NewInproc[*testRuntimeNamedAppSpec]().(*_AppImpl)
	runtimeApp := runtime.Application()

	assert.Equal(t, runtimeApp.Name()+"@"+runtimeApp.Name(), app.info.Name())
	assert.Equal(t, runtimeApp.Version(), app.info.Version())
	assert.NotEqual(t, runtimeApp.InstanceId(), app.info.InstanceId())
}

func TestNewDerivesDedicatedAppInfoWhenSpecNameDiffersFromRuntime(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testNamedAppSpec]().(*_AppImpl)
	runtimeApp := runtime.Application()

	assert.Equal(t, "demo.worker@"+runtimeApp.Name(), app.info.Name())
	assert.Equal(t, runtimeApp.Version(), app.info.Version())
	assert.NotEqual(t, runtimeApp.InstanceId(), app.info.InstanceId())
}

func TestNewUsesInternalAttrsInfoAndLinker(t *testing.T) {
	restoreAppRegistry(t)

	app := NewInternal[*testInternalAppSpec](With(&testInjectedAppFlag{Value: "demo"})).(*_AppImpl)

	assert.Equal(t, testRuntimeApp{
		name:       "internal.demo",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000321",
	}, app.info)
	assert.Equal(t, "http://demo.local:7071/rpc/invoke", app.linker.RpcProxyEndpoint())
}

func TestNewInternalByTypePanicsWhenSpecIsNotInternal(t *testing.T) {
	restoreAppRegistry(t)

	assert.PanicsWithError(t, "application spec *app.testAppSpec is not internal", func() {
		newInternalByType(T[*testAppSpec](), false)
	})
}

func TestNewProvidesDefaultInjectedFlagWhenNotPassed(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testInjectedAppSpec]().(*_AppImpl)
	spec := app.spec.(*testInjectedAppSpec)

	assert.NotNil(t, spec.Flag)
	assert.Equal(t, "", spec.Flag.Value)
}

func TestNewProvidesDefaultRunFlagWhenNotPassed(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testAppSpec]().(*_AppImpl)
	spec := app.spec.(*testAppSpec)

	assert.NotNil(t, spec.AppFlag)
	assert.Equal(t, "", spec.AppFlag.ListenAddr)
}

func TestEnabledSpecsDefaultToNil(t *testing.T) {
	assert.Equal(t, "", (&Application{AppFlag: &RunFlag{}}).AppFlag.ListenAddr)
	assert.Equal(t, "", (&Application{}).Name())
	(&Application{}).BindCommon(nil)
	(&ServicerEnabled{}).ServicerInitHandlers(nil)
	(&ServicerEnabled{}).ServicerInitFilters(nil)
	(&ServicerEnabled{}).ServicerBind(nil)
	(&WebberEnabled{}).WebberInitHandlers(nil)
	(&WebberEnabled{}).WebberInitFilters(nil)
	(&WebberEnabled{}).WebberBind(nil)
	(&EventerEnabled{}).EventerBind(nil)
	(&EventerEnabled{}).EventerInitListeners(nil)
	(&EventerEnabled{}).EventerInitFilters(nil)
	(&TaskerEnabled{}).TaskerBind(nil)
	(&TaskerEnabled{}).TaskerInitRunners(nil)
	(&TaskerEnabled{}).TaskerInitFilters(nil)
}
