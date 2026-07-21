package app

import (
	"context"
	"os"
	"reflect"
	"testing"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
)

type testLinker = link.TestLinker

type testRuntimeApp struct {
	name       string
	version    string
	instanceID string
}

func (a testRuntimeApp) Name() string {
	return a.name
}

func (a testRuntimeApp) Version() string {
	return a.version
}

func (a testRuntimeApp) InstanceId() string {
	return a.instanceID
}

type stubApp struct{}

func (stubApp) Name() string    { return "stub" }
func (stubApp) Start()          {}
func (stubApp) StopGracefully() {}
func (stubApp) StartAndWait()   {}

type testHelperAppSpec struct {
	Application
}

func (*testHelperAppSpec) Name() string {
	return "test.helper"
}

func TestMain(m *testing.M) {
	restoreLinkerFactory := link.SetNewLinkerForTest(func(_ meta.App, _ string) link.Linker {
		return &testLinker{
			RpcProxyOutEndpointValue: "http://127.0.0.1:7079/rpc/proxy/out",
		}
	})
	code := m.Run()
	restoreLinkerFactory()
	os.Exit(code)
}

func useTestLinker(t *testing.T, linker link.Linker) {
	t.Helper()

	restoreLinkerFactory := link.SetNewLinkerForTest(func(_ meta.App, _ string) link.Linker {
		return linker
	})
	t.Cleanup(restoreLinkerFactory)
}

func restoreAppRegistry(t *testing.T) {
	t.Helper()

	prevApps := appsByType
	prevNames := appsByName

	appsByType = map[reflect.Type]App{}
	appsByName = map[string]struct{}{}

	t.Cleanup(func() {
		appsByType = prevApps
		appsByName = prevNames
	})
}

func newTestAppImpl() *_AppImpl {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := &_AppImpl{
		spec: &testHelperAppSpec{},
		info: testRuntimeApp{
			name:       "test.app",
			version:    "1.2.3",
			instanceID: "00000000-0000-0000-0000-000000000123",
		},
		ctx:        context.Background(),
		cancel:     func() {},
		doneSignal: make(chan struct{}),
		flags:      flags,
	}
	app.inprocFlag = app.flags.InprocFlag()
	app.linker = link.NewRedirectedInternalLinker(app.info, "http://test.local:7071")
	app.initInjector()
	app.initServers()
	return app
}
