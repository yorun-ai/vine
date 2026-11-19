package config

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
)

func TestGetInstantReturnsEmptyWhenConfigDoesNotExist(t *testing.T) {
	reader := newTestReader(t, nil)
	registerTestAppInstance(reader, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	value := reader.GetInstant(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "demo.FeatureConfig")
	assert.Equal(t, "", value)
}

func TestGetInstantReturnsEmptyWhenStoredValueIsInvalid(t *testing.T) {
	reader := newTestReader(t, nil)
	registerTestAppInstance(reader, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	reader.Client.(*watchtest.Client).SetValue(watched.FormatConfigKey("demo.FeatureConfig"), "not-a-config-value")

	value := reader.GetInstant(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "demo.FeatureConfig")

	assert.Equal(t, "", value)
	assert.Equal(t, "", configValueSnapshot(reader, "demo.FeatureConfig", uuid.MustParse("11111111-1111-1111-1111-111111111111")))
}

func TestGetEternalDoesNotStartWatcher(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	registerTestAppInstance(reader, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	reader.GetEternal(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "demo.FeatureConfig")

	reader.mutex.RLock()
	state, ok := reader.instantConfigStatesByKey[watched.FormatConfigKey("demo.FeatureConfig")]
	reader.mutex.RUnlock()
	assert.False(t, ok)
	assert.Nil(t, state)
	assert.Equal(t, `{"enabled":true}`, configValueSnapshot(reader, "demo.FeatureConfig", uuid.MustParse("11111111-1111-1111-1111-111111111111")))
}

func TestGetInstantCreatesSharedStateForMultipleApps(t *testing.T) {
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

	reader.mutex.RLock()
	state := reader.instantConfigStatesByKey[watched.FormatConfigKey("demo.FeatureConfig")]
	firstValuesByKey := reader.configValuesByAppInstanceID[uuid.MustParse(firstAppInstanceID)]
	secondValuesByKey := reader.configValuesByAppInstanceID[uuid.MustParse(secondAppInstanceID)]
	reader.mutex.RUnlock()
	assert.NotNil(t, state)
	assert.NotNil(t, state.cancel)
	assert.Equal(t, `{"enabled":true}`, state.value)
	assert.Len(t, state.refsByAppInstanceID, 2)
	assert.Equal(t, `{"enabled":true}`, firstValuesByKey[watched.FormatConfigKey("demo.FeatureConfig")])
	assert.Equal(t, `{"enabled":true}`, secondValuesByKey[watched.FormatConfigKey("demo.FeatureConfig")])
}

func TestGetInstantReturnsRetainedSnapshot(t *testing.T) {
	reader := newTestReader(t, map[string]watched.ConfigValue{
		"demo.FeatureConfig": {
			Name:  "demo.FeatureConfig",
			Value: []byte(`{"enabled":true}`),
		},
	})
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, uuid.MustParse(appInstanceID))
	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
	reader.Client.(*watchtest.Client).SetValue(
		watched.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetInstant(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
}

func TestGetEternalReloadsValuePerAppInstance(t *testing.T) {
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
	reader.GetEternal(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig")
	reader.Client.(*watchtest.Client).SetValue(
		watched.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	)
	reader.GetEternal(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig")

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig"))
	assert.Equal(t, `{"enabled":false}`, reader.GetEternal(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig"))
}

func TestGetEternalRetriesAfterMissingConfig(t *testing.T) {
	reader := newTestReader(t, nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, uuid.MustParse(appInstanceID))

	assert.Equal(t, "", reader.GetEternal(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
	reader.Client.(*watchtest.Client).SetValue(
		watched.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
}

func TestGetEternalRetriesAfterInvalidConfig(t *testing.T) {
	reader := newTestReader(t, nil)
	appInstanceID := "11111111-1111-1111-1111-111111111111"
	registerTestAppInstance(reader, uuid.MustParse(appInstanceID))
	reader.Client.(*watchtest.Client).SetValue(watched.FormatConfigKey("demo.FeatureConfig"), "not-a-config-value")

	assert.Equal(t, "", reader.GetEternal(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
	reader.Client.(*watchtest.Client).SetValue(
		watched.FormatConfigKey("demo.FeatureConfig"),
		marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`),
	)

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(uuid.MustParse(appInstanceID), "demo.FeatureConfig"))
}

func TestGetEternalKeepsSnapshotAfterDeleteForExistingInstance(t *testing.T) {
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

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig"))
	reader.Client.(*watchtest.Client).SetValue(watched.FormatConfigKey("demo.FeatureConfig"), "")

	assert.Equal(t, `{"enabled":true}`, reader.GetEternal(uuid.MustParse(firstAppInstanceID), "demo.FeatureConfig"))
	assert.Equal(t, "", reader.GetEternal(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig"))
}
