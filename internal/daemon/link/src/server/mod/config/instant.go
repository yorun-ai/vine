package config

import (
	"context"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func (c *Reader) newInstantConfigState(redisKey string) *_InstantConfigState {
	watchCtx, cancel := context.WithCancel(c.Context)
	state := &_InstantConfigState{
		refsByAppInstanceID: map[string]struct{}{},
		cancel:              cancel,
	}
	loadedConfigValue, loadedOk := c.loadAndWatchInstantConfigValue(redisKey, watchCtx)
	if !loadedOk {
		return state
	}

	state.value = string(loadedConfigValue.Value)
	return state
}

func (c *Reader) loadAndWatchInstantConfigValue(redisKey string, ctx context.Context) (redised.ConfigValue, bool) {
	value, ok := c.Client.LoadAndSubscribe(ctx, redisKey, func(event hubredis.Event) {
		c.handleInstantConfigEvent(redisKey, event)
	})
	if !ok {
		return redised.ConfigValue{}, false
	}
	configValue, err := unmarshalConfigValue(value)
	if err != nil {
		return redised.ConfigValue{}, false
	}
	return configValue, true
}

func (c *Reader) handleInstantConfigEvent(redisKey string, event hubredis.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state, ok := c.instantConfigStatesByKey[redisKey]
	if !ok {
		return
	}

	switch event.Kind {
	case hubredis.EventKindDelete:
		state.value = ""
		for appInstanceID := range state.refsByAppInstanceID {
			c.setConfigValueSnapshot(appInstanceID, redisKey, "")
		}
	default:
		configValue, err := unmarshalConfigValue(event.Value)
		if err != nil {
			return
		}
		value := string(configValue.Value)
		state.value = value
		for appInstanceID := range state.refsByAppInstanceID {
			c.setConfigValueSnapshot(appInstanceID, redisKey, value)
		}
	}
}

func (c *Reader) stopAllInstantConfigWatchers() {
	c.mutex.Lock()
	cancels := make([]context.CancelFunc, 0, len(c.instantConfigStatesByKey))
	for _, state := range c.instantConfigStatesByKey {
		if state.cancel != nil {
			cancels = append(cancels, state.cancel)
		}
	}
	clear(c.instantConfigStatesByKey)
	clear(c.configValuesByAppInstanceID)
	c.mutex.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (c *Reader) releaseInstantConfigStateByInstance(appInstanceID string) {
	c.mutex.Lock()
	cancels := make([]context.CancelFunc, 0)
	for key, state := range c.instantConfigStatesByKey {
		delete(state.refsByAppInstanceID, appInstanceID)
		if len(state.refsByAppInstanceID) > 0 {
			continue
		}
		delete(c.instantConfigStatesByKey, key)
		if state.cancel != nil {
			cancels = append(cancels, state.cancel)
		}
	}
	delete(c.configValuesByAppInstanceID, appInstanceID)
	c.mutex.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}
