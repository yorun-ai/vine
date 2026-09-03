package minder

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func newTestMinderWithHubClient(client *_RegistryServiceClient) *AppMinder {
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		App:                   mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
	}
	minder.DIInit()
	return minder
}

func TestStartHeartbeatReRegistersWhenHubLosesRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		appInfo := mustTestMetaApp()
		client := &_RegistryServiceClient{registered: false}
		minder := newTestMinderWithHubClient(client)
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
		assert.Equal(t, skel.NewUUID(uuid.MustParse(appInfo.InstanceId())), client.registrations[0].InstanceId)
	})
}

func TestStartHeartbeatSkipsWhenHubInprocModeEnabled(t *testing.T) {
	appInfo := mustTestMetaApp()
	client := &_RegistryServiceClient{}
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		App:                   appInfo,
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: client,
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
