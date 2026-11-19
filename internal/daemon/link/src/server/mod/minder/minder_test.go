package minder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _RegistryServiceClient struct {
	mutex         sync.Mutex
	registrations []hubskeled.AppRegistration
	unregistered  []uuid.UUID
	heartbeats    []hubskeled.AppStatus
	registered    bool
	onRegister    func(hubskeled.AppRegistration)
}

func (c *_RegistryServiceClient) Register(registration hubskeled.AppRegistration, _ivOpts ...client.InvokeOption) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.registrations = append(c.registrations, registration)
	if c.onRegister != nil {
		c.onRegister(registration)
	}
}

func (c *_RegistryServiceClient) Unregister(_ string, instanceId uuid.UUID, _ivOpts ...client.InvokeOption) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.unregistered = append(c.unregistered, instanceId)
}

func (c *_RegistryServiceClient) Heartbeat(status hubskeled.AppStatus, _ ...client.InvokeOption) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.heartbeats = append(c.heartbeats, status)
	return c.registered
}

func mustTestMetaApp() meta.App {
	app, err := meta.NewApp("demo.app", "1.0.0", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	if err != nil {
		panic(err)
	}
	return app
}

type _TestInfoServiceClient struct {
	mutex       sync.Mutex
	info        hubskeled.Info
	getInfoCall int
}

func (c *_TestInfoServiceClient) GetInfo(_ ...client.InvokeOption) hubskeled.Info {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.getInfoCall++
	return c.info
}

func (c *_TestInfoServiceClient) calls() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.getInfoCall
}

func newTestHubInfo(client *_TestInfoServiceClient) *hubinfo.HubInfo {
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize(false)
	component := &hubinfo.HubInfo{Flag: flags, InfoServiceClient: client}
	component.DIInit()
	return component
}

func newTestMinder(flagValue *flag.Flag, inprocFlag *app.InternalInprocFlag, client *_RegistryServiceClient) *AppMinder {
	if flagValue == nil {
		flagValue = &flag.Flag{}
	}
	if inprocFlag == nil {
		inprocFlag = &app.InternalInprocFlag{}
	}
	if client == nil {
		client = &_RegistryServiceClient{}
	}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  flagValue,
		CurrentApp:            mustTestMetaApp(),
		InprocFlag:            inprocFlag,
		RegistryServiceClient: client,
		HubInfo:               newTestHubInfo(new(_TestInfoServiceClient)),
	}
	minder.DIInit()
	return minder
}

func TestRegisterInstanceRegistersHubAndProfileState(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := newTestMinder(&flag.Flag{HubInprocMode: true}, &app.InternalInprocFlag{}, client)
	minder.RegisterInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "",
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}},
		WebHandlers:     []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
	})

	client.mutex.Lock()
	assert.Len(t, client.registrations, 1)
	assert.Equal(t, appInfo.InstanceId(), client.registrations[0].InstanceId)
	assert.Equal(t, "", client.registrations[0].Endpoint)
	assert.Equal(t, []hubskeled.ServiceHandlerRegistration{{
		ServiceSkelName: "demo.service.UserService",
		Endpoint:        "",
	}}, client.registrations[0].ServiceHandlers)
	assert.Equal(t, []hubskeled.WebHandlerRegistration{{
		WebSkelName: "default@demo.app",
		Endpoint:    "",
	}}, client.registrations[0].WebHandlers)
	assert.Empty(t, client.registrations[0].EventListeners)
	assert.Empty(t, client.registrations[0].TaskRunners)
	client.mutex.Unlock()

	instance, ok := minder.appInstance(appInfo.InstanceId())
	assert.True(t, ok)
	assert.Equal(t, "http://127.0.0.1:8080/console", instance.ConsoleEndpoint)
	assert.Equal(t, "", instance.IngressEndpoint)
	assert.Equal(t, []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}}, instance.ServiceHandlers)
	assert.Equal(t, []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}}, instance.WebHandlers)

	minder.AfterAppStop()
}

func TestUnregisterInstanceRemovesCatalogState(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := newTestMinder(&flag.Flag{HubInprocMode: true}, &app.InternalInprocFlag{}, client)

	minder.RegisterInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "",
		ServiceHandlers: []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}},
		WebHandlers:     []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
	})
	minder.UnregisterInstance(appInfo.InstanceId())

	client.mutex.Lock()
	assert.Equal(t, []uuid.UUID{appInfo.InstanceId()}, client.unregistered)
	client.mutex.Unlock()
	_, ok := minder.appInstance(appInfo.InstanceId())
	assert.False(t, ok)
}

type registrationMutator struct {
	_Mutator
	setup func()
}

func (m *registrationMutator) OnSetup(instance *AppInstance) {
	m._Mutator.OnSetup(instance)
	m.setup()
}

