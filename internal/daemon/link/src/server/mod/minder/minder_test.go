package minder

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _RegistryServiceClient struct {
	mutex         sync.Mutex
	registrations []hubskeled.AppRegistration
	unregistered  []skel.UUID
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

func (c *_RegistryServiceClient) Unregister(_ string, instanceId skel.UUID, _ivOpts ...client.InvokeOption) {
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
	app, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		panic(err)
	}
	return app
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
		App:                   mustTestMetaApp(),
		InprocFlag:            inprocFlag,
		RegistryServiceClient: client,
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
	assert.Equal(t, skel.NewUUID(uuid.MustParse(appInfo.InstanceId())), client.registrations[0].InstanceId)
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
	assert.Equal(t, []skel.UUID{skel.NewUUID(uuid.MustParse(appInfo.InstanceId()))}, client.unregistered)
	client.mutex.Unlock()
	_, ok := minder.appInstance(appInfo.InstanceId())
	assert.False(t, ok)
}

func TestRegisterInstanceStartsRuntimeLifecycle(t *testing.T) {
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
	defer minder.AfterAppStop()

	instance, ok := minder.appInstance(appInfo.InstanceId())
	assert.True(t, ok)
	assert.NotNil(t, instance.heartbeatCancel)
	assert.NotNil(t, instance.healthcheckCancel)
	assert.Eventually(t, func() bool {
		client.mutex.Lock()
		defer client.mutex.Unlock()
		return len(client.heartbeats) > 0
	}, time.Second, 10*time.Millisecond)
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

	assert.Eventually(t, func() bool {
		return len(registerObserved) > 0
	}, time.Second, 10*time.Millisecond)
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
