package webgw

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
)

type WebGateway struct {
	name     string
	actorVia redised.PortalActorVia
	cors     redised.PortalCors
	webName  string

	context context.Context
	cancel  context.CancelFunc
	access  *access.Access
	epmgr   *epmgr.Manager

	mutex   sync.Mutex
	watcher *epmgr.Watcher
}

func New(ctx context.Context, accessManager *access.Access, epmgrManager *epmgr.Manager, config redised.PortalSite) *WebGateway {
	gatewayCtx, cancel := context.WithCancel(ctx)
	gateway := &WebGateway{
		context: gatewayCtx,
		cancel:  cancel,
		access:  accessManager,
		epmgr:   epmgrManager,
	}
	gateway.init(config)
	return gateway
}

func (g *WebGateway) init(config redised.PortalSite) {
	g.name = config.Name
	g.actorVia = config.ActorVia
	g.cors = config.Cors
	g.webName = config.WebgwConfig.WebName
	g.watcher = g.epmgr.WatchWeb(g.webName)
}

func (g *WebGateway) Update(config redised.PortalSite) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.actorVia = config.ActorVia
	g.cors = config.Cors
	nextWebName := config.WebgwConfig.WebName
	if g.webName == nextWebName {
		return true
	}

	g.watcher.Release()

	g.webName = nextWebName
	g.watcher = g.epmgr.WatchWeb(nextWebName)
	return true
}

func (g *WebGateway) Name() string {
	return g.name
}

func (g *WebGateway) routeWeb() (*redised.WebRegistration, bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.epmgr.NextWebEndpoint(g.webName)
}

func (g *WebGateway) Stop() {
	g.mutex.Lock()
	g.watcher.Release()
	g.watcher = nil
	g.mutex.Unlock()
	g.cancel()
}
