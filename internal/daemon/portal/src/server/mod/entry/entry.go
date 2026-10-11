package entry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/listenip"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/internal/util/httputil"
)

const (
	entryReadHeaderTimeout = 10 * time.Second
	entryIdleTimeout       = 2 * time.Minute
	entryMaxHeaderBytes    = 128 << 10
)

var (
	entryLogger    = logger.New("daemon:portal:entry")
	listenEntryTCP = net.Listen
)

type _Entry struct {
	listenIP string
	scheme   spec.Scheme
	port     int
	vault    *vault.Vault

	mutex sync.RWMutex
	rules []*_Rule
	hosts []string

	server   *http.Server
	listener net.Listener
	addr     string
	started  bool
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
		scheme:   e.scheme,
		listenIP: e.listenIP,
		port:     e.port,
	}
}

func (e *_Entry) SetOrUpdateRouting(hosts []string, rules []*_Rule) {
	sort.SliceStable(rules, func(i int, j int) bool {
		left := rules[i]
		right := rules[j]
		if left.hostPriority() != right.hostPriority() {
			return left.hostPriority() > right.hostPriority()
		}
		if (left.autoHTTPSPort != 0) != (right.autoHTTPSPort != 0) {
			return left.autoHTTPSPort != 0
		}
		if len(left.matchPathPrefix) != len(right.matchPathPrefix) {
			return len(left.matchPathPrefix) > len(right.matchPathPrefix)
		}
		return left.name < right.name
	})

	e.mutex.Lock()
	e.rules = rules
	e.hosts = hosts
	e.mutex.Unlock()
}

func (e *_Entry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host == "" {
		http.Error(w, "missing Host", http.StatusBadRequest)
		return
	}
	if e.scheme == spec.SchemeHTTPS && (r.TLS == nil || !e.acceptsSNI(r.TLS.ServerName) || normalizeHost(requestHost(r)) != normalizeHost(r.TLS.ServerName)) {
		http.Error(w, "request host does not match TLS entry", http.StatusMisdirectedRequest)
		return
	}
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
	hosts := e.hosts
	e.mutex.RUnlock()

	var selectedHost string
	var ok bool
	if e.scheme == spec.SchemeHTTPS {
		if r.TLS == nil {
			return nil, false
		}
		selectedHost, ok = matchSNI(hosts, r.TLS.ServerName)
	} else {
		selectedHost, ok = matchEntryHost(hosts, requestHost(r))
	}
	if !ok {
		return nil, false
	}
	for _, rule := range rules {
		if normalizeHost(rule.matchHost) != selectedHost {
			continue
		}
		if rule.Matches(r) {
			return rule, true
		}
	}
	return nil, false
}

func (e *_Entry) prepareRequest(r *http.Request) *http.Request {
	request := r.Clone(r.Context())
	request.URL.Scheme = string(e.scheme)
	return request
}

func (e *_Entry) Start() error {
	if e.started {
		return nil
	}

	network, listenAddr := listenip.Binding(e.listenIP, e.port)
	server := newEntryHTTPServer(listenAddr, e)
	if e.scheme == spec.SchemeHTTPS {
		server.TLSConfig = &tls.Config{
			GetConfigForClient: e.getTLSConfig,
			GetCertificate:     e.vault.GetCertificate,
		}
	}
	listener, err := listenEntryTCP(network, listenAddr)
	if err != nil {
		return fmt.Errorf("portal entry %s listener %s failed: %w", e.scheme, listenAddr, err)
	}
	e.server = server
	e.listener = listener
	e.addr = listener.Addr().String()
	e.started = true

	addr := e.addr
	go func() {
		entryLogger.Info("vine.portal entry started", "addr", addr)
		err := e.serve(server, listener)
		if errors.Is(err, http.ErrServerClosed) {
			entryLogger.Debug("vine.portal entry stopped", "addr", addr)
			return
		}
		if err != nil {
			entryLogger.Error("vine.portal entry failed", "addr", addr, "error", err)
		}
	}()
	return nil
}

func newEntryHTTPServer(addr string, handler http.Handler) *http.Server {
	server := httputil.NewServer(addr, handler)
	server.ReadHeaderTimeout = entryReadHeaderTimeout
	server.IdleTimeout = entryIdleTimeout
	server.MaxHeaderBytes = entryMaxHeaderBytes
	return server
}

func (e *_Entry) serve(server *http.Server, listener net.Listener) error {
	if e.scheme == spec.SchemeHTTPS {
		return server.ServeTLS(listener, "", "")
	}
	return server.Serve(listener)
}

// getTLSConfig gates every handshake, including resumed sessions, before the
// vault selects a certificate.
func (e *_Entry) getTLSConfig(hello *tls.ClientHelloInfo) (*tls.Config, error) {
	if !e.acceptsSNI(hello.ServerName) {
		return nil, errors.New("no HTTPS entry matches SNI")
	}
	return nil, nil
}

// acceptsSNI checks configured entries on this listener, independently of routes
// and certificates. Empty names and IP literals cannot select an HTTPS entry.
func (e *_Entry) acceptsSNI(serverName string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	_, ok := matchSNI(e.hosts, serverName)
	return ok
}

func matchSNI(hosts []string, serverName string) (string, bool) {
	host := normalizeHost(serverName)
	if host == "" || isIPHost(host) {
		return "", false
	}
	return matchEntryHost(hosts, host)
}

// matchEntryHost selects an entry independently of its rules. An exact host
// takes precedence over wildcard hosts and the empty-host HTTP IP entry, even
// when the selected entry has no rule matching the requested path.
func matchEntryHost(hosts []string, host string) (string, bool) {
	selected := ""
	matched := false
	for _, pattern := range hosts {
		if !matchesHost(pattern, host) {
			continue
		}
		if pattern != "" && !strings.HasPrefix(pattern, "*.") {
			return normalizeHost(pattern), true
		}
		if !matched || pattern != "" {
			selected, matched = normalizeHost(pattern), true
		}
	}
	return selected, matched
}

func (e *_Entry) Stop() {
	if !e.started {
		return
	}
	server := e.server
	listener := e.listener
	defer listener.Close()
	addr := e.addr

	ctx, cancel := context.WithTimeout(context.Background(), entryShutdownTimeout)
	defer cancel()

	e.started = false
	e.server = nil
	e.listener = nil
	err := httputil.ShutdownServer(server, ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	if err != nil {
		entryLogger.Error("vine.portal entry shutdown failed, force closed", "addr", addr, "error", err)
	}
}
