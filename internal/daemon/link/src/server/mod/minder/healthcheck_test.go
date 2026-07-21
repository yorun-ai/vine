package minder

import (
	"context"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _HealthcheckConsoleClient struct {
	ping func() ex.Error
}

func (c *_HealthcheckConsoleClient) Ping(...rpcclient.InvokeOption) ex.Error {
	return c.ping()
}

type _HealthcheckFailureHooks struct {
	mutex        sync.Mutex
	failureCount int
}

func (h *_HealthcheckFailureHooks) onHealthcheckFailure() {
	h.mutex.Lock()
	h.failureCount++
	h.mutex.Unlock()
}

func newTestHealthcheckMinder(ctx context.Context) *AppMinder {
	minder := &AppMinder{
		Context:               ctx,
		Flag:                  &flag.Flag{},
		App:                   mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: &_RegistryServiceClient{},
	}
	minder.DIInit()
	return minder
}

func TestHealthcheckAutoUnregistersAfterConsecutiveFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldFactory := newConsoleServiceClient
	defer func() {
		newConsoleServiceClient = oldFactory
	}()

	newConsoleServiceClient = func(context.Context, runtime.App, string) appskeled.ConsoleServiceClientER {
		return &_HealthcheckConsoleClient{
			ping: func() ex.Error {
				return ex.New(ex.ServerUnreachable, "down")
			},
		}
	}

	hooks := &_HealthcheckFailureHooks{}
	minder := newTestHealthcheckMinder(ctx)
	appInfo := mustTestMetaApp()
	instance := minder.newAppInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
	})
	minder.addInstance(instance)
	prevInterval := healthcheckInterval
	prevMaxFailedCount := healthcheckMaxFailedCount
	healthcheckInterval = 10 * time.Millisecond
	healthcheckMaxFailedCount = 3
	defer func() {
		healthcheckInterval = prevInterval
		healthcheckMaxFailedCount = prevMaxFailedCount
	}()

	instance.startHealthcheck(hooks.onHealthcheckFailure)
	assert.Eventually(t, func() bool {
		hooks.mutex.Lock()
		defer hooks.mutex.Unlock()
		return hooks.failureCount == 1
	}, time.Second, 10*time.Millisecond)
	assert.Nil(t, instance.healthcheckCancel)
}

func TestHealthcheckIgnoresPingTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldFactory := newConsoleServiceClient
	defer func() {
		newConsoleServiceClient = oldFactory
	}()

	newConsoleServiceClient = func(context.Context, runtime.App, string) appskeled.ConsoleServiceClientER {
		return &_HealthcheckConsoleClient{
			ping: func() ex.Error {
				return ex.New(ex.InvocationTimeout, "timeout")
			},
		}
	}

	hooks := &_HealthcheckFailureHooks{}
	minder := newTestHealthcheckMinder(ctx)
	appInfo := mustTestMetaApp()
	instance := minder.newAppInstance(AppRegistration{
		AppInfo:         appInfo,
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
	})
	minder.addInstance(instance)
	prevInterval := healthcheckInterval
	prevMaxFailedCount := healthcheckMaxFailedCount
	healthcheckInterval = 10 * time.Millisecond
	healthcheckMaxFailedCount = 1
	defer func() {
		healthcheckInterval = prevInterval
		healthcheckMaxFailedCount = prevMaxFailedCount
	}()

	instance.startHealthcheck(hooks.onHealthcheckFailure)
	assert.Eventually(t, func() bool {
		return instance.healthcheckCancel != nil
	}, time.Second, 10*time.Millisecond)
	instance.stopHealthcheck()

	hooks.mutex.Lock()
	defer hooks.mutex.Unlock()
	assert.Zero(t, hooks.failureCount)
}

func TestStartHealthcheckSkipsWhenInprocModeEnabled(t *testing.T) {
	oldFactory := newConsoleServiceClient
	defer func() {
		newConsoleServiceClient = oldFactory
	}()

	called := false
	newConsoleServiceClient = func(context.Context, runtime.App, string) appskeled.ConsoleServiceClientER {
		called = true
		return &_HealthcheckConsoleClient{
			ping: func() ex.Error {
				return nil
			},
		}
	}

	minder := newTestMinder(&flag.Flag{}, &app.InternalInprocFlag{Enabled: true}, &_RegistryServiceClient{})
	instance := minder.newAppInstance(AppRegistration{
		AppInfo:         mustTestMetaApp(),
		ConsoleEndpoint: "http://127.0.0.1:8080/console",
	})

	instance.startHealthcheck(func() {})

	assert.Nil(t, instance.healthcheckCancel)
	assert.False(t, called)
}
