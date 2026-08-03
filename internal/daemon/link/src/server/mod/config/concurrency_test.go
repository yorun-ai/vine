package config

import (
	"sync"
	"testing"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func TestGetInstantConcurrentWithEventUpdate(t *testing.T) {
	const (
		appInstanceID = "11111111-1111-1111-1111-111111111111"
		configName    = "demo.FeatureConfig"
		firstValue    = `{"enabled":true}`
		secondValue   = `{"enabled":false}`
	)
	reader := newTestReader(map[string]redised.ConfigValue{
		configName: {
			Name:  configName,
			Value: []byte(firstValue),
		},
	})
	registerTestAppInstance(reader, appInstanceID)
	if value := reader.GetInstant(appInstanceID, configName); value != firstValue {
		t.Fatalf("unexpected initial config value: %s", value)
	}

	redisKey := redised.FormatConfigKey(configName)
	eventValues := []string{
		marshalTestConfigValue(configName, firstValue),
		marshalTestConfigValue(configName, secondValue),
	}
	start := make(chan struct{})
	invalidValues := make(chan string, 1)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		<-start
		for idx := 0; idx < 1000; idx++ {
			reader.handleInstantConfigEvent(redisKey, hubredis.Event{
				Kind:  hubredis.EventKindUpsert,
				Key:   redisKey,
				Value: eventValues[idx%len(eventValues)],
			})
		}
	}()
	go func() {
		defer waitGroup.Done()
		<-start
		for idx := 0; idx < 1000; idx++ {
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
	}()
	close(start)
	waitGroup.Wait()
	close(invalidValues)

	if value, ok := <-invalidValues; ok {
		t.Fatalf("unexpected concurrent config value: %q", value)
	}
}
