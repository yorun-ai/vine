package epmgr

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"sync"

	"go.yorun.ai/vine/internal/core/logger"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

var epmgrLogger = logger.New("daemon:portal:epmgr")

type Watcher struct {
	manager *Manager
	route   *_Route
	once    sync.Once
}

func (w *Watcher) Release() {
	w.once.Do(func() { w.manager.release(w.route) })
}

func (m *Manager) watch(prefix string, registrationType reflect.Type) *Watcher {
	m.mutex.Lock()
	route := m.routesByPrefix[prefix]
	if route != nil {
		if route.registrationType != registrationType {
			m.mutex.Unlock()
			panic(fmt.Sprintf("endpoint route %s registration type mismatch: got %s want %s", prefix, registrationType, route.registrationType))
		}
		route.refCount++
		m.mutex.Unlock()
		return &Watcher{manager: m, route: route}
	}
	m.mutex.Unlock()

	watchCtx, cancel := context.WithCancel(m.Context)
	route = &_Route{
		prefix:           prefix,
		registrationType: registrationType,
		cancel:           cancel,
		refCount:         1,
	}
	valuesByKey, subscription := m.Watch.LoadListAndSubscribe(watchCtx, prefix, func(event hubwatch.Event) {
		m.handleRegistrationEvent(route, event)
	})
	route.registrationsByKey = parseRegistrations(valuesByKey, registrationType)
	route.rebuildEndpoints()

	m.mutex.Lock()

	existing := m.routesByPrefix[prefix]
	if existing != nil {
		if existing.registrationType != registrationType {
			m.mutex.Unlock()
			cancel()
			panic(fmt.Sprintf("endpoint route %s registration type mismatch: got %s want %s", prefix, registrationType, existing.registrationType))
		}
		existing.refCount++
		m.mutex.Unlock()
		cancel()
		return &Watcher{manager: m, route: existing}
	}
	m.routesByPrefix[prefix] = route
	subscription.Start()
	m.mutex.Unlock()
	return &Watcher{manager: m, route: route}
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
		epmgrLogger.Warn("vine.portal epmgr registration is invalid", "key", key, "error", err)
		return nil, false
	}
	return registration.Interface(), true
}

func (m *Manager) release(route *_Route) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	route.refCount--
	if route.refCount > 0 {
		return
	}
	route.cancel()
	delete(m.routesByPrefix, route.prefix)
}

func (m *Manager) handleRegistrationEvent(route *_Route, event hubwatch.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.routesByPrefix[route.prefix] != route {
		return
	}

	switch event.Kind {
	case hubwatch.EventKindDelete:
		delete(route.registrationsByKey, event.Key)
	default:
		registration, ok := parseRegistration(event.Key, event.Value, route.registrationType)
		if !ok {
			return
		}
		if reflect.DeepEqual(route.registrationsByKey[event.Key], registration) {
			return
		}
		route.registrationsByKey[event.Key] = registration
	}
	route.rebuildEndpoints()
}

func (r *_Route) rebuildEndpoints() {
	endpoints := make([]any, 0, len(r.registrationsByKey))
	states := make(map[any]*_EndpointState, len(r.registrationsByKey))
	for _, registration := range r.registrationsByKey {
		endpoints = append(endpoints, registration)
		state := r.endpointStates[registration]
		if state == nil {
			state = new(_EndpointState)
		}
		states[registration] = state
	}
	r.endpoints = endpoints
	r.endpointStates = states
	if len(endpoints) == 0 || r.nextIndex >= len(endpoints) {
		r.nextIndex = 0
	}
}
