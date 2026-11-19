package config

import (
	"context"
	"sync"
	"uuid"

	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

func (c *Reader) newInstantConfigState(watchKey string) *_InstantConfigState {
	watchCtx, cancel := context.WithCancel(c.Context)
	state := &_InstantConfigState{
		refsByAppInstanceID: map[uuid.UUID]struct{}{},
		cancel:              cancel,
	}
	state.start = sync.OnceFunc(func() {
		c.startInstantConfig(watchCtx, watchKey, state)
	})
	return state
}

func (c *Reader) startInstantConfig(ctx context.Context, watchKey string, state *_InstantConfigState) {
	// A failed Watch attachment panics. Release its resources and allow the
	// next request to start a fresh subscription for this key.
	started := false
	defer func() {
		if !started {
			state.cancel()
			c.mutex.Lock()
			if c.instantConfigStatesByKey[watchKey] == state {
				delete(c.instantConfigStatesByKey, watchKey)
			}
			c.mutex.Unlock()
		}
	}()
	raw, ok, subscription := c.Client.LoadAndSubscribe(ctx, watchKey, func(event hubwatch.Event) {
		c.handleInstantConfigEvent(watchKey, state, event)
	})
	if ok {
		if value, err := unmarshalConfigValue(raw); err == nil {
			c.mutex.Lock()
			state.value = string(value.Value)
			c.mutex.Unlock()
		}
	}
	subscription.Start()
	started = true
}

func (c *Reader) handleInstantConfigEvent(watchKey string, state *_InstantConfigState, event hubwatch.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.instantConfigStatesByKey[watchKey] != state {
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

func (c *Reader) releaseInstantConfigStateByInstance(appInstanceID uuid.UUID) {
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
