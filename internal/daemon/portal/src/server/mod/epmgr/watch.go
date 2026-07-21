package epmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"go.yorun.ai/vine/internal/core/logger"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
)

type Watcher struct {
	manager  *Manager
	prefix   string
	mutex    sync.Mutex
	released bool
}

func (w *Watcher) Release() {
	w.mutex.Lock()
	if w.released {
		w.mutex.Unlock()
		return
	}
	w.released = true
	w.mutex.Unlock()

	w.manager.release(w.prefix)
}

func (m *Manager) watch(prefix string, registrationType reflect.Type) *Watcher {
	m.mutex.Lock()
	route := m.routesByPrefix[prefix]
	if route != nil {
		if route.registrationType != registrationType {
			panic(fmt.Sprintf("endpoint route %s registration type mismatch: got %s want %s", prefix, registrationType, route.registrationType))
		}
		route.refCount++
		m.mutex.Unlock()
		return m.newWatcher(prefix)
	}
	m.mutex.Unlock()

	watchCtx, cancel := context.WithCancel(m.Context)
	valuesByKey := m.Redis.LoadListAndSubscribe(watchCtx, prefix, func(event hubredis.Event) {
		m.handleRegistrationEvent(prefix, event)
	})
	route = &_Route{
		prefix:             prefix,
		registrationType:   registrationType,
		cancel:             cancel,
		refCount:           1,
		registrationsByKey: parseRegistrations(valuesByKey, registrationType),
	}
	route.rebuildEndpoints()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing := m.routesByPrefix[prefix]
	if existing != nil {
		if existing.registrationType != registrationType {
			cancel()
			panic(fmt.Sprintf("endpoint route %s registration type mismatch: got %s want %s", prefix, registrationType, existing.registrationType))
		}
		existing.refCount++
		cancel()
		return m.newWatcher(prefix)
	}
	m.routesByPrefix[prefix] = route
	return m.newWatcher(prefix)
}

func (m *Manager) newWatcher(prefix string) *Watcher {
	return &Watcher{
		manager: m,
		prefix:  prefix,
	}
}

func parseRegistrations(valuesByKey map[string]string, registrationType reflect.Type) map[string]any {
	registrationsByKey := map[string]any{}
	for key, value := range valuesByKey {
		if registration, ok := parseRegistration(key, value, registrationType); ok {
			registrationsByKey[key] = registration
		}
	}
	return registrationsByKey
}

func parseRegistration(key string, value string, registrationType reflect.Type) (any, bool) {
	registration := reflect.New(registrationType)
	if err := json.Unmarshal([]byte(value), registration.Interface()); err != nil {
		logger.Warn("vine.portal epmgr registration is invalid", "key", key, "error", err)
		return nil, false
	}
	return registration.Interface(), true
}

func (m *Manager) release(prefix string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	route := m.routesByPrefix[prefix]
	if route == nil {
		return
	}
	route.refCount--
	if route.refCount > 0 {
		return
	}
	route.cancel()
	delete(m.routesByPrefix, prefix)
}

func (m *Manager) handleRegistrationEvent(prefix string, event hubredis.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	route := m.routesByPrefix[prefix]
	if route == nil {
		return
	}

	switch event.Kind {
	case hubredis.EventKindDelete:
		delete(route.registrationsByKey, event.Key)
	default:
		registration, ok := parseRegistration(event.Key, event.Value, route.registrationType)
		if !ok {
			return
		}
		route.registrationsByKey[event.Key] = registration
	}
	route.rebuildEndpoints()
}

func (r *_Route) rebuildEndpoints() {
	endpoints := make([]any, 0, len(r.registrationsByKey))
	for _, registration := range r.registrationsByKey {
		endpoints = append(endpoints, registration)
	}
	r.endpoints = endpoints
	if len(endpoints) == 0 || r.nextIndex >= len(endpoints) {
		r.nextIndex = 0
	}
}
