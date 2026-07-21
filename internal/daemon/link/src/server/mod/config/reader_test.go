package config

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAfterAppStopClearsRetainedStates(t *testing.T) {
	reader := newTestReader(map[string]redised.ConfigValue{
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
	reader := newTestReader(map[string]redised.ConfigValue{
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
	_, ok := reader.instantConfigStatesByKey[redised.FormatConfigKey("demo.FeatureConfig")]
	reader.mutex.RUnlock()
	assert.False(t, ok)
}
