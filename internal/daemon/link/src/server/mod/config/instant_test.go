package config

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
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

	reader.handleInstantConfigEvent(watched.FormatConfigKey("demo.FeatureConfig"), hubwatch.Event{
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
	reader.handleInstantConfigEvent(watched.FormatConfigKey("demo.FeatureConfig"), hubwatch.Event{
		Kind:  hubwatch.EventKindUpsert,
		Value: marshalTestConfigValue("demo.FeatureConfig", `{"enabled":false}`),
	})

	assert.Equal(t, `{"enabled":false}`, reader.GetInstant(uuid.MustParse(secondAppInstanceID), "demo.FeatureConfig"))
}
