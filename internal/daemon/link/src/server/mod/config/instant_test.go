package config

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
)

func TestHandleInstantConfigEventRefreshesInstantValueForAllInstances(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})

	firstAppInstanceID := "11111111-1111-1111-1111-111111111111"
	secondAppInstanceID := "22222222-2222-2222-2222-222222222222"
	registerTestAppInstance(reader, uuid.MustParse(firstAppInstanceID))
	registerTestAppInstance(reader, uuid.MustParse(secondAppInstanceID))
	reader.GetInstant(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig")
	reader.GetInstant(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig")

	key := watched.FormatConfigKey("demo.FeatureConfig")
	reader.handleInstantConfigEvent(key, reader.instantConfigStatesByKey[key], hubwatch.Event{
		Kind:  hubwatch.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	})

	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig"))
	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig"))
}

func TestGetInstantInitializesNewSnapshotFromStateValue(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	firstAppInstanceID := "11111111-1111-1111-1111-111111111111"
	secondAppInstanceID := "22222222-2222-2222-2222-222222222222"
	registerTestAppInstance(reader, uuid.MustParse(firstAppInstanceID))
	registerTestAppInstance(reader, uuid.MustParse(secondAppInstanceID))
	reader.GetInstant(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig")
	key := watched.FormatConfigKey("demo.FeatureConfig")
	reader.handleInstantConfigEvent(key, reader.instantConfigStatesByKey[key], hubwatch.Event{
		Kind:  hubwatch.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	})

	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig"))
}

type instantWatchStub struct {
	hubwatch.ClientOps
	subscribe func(context.Context, string, func(hubwatch.Event)) (string, bool, hubwatch.Subscription)
}

func (s *instantWatchStub) LoadAndSubscribe(ctx context.Context, key string, handle func(hubwatch.Event)) (string, bool, hubwatch.Subscription) {
	return s.subscribe(ctx, key, handle)
}

func TestInstantSubscriptionFailureAllowsRetry(t *testing.T) {
	const name = "demo.FeatureConfig"
	client := watchtest.New(t, map[string]string{watched.FormatConfigKey(name): marshalTestConfigValue(name, "1")})
	stub := &instantWatchStub{ClientOps: client}
	reader := &Reader{Context: t.Context(), Client: stub, AppMinder: newTestMinder()}
	reader.DIInit()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	registerTestAppInstance(reader, id)
	var failedCtx context.Context
	var unlockedDuringLoad bool
	stub.subscribe = func(ctx context.Context, key string, handle func(hubwatch.Event)) (string, bool, hubwatch.Subscription) {
		failedCtx = ctx
		unlockedDuringLoad = reader.mutex.TryRLock()
		if unlockedDuringLoad {
			reader.mutex.RUnlock()
		}
		panic("Watch unavailable")
	}
	require.PanicsWithValue(t, "Watch unavailable", func() { reader.GetInstant(id, name) })
	require.True(t, unlockedDuringLoad, "a Watch load must leave the reader available")
	require.ErrorIs(t, failedCtx.Err(), context.Canceled)
	stub.subscribe = client.LoadAndSubscribe
	t.Cleanup(reader.AfterAppStop)
	require.Equal(t, "1", reader.GetInstant(id, name))
}

func TestInstantCallbackFromPreviousSubscriptionIsIgnored(t *testing.T) {
	const name = "demo.FeatureConfig"
	key := watched.FormatConfigKey(name)
	reader := newTestReader(t, nil)
	client := reader.Client.(*watchtest.Client)
	var handlers []func(hubwatch.Event)
	reader.Client = &instantWatchStub{
		ClientOps: client,
		subscribe: func(ctx context.Context, key string, handle func(hubwatch.Event)) (string, bool, hubwatch.Subscription) {
			handlers = append(handlers, handle)
			return client.LoadAndSubscribe(ctx, key, handle)
		},
	}
	first := registerTestAppInstance(reader, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	reader.GetInstant(first.AppInfo.InstanceId(), name)
	reader.OnDestroy(first)
	client.SetValue(key, marshalTestConfigValue(name, "2"))
	second := registerTestAppInstance(reader, uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	require.Equal(t, "2", reader.GetInstant(second.AppInfo.InstanceId(), name))
	require.Len(t, handlers, 2)
	// A callback that already entered the dispatcher can finish after cancellation.
	handlers[0](hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	require.Equal(t, "2", reader.GetInstant(second.AppInfo.InstanceId(), name))
	handlers[1](hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	require.Empty(t, reader.GetInstant(second.AppInfo.InstanceId(), name))
}
