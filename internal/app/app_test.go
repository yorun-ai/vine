package app

import (
	coreapp "go.yorun.ai/vine/internal/core/app"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
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
	currentApp := testRuntimeApp{
		name:       "internal." + s.Flag.Value,
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000321",
	}
	s.InternalAttrs.CurrentApp = currentApp
	s.InternalAttrs.Linker = link.NewRedirectedInternalLinker(currentApp, "http://"+s.Flag.Value+".local:7071")
}

func TestNewPanicsWhenAppAlreadyCreated(t *testing.T) {
	restoreAppRegistry(t)

	specType := T[*testAppSpec]()
	defaultGuard.types[specType] = new(_GuardEntry)

	assert.PanicsWithError(t, "application *app.testAppSpec already created", func() {
		New[*testAppSpec]()
	})
}

func TestNewDoesNotRequireRegistration(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testAppSpec]()

	assert.Equal(t, "test.app", app.Name())
}

func TestNewPanicsWhenApplicationNameIsInvalid(t *testing.T) {
	restoreAppRegistry(t)

	assert.PanicsWithError(t, `invalid application name: "demo@worker"`, func() {
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
	assert.Equal(t, coreapp.InprocHostPath(app1.currentApp.InstanceId()), app1.inprocFlag.HostPath)
}

func TestNewInjectsProvidedFlagIntoAppSpec(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testInjectedAppSpec](With(&testInjectedAppFlag{Value: "demo"})).(*_AppImpl)
	spec := app.spec.(*testInjectedAppSpec)

	assert.NotNil(t, spec.Flag)
	assert.Equal(t, "demo", spec.Flag.Value)
}

// Every application derives its own identity from the declared name and the
// version linked into the binary.
func TestNewDerivesAppInfoFromSpecNameAndBuildVersion(t *testing.T) {
	restoreAppRegistry(t)

	app := New[*testNamedAppSpec]().(*_AppImpl)
	version := buildinfo.Version()

	assert.Equal(t, "demo.worker", app.currentApp.Name())
	assert.Equal(t, version, app.currentApp.Version())
	assert.True(t, meta.IsValidInstanceId(app.currentApp.InstanceId()))
}

func TestNewGivesEachApplicationItsOwnInstanceID(t *testing.T) {
	restoreAppRegistry(t)

	first := New[*testAppSpec]().(*_AppImpl)
	second := New[*testNamedAppSpec]().(*_AppImpl)

	assert.NotEqual(t, first.currentApp.InstanceId(), second.currentApp.InstanceId())
}

func TestNewUsesInternalAttrsInfoAndLinker(t *testing.T) {
	restoreAppRegistry(t)

	app := NewInternal[*testInternalAppSpec](With(&testInjectedAppFlag{Value: "demo"})).(*_AppImpl)

	assert.Equal(t, testRuntimeApp{
		name:       "internal.demo",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000321",
	}, app.currentApp)
	assert.Equal(t, "http://demo.local:7071/rpc/invoke", app.linker.RpcProxyEndpoint())
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
