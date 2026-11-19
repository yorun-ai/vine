package config

import (
	"encoding/json/v2"
	"uuid"

	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

func (c *Reader) GetEternal(appInstanceID uuid.UUID, key string) string {
	if value, ok := c.findConfigValueSnapshot(appInstanceID, key); ok {
		return value
	}
	return c.retainEternalConfig(appInstanceID, key)
}

func (c *Reader) retainEternalConfig(appInstanceID uuid.UUID, key string) string {
	watchKey := watched.FormatConfigKey(key)
	configValue, ok := c.loadConfigValue(watchKey)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	values := c.configValuesByAppInstanceID[appInstanceID]
	if cached, exists := values[watchKey]; exists {
		return cached
	}
	if !ok {
		return ""
	}
	value := string(configValue.Value)
	if values != nil {
		values[watchKey] = value
	}
	return value
}

func (c *Reader) GetInstant(appInstanceID uuid.UUID, key string) string {
	if value, ok := c.findConfigValueSnapshot(appInstanceID, key); ok {
		return value
	}
	return c.retainInstantConfig(appInstanceID, key)
}

func (c *Reader) retainInstantConfig(appInstanceID uuid.UUID, key string) string {
	watchKey := watched.FormatConfigKey(key)
	c.mutex.Lock()
	state := c.instantConfigStatesByKey[watchKey]
	if state == nil {
		state = c.newInstantConfigState(watchKey)
		c.instantConfigStatesByKey[watchKey] = state
	}
	state.refsByAppInstanceID[appInstanceID] = struct{}{}
	c.mutex.Unlock()

	// Callers for the same key share the initial load. Other keys and
	// instance teardown can proceed while Watch attaches the subscription.
	state.start()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, retained := state.refsByAppInstanceID[appInstanceID]; retained && c.instantConfigStatesByKey[watchKey] == state {
		c.setConfigValueSnapshotLocked(appInstanceID, watchKey, state.value)
	}
	return state.value
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

func (c *Reader) findConfigValueSnapshot(appInstanceID uuid.UUID, key string) (string, bool) {
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

func (c *Reader) setConfigValueSnapshotLocked(appInstanceID uuid.UUID, watchKey string, value string) {
	valuesByKey, ok := c.configValuesByAppInstanceID[appInstanceID]
	if !ok {
		return
	}
	valuesByKey[watchKey] = value
}
