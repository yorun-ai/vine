package epmgr

import (
	"context"
	"reflect"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

type Manager struct {
	app.BaseModule

	Context context.Context       `inject:""`
	Watch   hubapiwatch.ClientOps `inject:""`

	mutex          sync.Mutex
	routesByPrefix map[string]*_Route
}

var (
	rpcServiceRegistrationType = reflect.TypeFor[watched.RpcServiceRegistration]()
	webRegistrationType        = reflect.TypeFor[watched.WebRegistration]()
)

type _Route struct {
	prefix             string
	registrationType   reflect.Type
	cancel             context.CancelFunc
	refCount           int
	registrationsByKey map[string]any
	endpoints          []any
	nextIndex          int
	endpointStates     map[any]*_EndpointState
}

func (m *Manager) DIInit() {
	m.routesByPrefix = map[string]*_Route{}
}

func (m *Manager) WatchRpc(serviceName string) *Watcher {
	return m.watch(watched.FormatRpcServiceRegistrationPrefix(serviceName), rpcServiceRegistrationType)
}

func (m *Manager) WatchWeb(webName string) *Watcher {
	return m.watch(watched.FormatWebRegistrationPrefix(webName), webRegistrationType)
}

// NextRpcEndpoint reserves an available endpoint. The caller must finish the
// attempt, including when forwarding is abandoned before sending a request.
func (m *Manager) NextRpcEndpoint(serviceName string) (*watched.RpcServiceRegistration, *Attempt, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	prefix := watched.FormatRpcServiceRegistrationPrefix(serviceName)
	route := m.routesByPrefix[prefix]
	if route == nil {
		return nil, nil, false
	}
	now := time.Now()
	for range len(route.endpoints) {
		registration := route.endpoints[route.nextIndex]
		route.nextIndex = (route.nextIndex + 1) % len(route.endpoints)
		state := route.endpointStates[registration]
		if state.probing || now.Before(state.retryAt) {
			continue
		}
		probe := !state.retryAt.IsZero()
		state.probing = probe
		attempt := new(Attempt{manager: m, route: route, registration: registration, state: state, epoch: state.epoch, probe: probe})
		return registration.(*watched.RpcServiceRegistration), attempt, true
	}
	return nil, nil, true
}

func (m *Manager) NextWebEndpoint(webName string) (*watched.WebRegistration, bool) {
	endpoint, configured := m.nextEndpoint(watched.FormatWebRegistrationPrefix(webName))
	if endpoint == nil {
		return nil, configured
	}
	return endpoint.(*watched.WebRegistration), configured
}

func (m *Manager) nextEndpoint(prefix string) (any, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	route := m.routesByPrefix[prefix]
	if route == nil {
		return nil, false
	}

	if len(route.endpoints) == 0 {
		return nil, true
	}

	nextIndex := route.nextIndex
	if nextIndex >= len(route.endpoints) {
		nextIndex = 0
	}
	registration := route.endpoints[nextIndex]
	route.nextIndex = (nextIndex + 1) % len(route.endpoints)
	return registration, true
}
