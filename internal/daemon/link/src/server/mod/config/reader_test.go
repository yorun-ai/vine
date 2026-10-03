package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"testing"
	"testing/synctest"
	"time"
)

func TestAfterAppStopClearsRetainedStates(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	registerTestAppInstance(reader, "22222222-2222-2222-2222-222222222222")
	reader.GetInstant("22222222-2222-2222-2222-222222222222", "demo.FeatureConfig")

	reader.AfterAppStop()
	reader.mutex.RLock()
	assert.Empty(t, reader.instantConfigStatesByKey)
	reader.mutex.RUnlock()
}

func TestOnDestroyReleasesInstanceState(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	instance := newTestAppInstance("11111111-1111-1111-1111-111111111111")
	reader.OnSetup(instance)
	reader.GetInstant(instance.AppInfo.InstanceId(), "demo.FeatureConfig")

	reader.OnDestroy(instance)

	reader.mutex.RLock()
	_, ok := reader.instantConfigStatesByKey[watched.FormatConfigKey("demo.FeatureConfig")]
	reader.mutex.RUnlock()
	assert.False(t, ok)
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
		reader.OnDestroy(instance)
		synctest.Wait()
		client.Publish(watch.Event{Kind: watch.EventKindUpsert, Key: key, Value: marshalTestConfigValue(name, `{}`)})
		synctest.Wait()
		assert.Empty(t, reader.instantConfigStatesByKey)
	})
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
