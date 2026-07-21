package testkit

import (
	"context"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/app/standalone"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/core/rpc"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	internalrpc "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
)

// Runtime owns a standalone Vine runtime started for a test package.
type Runtime struct {
	app                 app.App
	clientApp           meta.App
	clientAppRegistered bool
}

var (
	runtimeMutex       sync.Mutex
	startedRuntimeSpec reflect.Type
)

// StartStandalone starts one process-wide standalone runtime and registers cleanup with t.
// Only one runtime may be started in a test process.
func StartStandalone[S app.ApplicationSpec](t testing.TB, option Option, appliers ...app.FlagApplier) *Runtime {
	t.Helper()

	specType := app.T[S]()
	runtimeMutex.Lock()
	if startedRuntimeSpec != nil {
		runtimeMutex.Unlock()
		t.Fatalf(
			"testkit standalone already started for %s in this process; "+
				"Vine standalone apps are process-level singletons, "+
				"so start one runtime per test package and share it",
			startedRuntimeSpec,
		)
	}
	runtimeMutex.Unlock()

	standaloneOption, cleanupSeed := prepareStandaloneOption(t, option)
	runtime := &Runtime{
		clientApp: meta.MustNewAppWithRandomId("vine.testkit", "0.0.0"),
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				runtime.stopAfterStartFailure()
				cleanupSeed()
				t.Fatalf(
					"start testkit standalone for %s failed: %v; "+
						"Vine app specs are process-level singletons and "+
						"cannot be created more than once in one test process",
					specType,
					r,
				)
			}
		}()
		runtime.app = standalone.NewWithOption[S](standaloneOption, appliers...)
		runtime.app.Start()
	}()

	t.Cleanup(func() {
		runtime.Stop()
		cleanupSeed()
	})
	runtime.registerClientApp(t)

	runtimeMutex.Lock()
	startedRuntimeSpec = specType
	runtimeMutex.Unlock()
	return runtime
}

// Stop unregisters the test client and gracefully stops the runtime.
func (r *Runtime) Stop() {
	if r.app != nil {
		r.unregisterClientApp()
		r.app.StopGracefully()
		r.app = nil
	}
}

func (r *Runtime) stopAfterStartFailure() {
	defer func() { _ = recover() }()
	r.Stop()
}

func prepareStandaloneOption(t testing.TB, option Option) (standalone.Option, func()) {
	t.Helper()

	seedYAMLFile := option.SeedYAMLFile
	cleanup := func() {}
	if len(option.ConfigOverrides) > 0 {
		var err error
		seedYAMLFile, cleanup, err = mergeSeedConfigOverrides(t, option.SeedYAMLFile, option.ConfigOverrides)
		if err != nil {
			t.Fatalf("prepare testkit seed yaml failed: %v", err)
		}
	}

	return standalone.Option{
		SeedYAMLFile: seedYAMLFile,
		SQLiteFile:   standaloneSQLiteFile(t),
	}, cleanup
}

func standaloneSQLiteFile(t testing.TB) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "vine-testkit-hub.sqlite")
}

func (r *Runtime) registerClientApp(t testing.TB) {
	t.Helper()

	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("register testkit client app failed: %v", err)
		}
	}()
	r.registryClient().Register(linkskeled.AppRegistration{
		ServiceHandlers: []linkskeled.ServiceHandlerRegistration{},
		WebHandlers:     []linkskeled.WebHandlerRegistration{},
		EventListeners:  []linkskeled.EventListenerRegistration{},
		TaskRunners:     []linkskeled.TaskRunnerRegistration{},
		DomainSchemas:   []skel.JSON{},
	}, rpc.WithTimeout(registerTimeout))
	r.clientAppRegistered = true
}

func (r *Runtime) unregisterClientApp() {
	if !r.clientAppRegistered {
		return
	}
	defer func() { _ = recover() }()
	r.registryClient().Unregister(rpc.WithTimeout(registerTimeout))
	r.clientAppRegistered = false
}

func (r *Runtime) registryClient() linkskeled.RegistryServiceClient {
	client := internalrpc.New(internalrpc.Option{
		Context:        rpc.NewContext(context.Background(), meta.InitialTrace(), r.clientApp, nil, meta.NewAbsentActor()),
		ClientApp:      r.clientApp,
		Logger:         logger.NewLogger(logger.GlobalOption()),
		ServerEndpoint: linkRpcInvokeEndpoint,
	})
	return linkskeled.NewRegistryServiceClient(linkskeled.NewRegistryServiceClientER(client))
}
