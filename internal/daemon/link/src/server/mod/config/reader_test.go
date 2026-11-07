package config

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
)

func TestAfterAppStopCancelsAllWatchesAndClearsSnapshots(t *testing.T) {
	reader := newTestReader(t, nil)
	client := &watchContextRecorder{ClientOps: reader.Client}
	reader.Client = client
	instance := registerTestAppInstance(reader, "22222222-2222-2222-2222-222222222222")
	for _, name := range []string{"demo.FeatureConfig", "demo.OtherConfig"} {
		reader.GetInstant(instance.AppInfo.InstanceId(), name)
	}
	require.Len(t, client.contexts, 2)

	reader.AfterAppStop()
	for _, ctx := range client.contexts {
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
	}
	reader.mutex.RLock()
	defer reader.mutex.RUnlock()
	assert.Empty(t, reader.instantConfigStatesByKey)
	assert.Empty(t, reader.configValuesByAppInstanceID)
}

func TestOnDestroyCancelsWatchAfterLastInstance(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	client := &watchContextRecorder{ClientOps: reader.Client}
	reader.Client = client
	first := registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
	second := registerTestAppInstance(reader, "22222222-2222-2222-2222-222222222222")
	reader.GetInstant(first.AppInfo.InstanceId(), "demo.FeatureConfig")
	reader.GetInstant(second.AppInfo.InstanceId(), "demo.FeatureConfig")
	require.Len(t, client.contexts, 1)

	reader.OnDestroy(first)
	assert.NoError(t, client.contexts[0].Err())
	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(second.AppInfo.InstanceId(), "demo.FeatureConfig"))
	reader.mutex.RLock()
	assert.NotContains(t, reader.configValuesByAppInstanceID, first.AppInfo.InstanceId())
	reader.mutex.RUnlock()

	reader.OnDestroy(second)
	assert.ErrorIs(t, client.contexts[0].Err(), context.Canceled)
	reader.mutex.RLock()
	defer reader.mutex.RUnlock()
	assert.Empty(t, reader.instantConfigStatesByKey)
	assert.Empty(t, reader.configValuesByAppInstanceID)
}

func TestInstantConfigReceivesSubscribedEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := newTestReader(t, nil)
		instance := registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
		name := "demo.FeatureConfig"
		assert.Empty(t, reader.GetInstant(instance.AppInfo.InstanceId(), name))
		client := reader.Client.(*watchtest.Client)
		key := watched.FormatConfigKey(name)
		client.Publish(watch.Event{Kind: watch.EventKindUpsert, Key: key, Value: marshalTestConfigValue(name, `{"enabled":true}`)})
		synctest.Wait()
		assert.Equal(t, `{"enabled":true}`, reader.GetInstant(instance.AppInfo.InstanceId(), name))
		client.Publish(watch.Event{Kind: watch.EventKindUpsert, Key: key, Value: "invalid"})
		synctest.Wait()
		assert.Equal(t, `{"enabled":true}`, reader.GetInstant(instance.AppInfo.InstanceId(), name))
		client.Publish(watch.Event{Kind: watch.EventKindDelete, Key: key})
		synctest.Wait()
		assert.Empty(t, reader.GetInstant(instance.AppInfo.InstanceId(), name))
	})
}

type watchContextRecorder struct {
	watch.ClientOps
	contexts []context.Context
}

func (c *watchContextRecorder) LoadAndSubscribe(ctx context.Context, key string, handle func(watch.Event)) (string, bool, watch.Subscription) {
	c.contexts = append(c.contexts, ctx)
	return c.ClientOps.LoadAndSubscribe(ctx, key, handle)
}

func TestInstantConfigThroughHubWatch(t *testing.T) {
	server, client := watchtest.NewServer(t, watch.LinkUsername)
	reader := &Reader{Context: t.Context(), Client: client, AppMinder: newTestMinder()}
	reader.DIInit()
	t.Cleanup(reader.AfterAppStop)
	instance := registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
	name := "demo.FeatureConfig"
	key := watched.FormatConfigKey(name)
	server.Set(key, marshalTestConfigValue(name, `{"enabled":false}`))
	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(instance.AppInfo.InstanceId(), name))
	server.SetAndNotify(key, marshalTestConfigValue(name, `{"enabled":true}`))
	require.Eventually(t, func() bool { return reader.GetInstant(instance.AppInfo.InstanceId(), name) == `{"enabled":true}` }, 2*time.Second, time.Millisecond)
	server.DeleteAndNotify(key)
	require.Eventually(t, func() bool { return reader.GetInstant(instance.AppInfo.InstanceId(), name) == "" }, 2*time.Second, time.Millisecond)
}
