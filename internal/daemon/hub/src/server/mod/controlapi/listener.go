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

// Listener exposes only the Hub Control API used by Link and Portal. Hub's
// admin Rpc services and Dashboard Web handler remain on the main Hub
// application listener and are deliberately absent from this server.
type Listener struct {
	app.BaseModule

	Context         context.Context         `inject:""`
	Flag            *flag.Flag              `inject:""`
	InprocFlag      *app.InternalInprocFlag `inject:""`
	InternalRuntime app.InternalRuntime     `inject:""`
	Identity        *mtls.Identity          `inject:""`

	rpcHTTPHandler http.Handler
	rpcHandler     rpcspec.RpcHandler
	inprocEndpoint string
	server         *http.Server
	wg             sync.WaitGroup
}

func (l *Listener) BeforeAppStart() error {
	l.rpcHTTPHandler, l.rpcHandler = l.InternalRuntime.AdditionalServicer(
		app.T[*impl.InfoServiceServerImpl](),
		app.T[*impl.RegistryServiceServerImpl](),
	)

	if l.InprocFlag.Enabled {
		l.startInproc()
		return nil
	}
	return l.startHTTP()
}

func (l *Listener) BeforeAppStop() {
	if l.InprocFlag.Enabled {
		l.stopInproc()
	} else {
		l.stopHTTP()
	}
	l.rpcHTTPHandler = nil
	l.rpcHandler = nil
}

func (l *Listener) startInproc() {
	l.inprocEndpoint = rpcinproc.Endpoint(hubapp.HubControlInprocHostPath, coreapp.PathRpcInvoke)
	rpcinproc.Register(l.inprocEndpoint, l.rpcHandler)
	controlLogger.Info("hub control API listener started", "endpoint", l.inprocEndpoint)
}

func (l *Listener) stopInproc() {
	if l.inprocEndpoint == "" {
		return
	}
	rpcinproc.Unregister(l.inprocEndpoint)
	controlLogger.Debug("hub control API listener stopped", "endpoint", l.inprocEndpoint)
	l.inprocEndpoint = ""
}

func (l *Listener) startHTTP() error {
	listener, err := listenTCP("tcp", l.Flag.ControlListen)
	if err != nil {
		l.rpcHTTPHandler = nil
		l.rpcHandler = nil
		return fmt.Errorf("hub control API listen failed: %w", err)
	}
	server := &http.Server{
		Addr: listener.Addr().String(),
	}
	serve := server.Serve
	if l.Identity.Enabled() {
		server.Handler = l
		server.TLSConfig = l.Identity.ServerConfig(daemon.LinkIdentity.SPIFFEPath(), daemon.PortalIdentity.SPIFFEPath())
		if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
			_ = listener.Close()
			return fmt.Errorf("hub control API HTTP/2 configure failed: %w", err)
		}
		serve = func(listener net.Listener) error { return server.ServeTLS(listener, "", "") }
	} else {
		server.Handler = h2c.NewHandler(l, &http2.Server{})
	}
	l.server = server

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		controlLogger.Info("hub control API listener started", "addr", server.Addr)
		err := serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			controlLogger.Debug("hub control API listener stopped", "addr", server.Addr)
			return
		}
		if err != nil {
			controlLogger.Error("hub control API listener failed", "addr", server.Addr, "error", err)
		}
	}()
	return nil
}

func (l *Listener) stopHTTP() {
	if l.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(l.Context, shutdownTimeout)
	defer cancel()
	if err := httputil.ShutdownServer(l.server, ctx); err != nil {
		controlLogger.Error("hub control API listener graceful shutdown failed, force closed", "addr", l.server.Addr, "error", err)
	}
	l.wg.Wait()
	l.server = nil
}

func (l *Listener) ServeHTTP(w http.ResponseWriter, request *http.Request) {
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
	l.rpcHTTPHandler.ServeHTTP(w, next)
}
