package config

import (
	"context"

	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

func (c *Reader) newInstantConfigState(watchKey string) (*_InstantConfigState, hubwatch.Subscription) {
	watchCtx, cancel := context.WithCancel(c.Context)
	state := &_InstantConfigState{
		refsByAppInstanceID: map[string]struct{}{},
		cancel:              cancel,
	}
	loadedConfigValue, loadedOk, subscription := c.loadAndWatchInstantConfigValue(watchKey, watchCtx)
	if !loadedOk {
		return state, subscription
	}

	state.value = string(loadedConfigValue.Value)
	return state, subscription
}

func (c *Reader) loadAndWatchInstantConfigValue(watchKey string, ctx context.Context) (watched.ConfigValue, bool, hubwatch.Subscription) {
	value, ok, subscription := c.Client.LoadAndSubscribe(ctx, watchKey, func(event hubwatch.Event) {
		c.handleInstantConfigEvent(watchKey, event)
	})
	if !ok {
		return watched.ConfigValue{}, false, subscription
	}
	configValue, err := unmarshalConfigValue(value)
	if err != nil {
		return watched.ConfigValue{}, false, subscription
	}
	return configValue, true, subscription
}

func (c *Reader) handleInstantConfigEvent(watchKey string, event hubwatch.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state, ok := c.instantConfigStatesByKey[watchKey]
	if !ok {
		return
	}

	switch event.Kind {
	case hubwatch.EventKindDelete:
		state.value = ""
		for appInstanceID := range state.refsByAppInstanceID {
			c.setConfigValueSnapshotLocked(appInstanceID, watchKey, "")
		}
	default:
		configValue, err := unmarshalConfigValue(event.Value)
		if err != nil {
			return
		}
		value := string(configValue.Value)
		state.value = value
		for appInstanceID := range state.refsByAppInstanceID {
			c.setConfigValueSnapshotLocked(appInstanceID, watchKey, value)
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
