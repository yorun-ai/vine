package config

import (
	"encoding/json/v2"

	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

func (c *Reader) GetEternal(appInstanceID string, key string) string {
	if value, ok := c.findConfigValueSnapshot(appInstanceID, key); ok {
		return value
	}
	return c.retainEternalConfig(appInstanceID, key)
}

func (c *Reader) retainEternalConfig(appInstanceID string, key string) string {
	watchKey := watched.FormatConfigKey(key)
	configValue, ok := c.loadConfigValue(watchKey)
	if !ok {
		return ""
	}

	value := string(configValue.Value)
	c.mutex.Lock()
	c.setConfigValueSnapshotLocked(appInstanceID, watchKey, value)
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
	watchKey := watched.FormatConfigKey(key)
	c.mutex.Lock()
	state, exists := c.instantConfigStatesByKey[watchKey]
	var subscription hubwatch.Subscription
	if !exists {
		state, subscription = c.newInstantConfigState(watchKey)
		c.instantConfigStatesByKey[watchKey] = state
	}
	value := state.value
	state.refsByAppInstanceID[appInstanceID] = struct{}{}
	c.setConfigValueSnapshotLocked(appInstanceID, watchKey, value)
	if subscription != nil {
		subscription.Start()
	}
	c.mutex.Unlock()
	return value
}

func (c *Reader) loadConfigValue(watchKey string) (watched.ConfigValue, bool) {
	value, ok := c.Client.Load(watchKey)
	if !ok {
		return watched.ConfigValue{}, false
	}

	configValue, err := unmarshalConfigValue(value)
	if err != nil {
		return watched.ConfigValue{}, false
	}
	return configValue, true
}

func unmarshalConfigValue(value string) (configValue watched.ConfigValue, err error) {
	err = json.Unmarshal([]byte(value), &configValue)
	return
}

func (c *Reader) findConfigValueSnapshot(appInstanceID string, key string) (string, bool) {
	watchKey := watched.FormatConfigKey(key)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return "", false
	}

	value, ok := valuesByKey[watchKey]
	return value, ok
}

func (c *Reader) setConfigValueSnapshotLocked(appInstanceID string, watchKey string, value string) {
	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return
	}
	valuesByKey[watchKey] = value
}
