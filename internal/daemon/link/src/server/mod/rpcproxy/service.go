package rpcproxy

import (
	"context"
	"encoding/json"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func (p *RpcProxy) retainService(serviceName string, appInstanceID string) {
	p.serviceStateMutex.Lock()
	if state, ok := p.serviceStatesByName[serviceName]; ok {
		state.refsByAppInstanceID[appInstanceID] = struct{}{}
		p.serviceStateMutex.Unlock()
		return
	}
	p.serviceStateMutex.Unlock()

	state := p.newServiceState(serviceName)
	state.refsByAppInstanceID[appInstanceID] = struct{}{}

	p.serviceStateMutex.Lock()
	if existing, ok := p.serviceStatesByName[serviceName]; ok {
		existing.refsByAppInstanceID[appInstanceID] = struct{}{}
		p.serviceStateMutex.Unlock()
		if state.cancel != nil {
			state.cancel()
		}
		return
	}
	p.serviceStatesByName[serviceName] = state
	p.serviceStateMutex.Unlock()
}

func (p *RpcProxy) releaseInstanceState(appInstanceID string) {
	p.serviceStateMutex.Lock()
	cancels := make([]context.CancelFunc, 0)
	for serviceName, state := range p.serviceStatesByName {
		delete(state.refsByAppInstanceID, appInstanceID)
		if len(state.refsByAppInstanceID) > 0 {
			continue
		}
		delete(p.serviceStatesByName, serviceName)
		if state.cancel != nil {
			cancels = append(cancels, state.cancel)
		}
	}
	p.serviceStateMutex.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (p *RpcProxy) nextServiceEndpoint(serviceName string) (redised.RpcServiceRegistration, bool) {
	p.serviceStateMutex.Lock()
	defer p.serviceStateMutex.Unlock()
	state, ok := p.serviceStatesByName[serviceName]
	if !ok || len(state.endpoints) == 0 {
		return redised.RpcServiceRegistration{}, false
	}
	if state.nextIndex >= len(state.endpoints) {
		state.nextIndex = 0
	}
	registration := state.endpoints[state.nextIndex]
	state.nextIndex = (state.nextIndex + 1) % len(state.endpoints)
	return registration, true
}

func (p *RpcProxy) newServiceState(serviceName string) *_ServiceState {
	watchCtx, cancel := context.WithCancel(p.Context)
	registrationsByKey := p.loadAndWatchServiceRegistrations(serviceName, watchCtx)
	state := &_ServiceState{
		refsByAppInstanceID: map[string]struct{}{},
		registrationsByKey:  registrationsByKey,
		cancel:              cancel,
	}
	p.rebuildServiceEndpointsLocked(state)
	return state
}

func (p *RpcProxy) loadAndWatchServiceRegistrations(serviceName string, ctx context.Context) map[string]redised.RpcServiceRegistration {
	valuesByKey := p.RedisClient.LoadListAndSubscribe(
		ctx,
		redised.FormatRpcServiceRegistrationPrefix(serviceName),
		func(event hubredis.Event) {
			p.handleServiceRegistrationEvent(serviceName, event)
		},
	)
	return parseServiceRegistrations(valuesByKey)
}

func parseServiceRegistrations(valuesByKey map[string]string) map[string]redised.RpcServiceRegistration {
	registrationsByKey := map[string]redised.RpcServiceRegistration{}
	for key, value := range valuesByKey {
		var registration redised.RpcServiceRegistration
		if err := json.Unmarshal([]byte(value), &registration); err != nil {
			continue
		}
		registrationsByKey[key] = registration
	}
	return registrationsByKey
}

func (p *RpcProxy) handleServiceRegistrationEvent(serviceName string, event hubredis.Event) {
	p.serviceStateMutex.Lock()
	defer p.serviceStateMutex.Unlock()
	state, ok := p.serviceStatesByName[serviceName]
	if !ok {
		return
	}

	switch event.Kind {
	case hubredis.EventKindDelete:
		delete(state.registrationsByKey, event.Key)
	default:
		var registration redised.RpcServiceRegistration
		if err := json.Unmarshal([]byte(event.Value), &registration); err != nil {
			return
		}
		state.registrationsByKey[event.Key] = registration
	}
	p.rebuildServiceEndpointsLocked(state)
}

func (p *RpcProxy) rebuildServiceEndpointsLocked(state *_ServiceState) {
	endpoints := make([]redised.RpcServiceRegistration, 0, len(state.registrationsByKey))
	for _, registration := range state.registrationsByKey {
		endpoints = append(endpoints, registration)
	}
	state.endpoints = endpoints
	if len(state.endpoints) == 0 || state.nextIndex >= len(state.endpoints) {
		state.nextIndex = 0
	}
}
