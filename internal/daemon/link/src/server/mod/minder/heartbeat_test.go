package minder

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func newTestMinderWithHubClient(client *_RegistryServiceClient, infoClient *_TestInfoServiceClient) *AppMinder {
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		CurrentApp:            mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
		HubInfo:               newTestHubInfo(infoClient),
	}
	minder.DIInit()
	return minder
}

func TestStartHeartbeatReRegistersWhenHubLosesRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		appInfo := mustTestMetaApp()
		client := &_RegistryServiceClient{registered: false}
		minder := newTestMinderWithHubClient(client, new(_TestInfoServiceClient))
		minder.Context = t.Context()
		instance := minder.newAppInstance(AppRegistration{
			AppInfo:         appInfo,
			ConsoleEndpoint: "http://127.0.0.1:8080/console",
			IngressEndpoint: "http://127.0.0.1:8081",
			ServiceHandlers: []skeled.ServiceHandlerRegistration{{ServiceSkelName: "demo.service.UserService"}},
			WebHandlers:     []skeled.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		})
		prev := heartbeatInterval
		heartbeatInterval = 10 * time.Millisecond
		defer func() {
			heartbeatInterval = prev
		}()

		instance.startHeartbeat()
		defer instance.stopHeartbeat()
		synctest.Sleep(10 * time.Millisecond)

		client.mutex.Lock()
		defer client.mutex.Unlock()
		assert.NotEmpty(t, client.heartbeats)
		require.NotEmpty(t, client.registrations)
		assert.Equal(t, appInfo.InstanceId(), client.registrations[0].InstanceId)
	})
}

func TestStartHeartbeatRefreshesHubInfoWhenHubLosesRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &_RegistryServiceClient{registered: false}
		infoClient := &_TestInfoServiceClient{}
		minder := newTestMinderWithHubClient(client, infoClient)
		minder.Context = t.Context()
		instance := minder.newAppInstance(AppRegistration{
			AppInfo:         mustTestMetaApp(),
			IngressEndpoint: "http://127.0.0.1:8081",
		})
		prev := heartbeatInterval
		heartbeatInterval = 10 * time.Millisecond
		defer func() {
			heartbeatInterval = prev
		}()

		// The initial Hub information lookup, before any heartbeat ran.
		assert.Equal(t, 1, infoClient.calls())

		instance.startHeartbeat()
		defer instance.stopHeartbeat()
		synctest.Sleep(10 * time.Millisecond)

		// A missed heartbeat refreshes Hub information, which is what repairs a
		// moved MQ or lock endpoint, before registering again.
		assert.GreaterOrEqual(t, infoClient.calls(), 2)
	})
}

func TestStartHeartbeatSkipsWhenHubInprocModeEnabled(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		CurrentApp:            appInfo,
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
		HubInfo:               newTestHubInfo(new(_TestInfoServiceClient)),
	}
	minder.DIInit()
	instance := minder.newAppInstance(AppRegistration{AppInfo: appInfo})

	instance.startHeartbeat()

	assert.Nil(t, instance.heartbeatCancel)
	client.mutex.Lock()
	defer client.mutex.Unlock()
	assert.Empty(t, client.heartbeats)
	assert.Empty(t, client.registrations)
}

type shutdownRegistryClient struct {
	hubskeled.RegistryServiceClient
	heartbeat  func() bool
	register   func()
	unregister func()
}

func (c *shutdownRegistryClient) Heartbeat(hubskeled.AppStatus, ...rpcclient.InvokeOption) bool {
	return c.heartbeat()
}
func (c *shutdownRegistryClient) Register(hubskeled.AppRegistration, ...rpcclient.InvokeOption) {
	c.register()
}
func (c *shutdownRegistryClient) Unregister(string, uuid.UUID, ...rpcclient.InvokeOption) {
	c.unregister()
}

type shutdownInfoClient struct {
	hubskeled.InfoServiceClient
	refresh func() hubskeled.Info
}

func (c *shutdownInfoClient) GetInfo(...rpcclient.InvokeOption) hubskeled.Info { return c.refresh() }

func TestUnregisterDuringHeartbeatRecovery(t *testing.T) {
	for _, stage := range []string{"heartbeat", "refresh", "register"} {
		t.Run(stage, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			operationDone := make(chan struct{})
			var releaseOnce atomic.Bool
			unblock := func() {
				if releaseOnce.CompareAndSwap(false, true) {
					close(release)
				}
			}
			defer unblock()
			block := func() { close(entered); <-release; close(operationDone) }
			var registrations atomic.Int32
			var unregistrations atomic.Int32
			client := &shutdownRegistryClient{
				heartbeat: func() bool {
					if stage == "heartbeat" {
						block()
					}
					return false
				},
				register: func() {
					if stage == "register" {
						block()
					}
					registrations.Add(1)
				},
				unregister: func() { unregistrations.Add(1) },
			}
			minder := newTestMinderWithHubClient(new(_RegistryServiceClient), new(_TestInfoServiceClient))
			minder.Context = t.Context()
			minder.RegistryServiceClient = client
			if stage == "refresh" {
				minder.HubInfo.InfoServiceClient = &shutdownInfoClient{refresh: func() hubskeled.Info { block(); return hubskeled.Info{} }}
			}
			instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
			minder.addInstance(instance)
			oldInterval := heartbeatInterval
			heartbeatInterval = time.Millisecond
			defer func() { heartbeatInterval = oldInterval }()
			instance.startHeartbeat()
			defer instance.stopHeartbeat()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("heartbeat recovery did not start")
			}
			done := make(chan struct{})
			go func() { defer close(done); minder.UnregisterInstance(instance.AppInfo.InstanceId()) }()
			require.Eventually(t, instance.isStopping, time.Second, time.Millisecond)
			if stage == "refresh" {
				// A blocked read-only Hub refresh must not hold up App shutdown.
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("unregister blocked on Hub refresh")
				}
			} else {
				require.Zero(t, unregistrations.Load(), "unregister must follow the in-flight Hub mutation")
			}
			unblock()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("unregister did not finish")
			}
			<-operationDone
			// Synchronize with any final registration attempt from the recovery callback.
			instance.hubMutex.Lock()
			instance.hubMutex.Unlock()
			if stage == "register" {
				require.Equal(t, int32(1), registrations.Load())
			} else {
				require.Zero(t, registrations.Load())
			}
			require.Equal(t, int32(1), unregistrations.Load())
			_, ok := minder.appInstance(instance.AppInfo.InstanceId())
			require.False(t, ok)
		})
	}
}
