package ingress

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"go.yorun.ai/vine/internal/app"
	corelink "go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/webproxy"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const ingressShutdownTimeout = 10 * time.Second

// detectHostIP is replaced in tests to make ingress endpoint generation deterministic.
var detectHostIP = func() string {
	return vnet.DetectHostIP()
}

type Ingress struct {
	app.BaseModule

	Context  context.Context    `inject:""`
	Flag     *flag.Flag         `inject:""`
	RpcProxy *rpcproxy.RpcProxy `inject:""`
	WebProxy *webproxy.WebProxy `inject:""`

	httpServer *http.Server
	httpWG     sync.WaitGroup
	routes     []_IngressRoute
	endpoint   string
}

type _IngressRoute struct {
	prefix  string
	handler http.Handler
}

func (g *Ingress) DIInit() {
	g.appendRoute(g.RpcProxy.IngressPathPrefixRoute())
	g.appendRoute(g.WebProxy.IngressPathPrefixRoute())
}

func (g *Ingress) AfterAppStart() {
	if g.Flag.HubInprocMode {
		g.startInprocServer()
		return
	}
	g.startHTTPServer()
}

func (g *Ingress) BeforeAppStop() {
	if g.Flag.HubInprocMode {
		g.stopInprocServer()
		return
	}
	g.stopHTTPServer()
}

func (g *Ingress) Endpoint() string {
	return g.endpoint
}

func (g *Ingress) startHTTPServer() {
	listener, err := net.Listen("tcp", g.Flag.IngressListen)
	vpre.CheckNilError(err, "link ingress server listen failed")

	port := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: h2c.NewHandler(g.httpHandler(), &http2.Server{}),
	}
	host := g.mustDetectHost()

	g.httpServer = server
	g.endpoint = endpointOfHostPort(host, port)

	g.httpWG.Add(1)
	go func() {
		defer g.httpWG.Done()

		logger.Info("link ingress server started", "addr", server.Addr)
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			logger.Debug("link ingress server stopped", "addr", server.Addr)
			return
		}
		if err != nil {
			logger.Error("link ingress server failed", "addr", server.Addr, "error", err)
		}
	}()
}

func (g *Ingress) startInprocServer() {
	g.endpoint = ingressinproc.Endpoint(corelink.InprocHostPath)
	ingressinproc.Register(g.endpoint, g.httpHandler())
}

func (g *Ingress) mustDetectHost() string {
	return detectHostIP()
}

func (g *Ingress) httpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.serveHTTPRoutes(w, r) {
			return
		}
		http.NotFound(w, r)
	})
}

func (g *Ingress) appendRoute(prefix string, handler http.Handler) {
	g.routes = append(g.routes, _IngressRoute{
		prefix:  prefix,
		handler: handler,
	})
}

func (g *Ingress) serveHTTPRoutes(w http.ResponseWriter, r *http.Request) bool {
	var matched *_IngressRoute
	for i := range g.routes {
		route := &g.routes[i]
		if !matchRoute(route.prefix, r.URL.Path) {
			continue
		}
		if matched == nil || len(route.prefix) > len(matched.prefix) {
			matched = route
		}
	}
	if matched == nil {
		return false
	}
	serveRoute(*matched, w, r)
	return true
}

func (g *Ingress) stopHTTPServer() {
	server := g.httpServer

	timeoutCtx, cancel := context.WithTimeout(context.Background(), ingressShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(timeoutCtx); err != nil {
		logger.Error("link ingress server shutdown failed", "addr", server.Addr, "error", err)
	}
	g.httpWG.Wait()

	g.httpServer = nil
	g.endpoint = ""
}

func (g *Ingress) stopInprocServer() {
	ingressinproc.Unregister(g.endpoint)
	g.endpoint = ""
}

func (g *Ingress) String() string {
	return fmt.Sprintf("Ingress(%s)", g.Endpoint())
}

func endpointOfHostPort(host string, port int) string {
	return fmt.Sprintf("http://%s:%d", host, port)
}

func matchRoute(prefix string, path string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func serveRoute(route _IngressRoute, w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, route.prefix)
	next := req.Clone(req.Context())
	next.URL.Path = path
	next.RequestURI = path
	route.handler.ServeHTTP(w, next)
}
