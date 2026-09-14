package config

import (
	"sync"
	"testing"

	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

func TestGetInstantConcurrentWithEventUpdate(t *testing.T) {
	const (
		appInstanceID = "11111111-1111-1111-1111-111111111111"
		configName    = "demo.FeatureConfig"
		firstValue    = `{"enabled":true}`
		secondValue   = `{"enabled":false}`
	)
	reader := newTestReader(map[string]watched.ConfigValue{
		configName: {
			Name:  configName,
			Value: []byte(firstValue),
		},
	})
	registerTestAppInstance(reader, appInstanceID)
	if value := reader.GetInstant(appInstanceID, configName); value != firstValue {
		t.Fatalf("unexpected initial config value: %s", value)
	}

	watchKey := watched.FormatConfigKey(configName)
	eventValues := []string{
		marshalTestConfigValue(configName, firstValue),
		marshalTestConfigValue(configName, secondValue),
	}
	start := make(chan struct{})
	invalidValues := make(chan string, 1)
	var waitGroup sync.WaitGroup
	waitGroup.Go(func() {
		<-start
		for idx := range 1000 {
			reader.handleInstantConfigEvent(watchKey, hubwatch.Event{
				Kind:  hubwatch.EventKindUpsert,
				Key:   watchKey,
				Value: eventValues[idx%len(eventValues)],
			})
		}
	})
	waitGroup.Go(func() {
		<-start
		for idx := range 1000 {
			value := reader.GetInstant(appInstanceID, configName)
			if idx%2 != 0 {
				value = reader.retainInstantConfig(appInstanceID, configName)
			}
			if value == firstValue || value == secondValue {
				continue
			}
			select {
			case invalidValues <- value:
			default:
			}
			return
		}
	})
	close(start)
	waitGroup.Wait()
	close(invalidValues)

	if value, ok := <-invalidValues; ok {
		t.Fatalf("unexpected concurrent config value: %q", value)
	}
}
