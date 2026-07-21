package webproxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http2"

	"go.yorun.ai/vine/internal/app"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const PathIn = "/web/proxy/in"

type WebProxy struct {
	app.BaseModule

	Context   context.Context   `inject:""`
	AppMinder *minder.AppMinder `inject:""`

	mutex     sync.RWMutex
	transport http.RoundTripper

	appStateByInstanceID map[string]*_AppState
}

type _AppState struct {
	webRouteByName map[string]*_WebRoute
}

type _WebRoute struct {
	endpoint string
	path     string
}

func (p *WebProxy) DIInit() {
	p.appStateByInstanceID = map[string]*_AppState{}
	p.transport = &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network string, addr string, _ *tls.Config) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
	}
	p.AppMinder.AddMutator(p)
}

func (p *WebProxy) IngressPathPrefixRoute() (string, http.Handler) {
	return PathIn, http.HandlerFunc(p.handleIn)
}

func (p *WebProxy) OnSetup(instance *minder.AppInstance) {
	appInstanceID := instance.AppInfo.InstanceId()
	for idx := range instance.HubWebHandlers {
		webSkelName := instance.HubWebHandlers[idx].WebSkelName
		instance.HubWebHandlers[idx].Endpoint = inEndpointOf(instance.IngressEndpoint, appInstanceID, webSkelName)
	}

	appState := &_AppState{
		webRouteByName: map[string]*_WebRoute{},
	}
	for _, webHandler := range instance.WebHandlers {
		endpointPrefix := instance.WebEndpointPrefix
		endpoint := endpointPrefix + "/" + webHandler.WebSkelName
		path := endpoint
		if webinproc.IsEndpoint(endpointPrefix) {
			endpoint = endpointPrefix
			path = "/" + webHandler.WebSkelName
		}
		route := &_WebRoute{
			endpoint: endpoint,
			path:     path,
		}
		appState.webRouteByName[webHandler.WebSkelName] = route
	}

	p.mutex.Lock()
	p.removeAppStateLocked(appInstanceID)
	p.appStateByInstanceID[appInstanceID] = appState
	p.mutex.Unlock()
}

func (p *WebProxy) OnDrain(instance *minder.AppInstance) {
	p.mutex.Lock()
	p.removeAppStateLocked(instance.AppInfo.InstanceId())
	p.mutex.Unlock()
}

func (*WebProxy) OnDestroy(*minder.AppInstance) {}

func (p *WebProxy) webRouteByInstanceID(appInstanceID string, webSkelName string) (*_WebRoute, bool) {
	p.mutex.RLock()
	appState := p.appStateByInstanceID[appInstanceID]
	p.mutex.RUnlock()
	if appState == nil {
		return nil, false
	}

	route := appState.webRouteByName[webSkelName]
	if route == nil {
		return nil, false
	}
	return route, true
}

func (p *WebProxy) removeAppStateLocked(appInstanceID string) {
	if _, ok := p.appStateByInstanceID[appInstanceID]; ok {
		delete(p.appStateByInstanceID, appInstanceID)
	}
}
