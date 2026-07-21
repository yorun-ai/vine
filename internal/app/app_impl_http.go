package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/util/vpre"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	httpShutdownTimeout   = time.Second * 10
	defaultHTTPListenAddr = "127.0.0.1:0"
)

// newHTTPListener is replaceable in tests so app startup can exercise HTTP
// server lifecycle without binding a real local port.
var newHTTPListener = net.Listen

func (a *_AppImpl) startHTTPServer() {
	listenAddr := a.listenAddr
	if listenAddr == "" {
		listenAddr = defaultHTTPListenAddr
	}
	serverName := a.spec.Name()

	listener, err := newHTTPListener("tcp", listenAddr)
	vpre.Check(err == nil, "%s http server listen failed: %v", serverName, err)

	tcpAddr := listener.Addr().(*net.TCPAddr)
	a.httpHost = tcpAddr.IP.String()
	a.httpPort = tcpAddr.Port
	a.httpServer = &http.Server{
		Addr:    net.JoinHostPort(a.httpHost, strconv.Itoa(a.httpPort)),
		Handler: h2c.NewHandler(a.httpHandler(), &http2.Server{}),
	}

	a.httpWG.Add(1)
	go func() {
		defer a.httpWG.Done()

		logger.Info(serverName+" http server started", "addr", a.httpServer.Addr)
		err := a.httpServer.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			logger.Debug(serverName+" http server stopped", "addr", a.httpServer.Addr)
			return
		}
		if err != nil {
			logger.Error(serverName+" http server failed", "addr", a.httpServer.Addr, "error", err)
		}
	}()
}

func (a *_AppImpl) httpHandler() http.Handler {
	return http.HandlerFunc(a.serveHTTP)
}

func (a *_AppImpl) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// RPC clients and gateways are expected to use h2c prior knowledge. If this log appears,
	// inspect the client or gateway implementation first, unless the request is an intentional browser access.
	if r.ProtoMajor == 1 || strings.EqualFold(r.Header.Get("Upgrade"), "h2c") {
		logger.Info("application received non-prior-h2c request",
			"method", r.Method,
			"path", r.URL.Path,
			"proto", r.Proto,
			"upgrade", r.Header.Get("Upgrade"),
		)
	}
	if a.serveHTTPRoutes(w, r) {
		return
	}
	http.NotFound(w, r)
}

func (a *_AppImpl) serveHTTPRoutes(w http.ResponseWriter, r *http.Request) bool {
	var matched *_ServerRoute
	for i := range a.routes {
		route := &a.routes[i]
		if route.HttpHandler == nil {
			continue
		}
		if !route.match(r.URL.Path) {
			continue
		}
		if matched == nil || len(route.Prefix) > len(matched.Prefix) {
			matched = route
		}
	}
	if matched == nil {
		return false
	}
	matched.serveHTTP(w, r)
	return true
}

func (a *_AppImpl) stopHTTPServer() {
	if a.httpServer == nil {
		return
	}

	timeoutCtx, cancel := context.WithTimeout(a.ctx, httpShutdownTimeout)
	defer cancel()
	if err := a.httpServer.Shutdown(timeoutCtx); err != nil {
		logger.Error(a.spec.Name()+" http server shutdown failed", "addr", a.httpServer.Addr, "error", err)
	}
	a.httpWG.Wait()
}
