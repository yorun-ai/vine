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

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"go.yorun.ai/vine/internal/app"
	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	impl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin"
	debugimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin/debug"
	"go.yorun.ai/vine/internal/util/httputil"
)

const shutdownTimeout = 10 * time.Second

var (
	adminLogger = logger.New("daemon:hub:admin")
	listenTCP   = net.Listen
)

// Server exposes the Hub Admin API on a listener of its own, the way the Hub
// Control API does. Hub serves the Admin API in every mode, so the operator
// reaches Hub without Link or Portal in the path.
type Server struct {
	app.BaseModule

	Context         context.Context         `inject:""`
	Flag            *flag.Flag              `inject:""`
	InprocFlag      *app.InternalInprocFlag `inject:""`
	InternalRuntime app.InternalRuntime     `inject:""`
	Identity        *mtls.Identity          `inject:""`

	rpcHTTPHandler   http.Handler
	rpcHandler       rpcspec.RpcHandler
	dashboardHandler http.Handler
	inprocEndpoint   string
	inprocCleanup    func()
	httpServer       *http.Server
	wg               sync.WaitGroup
}

func (s *Server) BeforeAppStart() error {
	s.rpcHTTPHandler, s.rpcHandler = s.InternalRuntime.AdditionalServicer(HandlerTypes()...)
	s.dashboardHandler = DashboardHandler()

	if s.InprocFlag.Enabled {
		s.startInproc()
		return nil
	}
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
		app.T[*impl.MaintenanceApiServiceServerImpl](),
		app.T[*impl.PortalSiteApiServiceServerImpl](),
	}
}

func (s *Server) BeforeAppStop() {
	if s.InprocFlag.Enabled {
		s.stopInproc()
	} else {
		s.stopHTTP()
	}
	s.rpcHTTPHandler = nil
	s.rpcHandler = nil
	s.dashboardHandler = nil
}

func (s *Server) startInproc() {
	s.inprocEndpoint = rpcinproc.Endpoint(hubapp.HubAdminInprocHostPath, coreapp.PathRpcInvoke)
	s.inprocCleanup = rpcinproc.Register(s.inprocEndpoint, s.rpcHandler)
	adminLogger.Info("hub admin API server started", "endpoint", s.inprocEndpoint)
}

func (s *Server) stopInproc() {
	if s.inprocEndpoint == "" {
		return
	}
	s.inprocCleanup()
	adminLogger.Debug("hub admin API server stopped", "endpoint", s.inprocEndpoint)
	s.inprocEndpoint = ""
	s.inprocCleanup = nil
}

func (s *Server) startHTTP() error {
	listener, err := listenTCP("tcp", s.Flag.AdminListen)
	if err != nil {
		s.rpcHTTPHandler = nil
		s.rpcHandler = nil
		return fmt.Errorf("hub admin API listen failed: %w", err)
	}
	server := httputil.NewServer(listener.Addr().String(), nil)
	serve := server.Serve
	if s.Identity.Enabled() {
		server.Handler = s
		// Hub accepts an Admin API call from any client the mesh CA issued, so an
		// operator is not tied to another component's identity.
		server.TLSConfig = s.Identity.ServerConfig()
		if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
			_ = listener.Close()
			return fmt.Errorf("hub admin API HTTP/2 configure failed: %w", err)
		}
		serve = func(listener net.Listener) error { return server.ServeTLS(listener, "", "") }
	} else {
		server.Handler = h2c.NewHandler(s, &http2.Server{})
	}
	s.httpServer = server

	s.wg.Go(func() {
		adminLogger.Info("hub admin API server started", "addr", server.Addr)
		err := serve(listener)
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
	prefix := coreapp.PathRpcInvoke
	if request.URL.Path != prefix && !strings.HasPrefix(request.URL.Path, prefix+"/") {
		// Everything outside the Admin API is the Dashboard build, so the
		// Dashboard reaches Hub on the same origin as the API it calls.
		s.dashboardHandler.ServeHTTP(w, request)
		return
	}

	path := strings.TrimPrefix(request.URL.Path, prefix)
	if path == "" {
		path = "/"
	}
	next := request.Clone(request.Context())
	next.URL.Path = path
	next.RequestURI = path
	s.rpcHTTPHandler.ServeHTTP(w, next)
}