func TestRegisterInstanceRetriesAfterFailure(t *testing.T) {
	for _, stage := range []string{"setup", "hub"} {
		t.Run(stage, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				registry := new(_RegistryServiceClient)
				manager := newTestMinder(&flag.Flag{}, &app.InternalInprocFlag{Enabled: true}, registry)
				manager.Context = t.Context()
				defer manager.AfterAppStop()
				reg := AppRegistration{AppInfo: mustTestMetaApp()}
				entered := make(chan struct{})
				resume := make(chan struct{})
				var first sync.Once
				failOnce := func() {
					first.Do(func() {
						close(entered)
						<-resume
						panic("registration failed")
					})
				}
				mutator := &registrationMutator{setup: func() {}}
				if stage == "setup" {
					mutator.setup = failOnce
				} else {
					registry.onRegister = func(hubskeled.AppRegistration) { failOnce() }
				}
				manager.AddMutator(mutator)
				last := new(_Mutator)
				manager.AddMutator(last)
				results := make(chan any, 2)
				register := func() {
					defer func() { results <- recover() }()
					manager.RegisterInstance(reg)
				}
				go register()
				<-entered
				failed, _ := manager.appInstance(reg.AppInfo.InstanceId())
				go register()
				synctest.Wait()
				assert.Empty(t, results, "a duplicate must wait for the initial registration")
				close(resume)
				require.Equal(t, "registration failed", <-results)
				require.Equal(t, "registration failed", <-results)
				_, exists := manager.appInstance(reg.AppInfo.InstanceId())
				require.False(t, exists)
				require.Nil(t, failed.heartbeatCancel)
				require.False(t, failed.TryStartWork())
				require.Len(t, mutator.drainedIDs, 1)
				require.Len(t, mutator.destroyedIDs, 1)
				if stage == "setup" {
					require.Empty(t, last.destroyedIDs)
					require.Empty(t, registry.unregistered)
				} else {
					require.Len(t, last.destroyedIDs, 1)
					require.Equal(t, []uuid.UUID{reg.AppInfo.InstanceId()}, registry.unregistered)
				}
				before := len(registry.registrations)
				manager.RegisterInstance(reg)
				manager.RegisterInstance(reg)
				require.Len(t, registry.registrations, before+1)
				require.Len(t, mutator.setupIDs, 2)
			})
		})
	}
}

func TestRegisterInstanceStartsRuntimeLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		appInfo := mustTestMetaApp()
		client := &_RegistryServiceClient{registered: true}
		minder := newTestMinder(&flag.Flag{}, &app.InternalInprocFlag{}, client)
		minder.Context = t.Context()
		prevHeartbeatInterval := heartbeatInterval
		prevHealthcheckInterval := healthcheckInterval
		heartbeatInterval = 10 * time.Millisecond
		healthcheckInterval = 10 * time.Millisecond
		defer func() {
			heartbeatInterval = prevHeartbeatInterval
			healthcheckInterval = prevHealthcheckInterval
		}()

		minder.RegisterInstance(AppRegistration{
			AppInfo:         appInfo,
			ConsoleEndpoint: "http://127.0.0.1:8080/console",
			IngressEndpoint: "http://127.0.0.1:8081",
		})
		defer minder.AfterAppStop()

		instance, ok := minder.appInstance(appInfo.InstanceId())
		assert.True(t, ok)
		assert.NotNil(t, instance.heartbeatCancel)
		assert.NotNil(t, instance.healthcheckCancel)
		synctest.Sleep(10 * time.Millisecond)

		client.mutex.Lock()
		defer client.mutex.Unlock()
		assert.NotEmpty(t, client.heartbeats)
	})
}

func TestRegisterInstanceStartsLifecycleBeforeHubRegistration(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{registered: true}
	minder := newTestMinder(&flag.Flag{}, &app.InternalInprocFlag{}, client)
	registerObserved := make(chan struct{}, 1)
	client.onRegister = func(hubskeled.AppRegistration) {
		instance, ok := minder.appInstance(appInfo.InstanceId())
		assert.True(t, ok)
		assert.NotNil(t, instance.heartbeatCancel)
		assert.NotNil(t, instance.healthcheckCancel)
		registerObserved <- struct{}{}
	}

	minder.RegisterInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "http://127.0.0.1:8081",
	})
	defer minder.AfterAppStop()

	select {
	case <-registerObserved:
	default:
		t.Fatal("hub registration did not observe started lifecycle")
	}
}

func TestAfterAppStopStopsInstanceLifecycles(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{registered: true}
	minder := newTestMinder(&flag.Flag{}, &app.InternalInprocFlag{}, client)
	prevHeartbeatInterval := heartbeatInterval
	prevHealthcheckInterval := healthcheckInterval
	heartbeatInterval = 10 * time.Millisecond
	healthcheckInterval = 10 * time.Millisecond
	defer func() {
		heartbeatInterval = prevHeartbeatInterval
		healthcheckInterval = prevHealthcheckInterval
	}()

	minder.RegisterInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
		IngressEndpoint: "http://127.0.0.1:8081",
	})

	instance, ok := minder.appInstance(appInfo.InstanceId())
	assert.True(t, ok)
	assert.NotNil(t, instance.heartbeatCancel)
	assert.NotNil(t, instance.healthcheckCancel)

	minder.AfterAppStop()

	assert.Nil(t, instance.heartbeatCancel)
	assert.Nil(t, instance.healthcheckCancel)
}

