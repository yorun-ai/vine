package rpcproxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http2"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const (
	PathIn  = "/rpc/proxy/in"
	PathOut = "/rpc/proxy/out"
)

type RpcProxy struct {
	app.BaseModule

	Context     context.Context   `inject:""`
	RedisClient *hubredis.Client  `inject:""`
	App         runtime.App       `inject:""`
	Logger      *logger.Logger    `inject:""`
	AppMinder   *minder.AppMinder `inject:""`

	transport http.RoundTripper

	appStateMutex        sync.RWMutex
	appStateByInstanceID map[string]*_AppState

	serviceStateMutex   sync.RWMutex
	serviceStatesByName map[string]*_ServiceState
}

type _AppState struct {
	instance        *minder.AppInstance
	appInfo         runtime.App
	serviceEndpoint string
	serviceNames    map[string]struct{}
	draining        bool
}

type _ServiceState struct {
	refsByAppInstanceID map[string]struct{}
	registrationsByKey  map[string]redised.RpcServiceRegistration
	endpoints           []redised.RpcServiceRegistration
	nextIndex           int
	cancel              context.CancelFunc
}

func (p *RpcProxy) DIInit() {
	p.transport = &http2.Transport{
		AllowHTTP:          true,
		DisableCompression: true,
		DialTLSContext: func(ctx context.Context, network string, addr string, _ *tls.Config) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
	}
	p.appStateByInstanceID = map[string]*_AppState{}
	p.serviceStatesByName = map[string]*_ServiceState{}
	p.AppMinder.AddMutator(p)
}

func (p *RpcProxy) OnSetup(instance *minder.AppInstance) {
	appInstanceID := instance.AppInfo.InstanceId()
	for idx := range instance.HubServiceHandlers {
		instance.HubServiceHandlers[idx].Endpoint = instance.IngressEndpoint + PathIn + "/" + appInstanceID
	}
	serviceNames := map[string]struct{}{}
	for _, serviceHandler := range instance.ServiceHandlers {
		serviceNames[serviceHandler.ServiceSkelName] = struct{}{}
	}
	p.appStateMutex.Lock()
	state := p.appStateByInstanceID[appInstanceID]
	if state == nil {
		state = &_AppState{}
		p.appStateByInstanceID[appInstanceID] = state
	}
	state.instance = instance
	state.appInfo = instance.AppInfo
	state.serviceEndpoint = instance.ServiceEndpoint
	state.serviceNames = serviceNames
	state.draining = false
	p.appStateMutex.Unlock()
}

func (p *RpcProxy) OnDrain(instance *minder.AppInstance) {
	p.appStateMutex.Lock()
	if state := p.appStateByInstanceID[instance.AppInfo.InstanceId()]; state != nil {
		state.draining = true
	}
	p.appStateMutex.Unlock()
}

func (p *RpcProxy) OnDestroy(instance *minder.AppInstance) {
	p.appStateMutex.Lock()
	delete(p.appStateByInstanceID, instance.AppInfo.InstanceId())
	p.appStateMutex.Unlock()
	p.releaseInstanceState(instance.AppInfo.InstanceId())
}

func (p *RpcProxy) getAppStateByInstanceID(appInstanceID string) (*_AppState, bool) {
	p.appStateMutex.RLock()
	state := p.appStateByInstanceID[appInstanceID]
	p.appStateMutex.RUnlock()
	if state == nil {
		return nil, false
	}
	return state, true
}

func (p *RpcProxy) InitPathPrefixRoute(add app.PathPrefixRouteAdder) {
	add(PathOut, http.HandlerFunc(p.handleOut), spec.RpcHandlerFunc(p.serveRpcOut))
}

func (p *RpcProxy) IngressPathPrefixRoute() (string, http.Handler) {
	return PathIn, http.HandlerFunc(p.handleIn)
}

func (s *_AppState) hasService(serviceName string) bool {
	_, ok := s.serviceNames[serviceName]
	return ok
}
