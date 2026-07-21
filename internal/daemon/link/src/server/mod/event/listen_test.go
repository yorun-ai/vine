package event

import (
	"context"
	"fmt"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const testPathEvent = "/event"

func TestManagerRegistersListenerAndDispatchesEvent(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	hooks := &_ManagerDispatchHooks{}
	newAppEventServiceClient = func(context.Context, runtime.App, string, eventspec.NATSMessage) appskeled.EventServiceClientER {
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				hooks.mutex.Lock()
				hooks.events = append(hooks.events, on)
				hooks.mutex.Unlock()
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		hooks.mutex.Lock()
		hooks.timeout = timeout
		hooks.callCount++
		hooks.mutex.Unlock()
		return client.OnEvent(on)
	}

	appInfo, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	endpoint := testLocalAppEndpoint(8080)
	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       appInfo,
		EventEndpoint: endpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     2500,
			Concurrency:   1,
		}},
	})

	manager.EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	require.Eventually(t, func() bool {
		hooks.mutex.Lock()
		defer hooks.mutex.Unlock()
		return len(hooks.events) == 1
	}, 2*time.Second, 10*time.Millisecond)

	hooks.mutex.Lock()
	defer hooks.mutex.Unlock()
	assert.Equal(t, time.Duration(2500)*time.Millisecond, hooks.timeout)
	assert.Equal(t, 1, hooks.callCount)
	assert.Equal(t, "demo.user.UserCreatedEvent", hooks.events[0].EventSkelName)
	assert.Equal(t, `{"userId":"u1"}`, hooks.events[0].EventJson)
	assert.Equal(t, "launcher.app", hooks.events[0].Metadata.AppName)
}

func TestManagerLimitsDispatchConcurrency(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	hooks := &_ManagerDispatchHooks{
		startedChan: make(chan struct{}, 2),
		releaseChan: make(chan struct{}),
	}
	newAppEventServiceClient = func(context.Context, runtime.App, string, eventspec.NATSMessage) appskeled.EventServiceClientER {
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				hooks.mutex.Lock()
				hooks.events = append(hooks.events, on)
				hooks.mutex.Unlock()
				hooks.startedChan <- struct{}{}
				<-hooks.releaseChan
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		return client.OnEvent(on)
	}

	appInfo, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	endpoint := testLocalAppEndpoint(8080)
	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       appInfo,
		EventEndpoint: endpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})

	firstEmission := skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	}
	secondEmission := firstEmission
	secondEmission.Metadata.TraceId = "trace-2"
	secondEmission.EventJson = `{"userId":"u2"}`

	manager.EmitEvent(firstEmission)
	manager.EmitEvent(secondEmission)

	select {
	case <-hooks.startedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("first event dispatch timeout")
	}

	select {
	case <-hooks.startedChan:
		t.Fatal("second event started before first event finished")
	case <-time.After(100 * time.Millisecond):
	}

	hooks.releaseChan <- struct{}{}

	select {
	case <-hooks.startedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("second event dispatch timeout")
	}

	hooks.releaseChan <- struct{}{}

	require.Eventually(t, func() bool {
		hooks.mutex.Lock()
		defer hooks.mutex.Unlock()
		return len(hooks.events) == 2
	}, 2*time.Second, 10*time.Millisecond)
}

func TestManagerFansOutByAppName(t *testing.T) {
	managers, cleanup := newTestManagers(t, 2)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{}
	secondHooks := &_ManagerDispatchHooks{}
	newAppEventServiceClient = func(_ context.Context, _ runtime.App, endpoint string, _ eventspec.NATSMessage) appskeled.EventServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				targetHooks.mutex.Lock()
				targetHooks.events = append(targetHooks.events, on)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		return client.OnEvent(on)
	}

	firstAppInfo, err := meta.NewApp("demo.first", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.second", "1.0.0", "22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)

	firstEndpoint := testLocalAppEndpoint(8080)
	managers[0].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       firstAppInfo,
		EventEndpoint: firstEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})
	secondEndpoint := testLocalAppEndpoint(8081)
	managers[1].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       secondAppInfo,
		EventEndpoint: secondEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})

	managers[0].EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	require.Eventually(t, func() bool {
		firstHooks.mutex.Lock()
		firstCount := firstHooks.callCount
		firstHooks.mutex.Unlock()
		secondHooks.mutex.Lock()
		secondCount := secondHooks.callCount
		secondHooks.mutex.Unlock()
		return firstCount == 1 && secondCount == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestManagerCompetesAcrossSameAppNameInstances(t *testing.T) {
	managers, cleanup := newTestManagers(t, 2)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{}
	secondHooks := &_ManagerDispatchHooks{}
	newAppEventServiceClient = func(_ context.Context, _ runtime.App, endpoint string, _ eventspec.NATSMessage) appskeled.EventServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				targetHooks.mutex.Lock()
				targetHooks.events = append(targetHooks.events, on)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		return client.OnEvent(on)
	}

	firstAppInfo, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.app", "1.0.0", "22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)

	firstEndpoint := testLocalAppEndpoint(8080)
	managers[0].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       firstAppInfo,
		EventEndpoint: firstEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})
	secondEndpoint := testLocalAppEndpoint(8081)
	managers[1].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       secondAppInfo,
		EventEndpoint: secondEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})

	managers[0].EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	require.Eventually(t, func() bool {
		firstHooks.mutex.Lock()
		firstCount := firstHooks.callCount
		firstHooks.mutex.Unlock()
		secondHooks.mutex.Lock()
		secondCount := secondHooks.callCount
		secondHooks.mutex.Unlock()
		return firstCount+secondCount == 1
	}, 2*time.Second, 10*time.Millisecond)

	firstHooks.mutex.Lock()
	firstCount := firstHooks.callCount
	firstHooks.mutex.Unlock()
	secondHooks.mutex.Lock()
	secondCount := secondHooks.callCount
	secondHooks.mutex.Unlock()
	assert.Equal(t, 1, firstCount+secondCount)
}

func testLocalAppEndpoint(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
