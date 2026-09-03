package config

import (
	"encoding/json/v2"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func (c *Reader) GetEternal(appInstanceID string, key string) string {
	if value, ok := c.findConfigValueSnapshot(appInstanceID, key); ok {
		return value
	}
	return c.retainEternalConfig(appInstanceID, key)
}

func (c *Reader) retainEternalConfig(appInstanceID string, key string) string {
	redisKey := redised.FormatConfigKey(key)
	configValue, ok := c.loadConfigValue(redisKey)
	if !ok {
		return ""
	}

	value := string(configValue.Value)
	c.mutex.Lock()
	c.setConfigValueSnapshotLocked(appInstanceID, redisKey, value)
	c.mutex.Unlock()
	return value
}

func (c *Reader) GetInstant(appInstanceID string, key string) string {
	if value, ok := c.findConfigValueSnapshot(appInstanceID, key); ok {
		return value
	}
	return c.retainInstantConfig(appInstanceID, key)
}

func (c *Reader) retainInstantConfig(appInstanceID string, key string) string {
	redisKey := redised.FormatConfigKey(key)
	c.mutex.Lock()
	state, exists := c.instantConfigStatesByKey[redisKey]
	var subscription hubredis.Subscription
	if !exists {
		state, subscription = c.newInstantConfigState(redisKey)
		c.instantConfigStatesByKey[redisKey] = state
	}
	value := state.value
	state.refsByAppInstanceID[appInstanceID] = struct{}{}
	c.setConfigValueSnapshotLocked(appInstanceID, redisKey, value)
	if subscription != nil {
		subscription.Start()
	}
	c.mutex.Unlock()
	return value
}

func (c *Reader) loadConfigValue(redisKey string) (redised.ConfigValue, bool) {
	value, ok := c.Client.Load(redisKey)
	if !ok {
		return redised.ConfigValue{}, false
	}

	configValue, err := unmarshalConfigValue(value)
	if err != nil {
		return redised.ConfigValue{}, false
	}
	return configValue, true
}

func unmarshalConfigValue(value string) (configValue redised.ConfigValue, err error) {
	err = json.Unmarshal([]byte(value), &configValue)
	return
}

func (c *Reader) findConfigValueSnapshot(appInstanceID string, key string) (string, bool) {
	redisKey := redised.FormatConfigKey(key)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return "", false
	}

	value, ok := valuesByKey[redisKey]
	return value, ok
}

func (c *Reader) setConfigValueSnapshotLocked(appInstanceID string, redisKey string, value string) {
	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return
	}
	valuesByKey[redisKey] = value
}
