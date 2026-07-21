package entry

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/util/vpre"
)

var listenEntryTCP = net.Listen

type _Entry struct {
	scheme spec.Scheme
	port   int
	vault  *vault.Vault

	mutex sync.RWMutex
	rules []*_Rule

	server  *http.Server
	addr    string
	started bool
}

func newEntry(scheme spec.Scheme, port int, vault *vault.Vault) *_Entry {
	return &_Entry{
		scheme: scheme,
		port:   port,
		vault:  vault,
	}
}

func (e *_Entry) Key() _Key {
	return _Key{
		scheme: e.scheme,
		port:   e.port,
	}
}

func (e *_Entry) SetOrUpdateRules(rules []*_Rule) {
	sort.SliceStable(rules, func(i int, j int) bool {
		left := rules[i]
		right := rules[j]
		if len(left.pathPrefix) != len(right.pathPrefix) {
			return len(left.pathPrefix) > len(right.pathPrefix)
		}
		return left.name < right.name
	})

	e.mutex.Lock()
	e.rules = rules
	e.mutex.Unlock()
}

func (e *_Entry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request := e.prepareRequest(r)
	if rule, ok := e.route(request); ok {
		rule.Serve(&spec.Context{
			Request:        request,
			ResponseWriter: w,
			RemoteAddr:     clientIP(request),
		})
		return
	}
	http.NotFound(w, r)
}

func (e *_Entry) route(r *http.Request) (*_Rule, bool) {
	e.mutex.RLock()
	rules := e.rules
	e.mutex.RUnlock()

	for _, rule := range rules {
		if rule.Matches(r) {
			return rule, true
		}
	}
	return nil, false
}

func (e *_Entry) prepareRequest(r *http.Request) *http.Request {
	request := r.Clone(r.Context())
	url := *request.URL
	url.Scheme = string(e.scheme)
	request.URL = &url
	return request
}

func (e *_Entry) Start() {
	if e.started {
		return
	}

	listenAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(e.port))
	server := &http.Server{
		Addr:    listenAddr,
		Handler: e,
	}
	if e.scheme == spec.SchemeHTTPS {
		server.TLSConfig = &tls.Config{
			GetCertificate: e.getCertificate,
		}
	}
	listener, err := listenEntryTCP("tcp", listenAddr)
	vpre.Check(err == nil, "portal entry server listen failed: %v", err)
	e.server = server
	e.addr = listener.Addr().String()
	e.started = true

	go func() {
		logger.Info("vine.portal entry started", "addr", e.addr)
		err := e.serve(server, listener)
		if errors.Is(err, http.ErrServerClosed) {
			logger.Debug("vine.portal entry stopped", "addr", e.addr)
			return
		}
		if err != nil {
			logger.Error("vine.portal entry failed", "addr", e.addr, "error", err)
		}
	}()
}

func (e *_Entry) serve(server *http.Server, listener net.Listener) error {
	if e.scheme == spec.SchemeHTTPS {
		return server.ServeTLS(listener, "", "")
	}
	return server.Serve(listener)
}

func (e *_Entry) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return e.vault.GetCertificate(hello)
}

func (e *_Entry) Stop() {
	if !e.started {
		return
	}
	server := e.server
	addr := e.addr

	ctx, cancel := context.WithTimeout(context.Background(), entryShutdownTimeout)
	defer cancel()

	e.started = false
	e.server = nil
	err := server.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	if err != nil {
		_ = server.Close()
		logger.Error("vine.portal entry shutdown failed, force closed", "addr", addr, "error", err)
	}
}
