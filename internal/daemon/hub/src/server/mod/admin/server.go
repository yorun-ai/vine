package admin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	impl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin"
	debugimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin/debug"
	"go.yorun.ai/vine/internal/util/httputil"
)

// shutdownTimeout bounds the graceful stop of the listener. A test shortens it.
var shutdownTimeout = 10 * time.Second

// apiPath is where the admin listener answers the Admin API. The listener owns
// the path, because it owns the Dashboard build that calls it, while the runtime
// path every component reaches other services through belongs to the runtime.
const apiPath = "/api/invoke"

var (
	adminLogger = logger.New("daemon:hub:admin")
	listenTCP   = net.Listen
)

// Server exposes the Hub Admin API on a listener of its own, the way the Hub
// Control API does, so the operator reaches Hub without Link or Portal in the
// path.
type Server struct {
	app.BaseModule

	Context         context.Context     `inject:""`
	Flag            *flag.Flag          `inject:""`
	InternalRuntime app.InternalRuntime `inject:""`

	rpcHTTPHandler    http.Handler
	rpcHandler        rpcspec.RpcHandler
	dashboardHandler  http.Handler
	dashboardDevProxy *_DashboardDevProxy
	httpServer        *http.Server
	wg                sync.WaitGroup
}

func (s *Server) BeforeAppStart() error {
	if s.Flag.AdminListen == "" {
		// An in-process Hub that asks for no listener serves no Admin API.
		return nil
	}
	s.rpcHTTPHandler, s.rpcHandler = s.InternalRuntime.AdditionalServicer(HandlerTypes()...)
	s.dashboardHandler = dashboardHandler()
	s.dashboardDevProxy = dashboardDevProxy()

	return s.startHTTP()
}

// HandlerTypes lists the Admin API services Hub serves. The Control API keeps
// its own list, so the two listeners never share a service.
func HandlerTypes() []reflect.Type {
	return []reflect.Type{
		app.T[*debugimpl.ServiceDebugApiServiceServerImpl](),
		app.T[*debugimpl.TaskDebugApiServiceServerImpl](),
		app.T[*debugimpl.EventDebugApiServiceServerImpl](),
		app.T[*impl.SkeletonApiServiceServerImpl](),
		app.T[*impl.AppStatusApiServiceServerImpl](),
		app.T[*impl.PortalStatusApiServiceServerImpl](),
		app.T[*impl.AppConfigApiServiceServerImpl](),
		app.T[*impl.PortalCertApiServiceServerImpl](),
		app.T[*impl.PortalEntryApiServiceServerImpl](),
		app.T[*impl.PortalRuleApiServiceServerImpl](),
		app.T[*impl.AdminApiServiceServerImpl](),
		app.T[*impl.PortalSiteApiServiceServerImpl](),
	}
}

func (s *Server) BeforeAppStop() {
	// The Dashboard development proxy ends first: it cancels the requests it
	// forwarded, so a request the development server does not answer cannot hold
	// the listener open while it stops.
	if s.dashboardDevProxy != nil {
		s.dashboardDevProxy.Close()
	}
	s.stopHTTP()
	s.rpcHTTPHandler = nil
	s.rpcHandler = nil
	s.dashboardHandler = nil
	s.dashboardDevProxy = nil
}

func (s *Server) startHTTP() error {
	listener, err := listenTCP("tcp", s.Flag.AdminListen)
	if err != nil {
		s.rpcHTTPHandler = nil
		s.rpcHandler = nil
		return fmt.Errorf("hub admin API listen failed: %w", err)
	}
	server := httputil.NewServer(listener.Addr().String(), nil)
	// The Admin API serves an operator's browser, which carries no mesh
	// certificate and does not speak cleartext HTTP/2, so this listener stays
	// plain HTTP/1.1 even when Hub enables backend mTLS for Link and Portal.
	// TODO: Serve HTTP/2 over a server certificate. A browser speaks HTTP/2 only
	// over TLS, and the certificate would encrypt the listener without
	// authenticating the caller the way the backend mTLS identities do.
	server.Handler = s
	s.httpServer = server

	s.wg.Go(func() {
		adminLogger.Info("hub admin API server started", "addr", server.Addr)
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			adminLogger.Debug("hub admin API server stopped", "addr", server.Addr)
			return
		}
		if err != nil {
			adminLogger.Error("hub admin API server failed", "addr", server.Addr, "error", err)
		}
	})
	return nil
}

func (s *Server) stopHTTP() {
	if s.httpServer == nil {
		return
	}

	ctx, cancel := context.WithTimeout(s.Context, shutdownTimeout)
	defer cancel()
	if err := httputil.ShutdownServer(s.httpServer, ctx); err != nil {
		adminLogger.Error("hub admin API server graceful shutdown failed, force closed", "addr", s.httpServer.Addr, "error", err)
	}
	s.wg.Wait()
	s.httpServer = nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if request.URL.Path == apiPath || strings.HasPrefix(request.URL.Path, apiPath+"/") {
		path := strings.TrimPrefix(request.URL.Path, apiPath)
		if path == "" {
			path = "/"
		}
		next := request.Clone(request.Context())
		next.URL.Path = path
		next.RequestURI = path
		s.rpcHTTPHandler.ServeHTTP(w, next)
		return
	}

	// Everything outside the Admin API is the Dashboard, so it reaches Hub on the
	// same origin as the API it calls: the development server a developer runs
	// beside Hub when they asked for one, and the embedded build otherwise.
	if s.dashboardDevProxy != nil && s.dashboardDevProxy.ServeHTTP(w, request) {
		return
	}
	s.dashboardHandler.ServeHTTP(w, request)
}
