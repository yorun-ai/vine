package controlapi

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
	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/daemon"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	impl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/control"
	"go.yorun.ai/vine/internal/util/httputil"
)

const shutdownTimeout = 10 * time.Second

var (
	controlLogger = logger.New("daemon:hub:controlapi")
	listenTCP     = net.Listen
)

// Server exposes only the Hub Control API used by Link and Portal. Hub's
// admin Rpc services and Dashboard Web handler remain on the main Hub
// application listener and are deliberately absent from this server.
type Server struct {
	app.BaseModule

	Context         context.Context         `inject:""`
	Flag            *flag.Flag              `inject:""`
	InprocFlag      *app.InternalInprocFlag `inject:""`
	InternalRuntime app.InternalRuntime     `inject:""`
	Identity        *mtls.Identity          `inject:""`

	rpcHTTPHandler http.Handler
	rpcHandler     rpcspec.RpcHandler
	inprocEndpoint string
	inprocCleanup  func()
	httpServer     *http.Server
	wg             sync.WaitGroup
}

func (s *Server) BeforeAppStart() error {
	s.rpcHTTPHandler, s.rpcHandler = s.InternalRuntime.AdditionalServicer(
		app.T[*impl.InfoServiceServerImpl](),
		app.T[*impl.RegistryServiceServerImpl](),
		app.T[*impl.LockServiceServerImpl](),
	)

	if s.InprocFlag.Enabled {
		s.startInproc()
		return nil
	}
	return s.startHTTP()
}

func (s *Server) BeforeAppStop() {
	if s.InprocFlag.Enabled {
		s.stopInproc()
	} else {
		s.stopHTTP()
	}
	s.rpcHTTPHandler = nil
	s.rpcHandler = nil
}

func (s *Server) startInproc() {
	s.inprocEndpoint = rpcinproc.Endpoint(hubapp.HubControlInprocHostPath, coreapp.PathRpcInvoke)
	s.inprocCleanup = rpcinproc.Register(s.inprocEndpoint, s.rpcHandler)
	controlLogger.Info("hub control API server started", "endpoint", s.inprocEndpoint)
}

func (s *Server) stopInproc() {
	if s.inprocEndpoint == "" {
		return
	}
	s.inprocCleanup()
	controlLogger.Debug("hub control API server stopped", "endpoint", s.inprocEndpoint)
	s.inprocEndpoint = ""
	s.inprocCleanup = nil
}

func (s *Server) startHTTP() error {
	listener, err := listenTCP("tcp", s.Flag.ControlListen)
	if err != nil {
		s.rpcHTTPHandler = nil
		s.rpcHandler = nil
		return fmt.Errorf("hub control API listen failed: %w", err)
	}
	server := httputil.NewServer(listener.Addr().String(), nil)
	serve := server.Serve
	if s.Identity.Enabled() {
		server.Handler = s
		server.TLSConfig = s.Identity.ServerConfig(daemon.LinkIdentity.SPIFFEPath(), daemon.PortalIdentity.SPIFFEPath())
		if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
			_ = listener.Close()
			return fmt.Errorf("hub control API HTTP/2 configure failed: %w", err)
		}
		serve = func(listener net.Listener) error { return server.ServeTLS(listener, "", "") }
	} else {
		server.Handler = h2c.NewHandler(s, &http2.Server{})
	}
	s.httpServer = server

	s.wg.Go(func() {
		controlLogger.Info("hub control API server started", "addr", server.Addr)
		err := serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			controlLogger.Debug("hub control API server stopped", "addr", server.Addr)
			return
		}
		if err != nil {
			controlLogger.Error("hub control API server failed", "addr", server.Addr, "error", err)
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
		controlLogger.Error("hub control API server graceful shutdown failed, force closed", "addr", s.httpServer.Addr, "error", err)
	}
	s.wg.Wait()
	s.httpServer = nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	prefix := coreapp.PathRpcInvoke
	if request.URL.Path != prefix && !strings.HasPrefix(request.URL.Path, prefix+"/") {
		http.NotFound(w, request)
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
