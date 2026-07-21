package config

import (
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleInstantConfigEventIgnoresInvalidConfigValue(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)
	reader.GetInstant(appInstanceID, "demo.FeatureConfig")

	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Value: "not-a-config-value",
	})

	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(appInstanceID, "demo.FeatureConfig"))
}

func TestHandleInstantConfigEventRefreshesInstantValueForAllInstances(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})

	firstAppInstanceID := "11111111-1111-1111-1111-111111111111"
	secondAppInstanceID := "22222222-2222-2222-2222-222222222222"
	registerTestAppInstance(reader, firstAppInstanceID)
	registerTestAppInstance(reader, secondAppInstanceID)
	reader.GetInstant(firstAppInstanceID, "demo.FeatureConfig")
	reader.GetInstant(secondAppInstanceID, "demo.FeatureConfig")

	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	})

	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(firstAppInstanceID, "demo.FeatureConfig"))
	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(secondAppInstanceID, "demo.FeatureConfig"))
}

func TestGetInstantInitializesNewSnapshotFromStateValue(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	firstAppInstanceID := "11111111-1111-1111-1111-111111111111"
	secondAppInstanceID := "22222222-2222-2222-2222-222222222222"
	registerTestAppInstance(reader, firstAppInstanceID)
	registerTestAppInstance(reader, secondAppInstanceID)
	reader.GetInstant(firstAppInstanceID, "demo.FeatureConfig")
	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	})

	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(secondAppInstanceID, "demo.FeatureConfig"))
}

func TestHandleInstantConfigEventDeletesInstantValueOnDeleteEvent(t *testing.T) {
	reader := newTestReader(nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)
	reader.GetInstant(appInstanceID, "demo.FeatureConfig")
	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	})

	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{Kind: hubredis.EventKindDelete})

	assert.Equal(t, "", reader.GetInstant(appInstanceID, "demo.FeatureConfig"))
}

func TestHandleInstantConfigEventRecoversValueAfterInitialMiss(t *testing.T) {
	reader := newTestReader(nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)

	assert.Equal(t, "", reader.GetInstant(appInstanceID, "demo.FeatureConfig"))

	reader.handleInstantConfigEvent(redised.FormatConfigKey("demo.FeatureConfig"), hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	})

	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(appInstanceID, "demo.FeatureConfig"))
}

func TestReleaseInstantConfigStateByInstanceRemovesRetainedRef(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)
	reader.GetInstant(appInstanceID, "demo.FeatureConfig")

	reader.releaseInstantConfigStateByInstance(appInstanceID)

	reader.mutex.RLock()
	_, ok := reader.instantConfigStatesByKey[redised.FormatConfigKey("demo.FeatureConfig")]
	reader.mutex.RUnlock()
	assert.False(t, ok)
}

func TestReleaseInstantConfigStateByInstanceKeepsConfigWhileRetainedByAnotherApp(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	firstAppInstanceID := "11111111-1111-1111-1111-111111111111"
	secondAppInstanceID := "22222222-2222-2222-2222-222222222222"
	registerTestAppInstance(reader, firstAppInstanceID)
	registerTestAppInstance(reader, secondAppInstanceID)
	reader.GetInstant(firstAppInstanceID, "demo.FeatureConfig")
	reader.GetInstant(secondAppInstanceID, "demo.FeatureConfig")

	reader.releaseInstantConfigStateByInstance(firstAppInstanceID)

	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(secondAppInstanceID, "demo.FeatureConfig"))
}
