package config

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstantLoadsAndGetsValue(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")

	value := reader.GetInstant("11111111-1111-1111-1111-111111111111", "demo.FeatureConfig")
	assert.Equal(t, `{"enabled":true}`, value)
}

func TestGetInstantReturnsEmptyWhenConfigDoesNotExist(t *testing.T) {
	reader := newTestReader(nil)
	registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
	value := reader.GetInstant("11111111-1111-1111-1111-111111111111", "demo.FeatureConfig")
	assert.Equal(t, "", value)
}

func TestGetInstantReturnsEmptyWhenStoredValueIsInvalid(t *testing.T) {
	reader := newTestReader(nil)
	registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
	reader.Client.SetValue(redised.FormatConfigKey("demo.FeatureConfig"), "not-a-config-value")

	value := reader.GetInstant("11111111-1111-1111-1111-111111111111", "demo.FeatureConfig")

	assert.Equal(t, "", value)
	assert.Equal(t, "", configValueSnapshot(reader, "demo.FeatureConfig", "11111111-1111-1111-1111-111111111111"))
}

func TestGetEternalDoesNotStartWatcher(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	registerTestAppInstance(reader, "11111111-1111-1111-1111-111111111111")
	reader.GetEternal("11111111-1111-1111-1111-111111111111", "demo.FeatureConfig")

	reader.mutex.RLock()
	state, ok := reader.instantConfigStatesByKey[redised.FormatConfigKey("demo.FeatureConfig")]
	reader.mutex.RUnlock()
	assert.False(t, ok)
	assert.Nil(t, state)
	assert.Equal(t, `{"enabled":true}`, configValueSnapshot(reader, "demo.FeatureConfig", "11111111-1111-1111-1111-111111111111"))
}

func TestGetInstantCreatesSharedStateForMultipleApps(t *testing.T) {
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

	reader.mutex.RLock()
	state := reader.instantConfigStatesByKey[redised.FormatConfigKey("demo.FeatureConfig")]
	firstValuesByKey := reader.configValuesByAppInstanceID[firstAppInstanceID]
	secondValuesByKey := reader.configValuesByAppInstanceID[secondAppInstanceID]
	reader.mutex.RUnlock()
	assert.NotNil(t, state)
	assert.NotNil(t, state.cancel)
	assert.Equal(t, `{"enabled":true}`, state.value)
	assert.Len(t, state.refsByAppInstanceID, 2)
	assert.Equal(t, `{"enabled":true}`, firstValuesByKey[redised.FormatConfigKey("demo.FeatureConfig")])
	assert.Equal(t, `{"enabled":true}`, secondValuesByKey[redised.FormatConfigKey("demo.FeatureConfig")])
}

func TestGetInstantReturnsRetainedSnapshot(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)
	reader.GetInstant(appInstanceID, "demo.FeatureConfig")
	reader.Client.SetValue(
		redised.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(appInstanceID, "demo.FeatureConfig"))
}

func TestGetEternalReloadsValuePerAppInstance(t *testing.T) {
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
	reader.GetEternal(firstAppInstanceID, "demo.FeatureConfig")
	reader.Client.SetValue(
		redised.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	)
	reader.GetEternal(secondAppInstanceID, "demo.FeatureConfig")

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(firstAppInstanceID, "demo.FeatureConfig"))
	assert.Equal(t, `{"enabled":false}`, reader.GetEternal(secondAppInstanceID, "demo.FeatureConfig"))
}

func TestGetEternalRetriesAfterMissingConfig(t *testing.T) {
	reader := newTestReader(nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)

	assert.Equal(t, "", reader.GetEternal(appInstanceID, "demo.FeatureConfig"))
	reader.Client.SetValue(
		redised.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(appInstanceID, "demo.FeatureConfig"))
}

func TestGetEternalRetriesAfterInvalidConfig(t *testing.T) {
	reader := newTestReader(nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, appInstanceID)
	reader.Client.SetValue(redised.FormatConfigKey("demo.FeatureConfig"), "not-a-config-value")

	assert.Equal(t, "", reader.GetEternal(appInstanceID, "demo.FeatureConfig"))
	reader.Client.SetValue(
		redised.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(appInstanceID, "demo.FeatureConfig"))
}

func TestGetEternalKeepsSnapshotAfterDeleteForExistingInstance(t *testing.T) {
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

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(firstAppInstanceID, "demo.FeatureConfig"))
	reader.Client.SetValue(redised.FormatConfigKey("demo.FeatureConfig"), "")

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(firstAppInstanceID, "demo.FeatureConfig"))
	assert.Equal(t, "", reader.GetEternal(secondAppInstanceID, "demo.FeatureConfig"))
}