func TestUnregisterFailureStillDrainsAndRemovesInstance(t *testing.T) {
	for _, inproc := range []bool{false, true} {
		name := "network"
		if inproc {
			name = "inproc"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				minder := newTestMinder(&flag.Flag{HubInprocMode: inproc}, &app.InternalInprocFlag{Enabled: true}, nil)
				minder.Context = t.Context()
				minder.RegistryServiceClient = &shutdownRegistryClient{unregister: func() { panic("hub unavailable") }}
				observer := new(_Mutator)
				minder.AddMutator(observer)
				instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
				minder.addInstance(instance)
				instance.startHeartbeat()
				require.PanicsWithValue(t, "hub unavailable", func() { minder.UnregisterInstance(instance.AppInfo.InstanceId()) })
				require.True(t, instance.isStopping())
				require.Nil(t, instance.heartbeatCancel)
				require.False(t, instance.TryStartWork())
				_, ok := minder.appInstance(instance.AppInfo.InstanceId())
				require.False(t, ok)
				require.Len(t, observer.drainedIDs, 1)
				require.Len(t, observer.destroyedIDs, 1)
			})
		})
	}
}

func TestConcurrentUnregisterWaitsForSameDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(_RegistryServiceClient)
		minder := newTestMinder(&flag.Flag{HubInprocMode: true}, &app.InternalInprocFlag{Enabled: true}, client)
		minder.Context = t.Context()
		observer := new(_Mutator)
		minder.AddMutator(observer)
		instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
		minder.addInstance(instance)
		require.True(t, instance.TryStartWork())
		first := make(chan struct{})
		second := make(chan struct{})
		go func() { defer close(first); minder.UnregisterInstance(instance.AppInfo.InstanceId()) }()
		synctest.Wait()
		require.True(t, instance.isStopping())
		// Propagation grace still admits traffic before drain starts.
		require.True(t, instance.TryStartWork())
		instance.FinishWork()
		go func() { defer close(second); minder.UnregisterInstance(instance.AppInfo.InstanceId()) }()
		synctest.Sleep(unregisterPropagationGrace)
		require.False(t, instance.TryStartWork())
		select {
		case <-first:
			t.Fatal("unregister returned before drain")
		default:
		}
		select {
		case <-second:
			t.Fatal("concurrent unregister returned before drain")
		default:
		}
		instance.FinishWork()
		<-first
		<-second
		require.Len(t, client.unregistered, 1)
		require.Len(t, observer.drainedIDs, 1)
		require.Len(t, observer.destroyedIDs, 1)
	})
}

func TestUnregisterHTTPFailureRemovesLocalInstance(t *testing.T) {
	requests := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- struct{}{}
		http.Error(w, "hub unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	minder := newTestMinder(nil, &app.InternalInprocFlag{Enabled: true}, nil)
	minder.Context = t.Context()
	minder.RegistryServiceClient = hubskeled.NewRegistryServiceClient(hubskeled.NewRegistryServiceClientER(client.New(client.Option{
		Context:        meta.NewContext(t.Context(), meta.InitialTrace(), nil, meta.NewAbsentActor()),
		ClientApp:      mustTestMetaApp(),
		Logger:         logger.New("test:minder:shutdown"),
		ServerEndpoint: server.URL,
		Transport:      http.DefaultTransport,
	})))
	instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
	minder.addInstance(instance)
	instance.startHeartbeat()
	defer instance.stopHeartbeat()
	require.Panics(t, func() { minder.UnregisterInstance(instance.AppInfo.InstanceId()) })
	select {
	case <-requests:
	default:
		t.Fatal("Hub did not receive HTTP unregister")
	}
	require.Nil(t, instance.heartbeatCancel)
	_, ok := minder.appInstance(instance.AppInfo.InstanceId())
	require.False(t, ok)
}

type shutdownTimeoutTransport struct{}

func (shutdownTimeoutTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	<-request.Context().Done()
	return nil, request.Context().Err()
}

func TestUnregisterTimeoutStillCompletesCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		minder := newTestMinder(nil, &app.InternalInprocFlag{Enabled: true}, nil)
		minder.Context = t.Context()
		minder.RegistryServiceClient = hubskeled.NewRegistryServiceClient(hubskeled.NewRegistryServiceClientER(client.New(client.Option{
			Context:        meta.NewContext(t.Context(), meta.InitialTrace(), nil, meta.NewAbsentActor()),
			ClientApp:      mustTestMetaApp(),
			Logger:         logger.New("test:minder:shutdown"),
			ServerEndpoint: "http://hub.invalid",
			Transport:      shutdownTimeoutTransport{},
		})))
		instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
		minder.addInstance(instance)
		start := time.Now()
		require.Panics(t, func() { minder.UnregisterInstance(instance.AppInfo.InstanceId()) })
		require.Equal(t, hubRegistryTimeout+unregisterPropagationGrace, time.Since(start))
		_, ok := minder.appInstance(instance.AppInfo.InstanceId())
		require.False(t, ok)
	})
}
