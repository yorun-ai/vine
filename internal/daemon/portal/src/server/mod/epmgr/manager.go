package epmgr

import (
	"context"
	"reflect"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
)

type Manager struct {
	app.BaseModule

	Context context.Context  `inject:""`
	Redis   *hubredis.Client `inject:""`

	mutex          sync.Mutex
	routesByPrefix map[string]*_Route
}

var (
	rpcServiceRegistrationType = reflect.TypeFor[redised.RpcServiceRegistration]()
	webRegistrationType        = reflect.TypeFor[redised.WebRegistration]()
)

type _Route struct {
	prefix             string
	registrationType   reflect.Type
	cancel             context.CancelFunc
	refCount           int
	registrationsByKey map[string]any
	endpoints          []any
	nextIndex          int
}

func (m *Manager) DIInit() {
	m.routesByPrefix = map[string]*_Route{}
}

func (m *Manager) WatchRpc(serviceName string) *Watcher {
	return m.watch(redised.FormatRpcServiceRegistrationPrefix(serviceName), rpcServiceRegistrationType)
}

func (m *Manager) WatchWeb(webName string) *Watcher {
	return m.watch(redised.FormatWebRegistrationPrefix(webName), webRegistrationType)
}

func (m *Manager) NextRpcEndpoint(serviceName string) (*redised.RpcServiceRegistration, bool) {
	endpoint, configured := m.nextEndpoint(redised.FormatRpcServiceRegistrationPrefix(serviceName))
	if endpoint == nil {
		return nil, configured
	}
	return endpoint.(*redised.RpcServiceRegistration), configured
}

func (m *Manager) NextWebEndpoint(webName string) (*redised.WebRegistration, bool) {
	endpoint, configured := m.nextEndpoint(redised.FormatWebRegistrationPrefix(webName))
	if endpoint == nil {
		return nil, configured
	}
	return endpoint.(*redised.WebRegistration), configured
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
