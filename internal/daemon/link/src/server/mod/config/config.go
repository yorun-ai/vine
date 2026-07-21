package config

import (
	"encoding/json"

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
	c.setConfigValueSnapshot(appInstanceID, redisKey, value)
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
	if !exists {
		state = c.newInstantConfigState(redisKey)
		c.instantConfigStatesByKey[redisKey] = state
	}
	state.refsByAppInstanceID[appInstanceID] = struct{}{}
	c.setConfigValueSnapshot(appInstanceID, redisKey, state.value)
	c.mutex.Unlock()
	return state.value
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
	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	c.mutex.RUnlock()
	if !ok {
		return "", false
	}

	value, ok := valuesByKey[redisKey]
	return value, ok
}

func (c *Reader) setConfigValueSnapshot(appInstanceID string, redisKey string, value string) {
	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return
	}
	valuesByKey[redisKey] = value
}
