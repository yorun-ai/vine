package rpcgw

import (
	"context"
	"net/http"
	"sync"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/util/httputil"
)

const pathInvoke = "/invoke"
const pathInspect = "/inspect"
const pathMcp = "/mcp"

var optionsAllowedMethods = []string{http.MethodPost}

type RpcGateway struct {
	name           string
	actorSkelName  string
	inspectEnabled bool
	mcpEnabled     bool
	cors           watched.PortalCors

	app     meta.App
	context context.Context
	cancel  context.CancelFunc
	access  *access.Access
	desc    *desc.Manager
	epmgr   *epmgr.Manager

	mutex    sync.Mutex
	watchers map[string]*epmgr.Watcher
}

func New(ctx context.Context, appInfo meta.App, accessManager *access.Access, descriptors *desc.Manager, epmgrManager *epmgr.Manager, config watched.PortalSite) *RpcGateway {
	gatewayCtx, cancel := context.WithCancel(ctx)
	gateway := &RpcGateway{
		context: gatewayCtx,
		cancel:  cancel,
		app:     appInfo,
		access:  accessManager,
		desc:    descriptors,
		epmgr:   epmgrManager,
	}
	gateway.init(config)
	return gateway
}

func (g *RpcGateway) init(config watched.PortalSite) {
	g.name = config.Name
	g.actorSkelName = config.ActorSkelName
	g.inspectEnabled = config.RpcgwConfig.InspectEnabled
	g.mcpEnabled = config.RpcgwConfig.McpEnabled
	g.cors = config.Cors
	g.watchers = map[string]*epmgr.Watcher{}
	for _, service := range config.RpcgwConfig.Services {
		g.watchService(service.SkelName)
	}
}

func (g *RpcGateway) Update(config watched.PortalSite) bool {
	if config.Type != "RPCGW" {
		return false
	}
	serviceNames := map[string]struct{}{}
	for _, service := range config.RpcgwConfig.Services {
		serviceNames[service.SkelName] = struct{}{}
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.actorSkelName = config.ActorSkelName
	g.inspectEnabled = config.RpcgwConfig.InspectEnabled
	g.mcpEnabled = config.RpcgwConfig.McpEnabled
	g.cors = config.Cors
	for serviceName, watcher := range g.watchers {
		if _, ok := serviceNames[serviceName]; ok {
			continue
		}
		watcher.Release()
		delete(g.watchers, serviceName)
	}
	for serviceName := range serviceNames {
		if _, ok := g.watchers[serviceName]; !ok {
			g.watchers[serviceName] = g.epmgr.WatchRpc(serviceName)
		}
	}
	return true
}

func (g *RpcGateway) watchService(serviceName string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, ok := g.watchers[serviceName]; ok {
		return
	}
	g.watchers[serviceName] = g.epmgr.WatchRpc(serviceName)
}

func (g *RpcGateway) routeService(serviceName string) (*watched.RpcServiceRegistration, bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if _, configured := g.watchers[serviceName]; !configured {
		return nil, false
	}
	return g.epmgr.NextRpcEndpoint(serviceName)
}

func (g *RpcGateway) Name() string {
	return g.name
}

func (g *RpcGateway) Serve(ctx *spec.Context) {
	g.mutex.Lock()
	cors := g.cors
	g.mutex.Unlock()
	path := httputil.PathPrefix(ctx.Request.URL.Path)
	if path == pathMcp {
		g.serveMcp(ctx, cors)
		return
	}
	if ctx.Request.Method == http.MethodOptions {
		methods := optionsAllowedMethods
		if path == pathInspect {
			methods = []string{http.MethodGet}
		}
		spec.ServeOptions(ctx.ResponseWriter, ctx.Request, cors, ctx.EntryOrigin, methods)
		return
	}
	if spec.ApplyCORS(ctx.ResponseWriter, ctx.Request, cors, ctx.EntryOrigin) {
		spec.ExposeHeader(ctx.ResponseWriter.Header(), spec.HeaderPortalTraceId)
	}
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, rpcTraceIdOrNew(ctx.Request.Header))

	mappedCtx := *ctx
	mappedCtx.ResponseWriter = newHTTPStatusMappingResponseWriter(ctx.ResponseWriter)
	switch path {
	case pathInvoke:
		g.serveInvoke(&mappedCtx)
	case pathInspect:
		g.serveInspect(&mappedCtx)
	default:
		g.writeError(mappedCtx.ResponseWriter, mappedCtx.Request, ex.NotFound, "rpcgw path is not found")
	}
}

func (g *RpcGateway) writeError(w http.ResponseWriter, r *http.Request, code ex.Code, message string) {
	_ = rpchttp.WriteRequestErrorResponse(w, r, g.app, ex.New(code, message))
}

func rpcTraceIdOrNew(header http.Header) string {
	traceValue := header.Get(rpchttp.HeaderRpcTrace)
	if traceValue == "" {
		return meta.NewId()
	}
	trace, err := meta.DecodeTraceFromDelimitedOrNewSpan(traceValue)
	if err != nil {
		return meta.NewId()
	}
	return trace.Id()
}

func (g *RpcGateway) Stop() {
	g.mutex.Lock()
	for serviceName, watcher := range g.watchers {
		watcher.Release()
		delete(g.watchers, serviceName)
	}
	g.mutex.Unlock()
	g.cancel()
}
