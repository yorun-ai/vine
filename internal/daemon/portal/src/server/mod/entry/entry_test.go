package entry

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

type _TestSite struct {
	name string
}

func (s *_TestSite) Name() string {
	return s.name
}

func (s *_TestSite) Serve(ctx *spec.Context) {
}

func (s *_TestSite) Update(config watched.PortalSite) bool {
	return false
}

func (s *_TestSite) Stop() {
}

func TestEntryRouteMatchesAttachedRules(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		scheme: spec.SchemeHTTPS,
		port:   8443,
		hosts:  []string{"demo.local"},
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTPS,
			matchHost:       "demo.local",
			matchPort:       8443,
			matchPathPrefix: "/admin",
			redirectionSite: target,
		}},
	}

	request := newTestRequest("https://demo.local:8443/admin/users")
	request.TLS = &tls.ConnectionState{ServerName: "demo.local"}
	rule, ok := entry.route(request)

	assert.True(t, ok)
	assert.Same(t, target, rule.redirectionSite)
}

func TestEntryRouteEmptyHostOnlyMatchesHTTPIP(t *testing.T) {
	for _, scheme := range []spec.Scheme{spec.SchemeHTTP, spec.SchemeHTTPS} {
		entry := newEntry(scheme, 8080, nil)
		entry.SetOrUpdateRouting([]string{""}, []*_Rule{{matchScheme: scheme, matchPathPrefix: "/admin"}})
		for _, host := range []string{"127.0.0.1", "[::1]", "demo.local"} {
			_, ok := entry.route(newTestRequest(string(scheme) + "://" + host + ":8080/admin/users"))
			assert.Equal(t, scheme == spec.SchemeHTTP && host != "demo.local", ok, "%s %s", scheme, host)
		}
	}
}

func TestEntryRouteDoesNotMatchPartialPathPrefix(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		scheme: spec.SchemeHTTPS,
		port:   8443,
		hosts:  []string{"demo.local"},
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTPS,
			matchHost:       "demo.local",
			matchPort:       8443,
			matchPathPrefix: "/admin",
			redirectionSite: target,
		}},
	}

	request := newTestRequest("https://demo.local:8443/admin2/users")
	request.TLS = &tls.ConnectionState{ServerName: "demo.local"}
	_, ok := entry.route(request)

	assert.False(t, ok)
}

func TestEntryServesRedirectRule(t *testing.T) {
	entry := &_Entry{
		scheme: spec.SchemeHTTP,
		port:   8080,
		hosts:  []string{"demo.local"},
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTP,
			matchHost:       "demo.local",
			matchPort:       8080,
			matchPathPrefix: "/old",
			redirectionSite: site.NewRedirectionSite(true, "https://demo.local/new"),
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local:8080/old/page", nil)
	entry.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusPermanentRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/new", recorder.Header().Get("Location"))
	assert.NotContains(t, recorder.Body.String(), "404")
}

func TestEntryServesRedirectRuleWithoutTrimmedPathPrefix(t *testing.T) {
	entry := &_Entry{
		scheme: spec.SchemeHTTP,
		port:   8080,
		hosts:  []string{"demo.local"},
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTP,
			matchHost:       "demo.local",
			matchPort:       8080,
			matchPathPrefix: "/old",
			redirectionSite: site.NewRedirectionSite(false, "https://demo.local{uri}"),
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local:8080/old/page", nil)
	entry.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/old/page", recorder.Header().Get("Location"))
}

func TestNewEntryHTTPServerAppliesConnectionLimits(t *testing.T) {
	entry := newEntry(spec.SchemeHTTP, 8080, nil)

	server := newEntryHTTPServer("0.0.0.0:8080", entry)

	assert.Equal(t, entryReadHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, entryIdleTimeout, server.IdleTimeout)
	assert.Equal(t, entryMaxHeaderBytes, server.MaxHeaderBytes)
	assert.Equal(t, httputil.DefaultMaxHeaderValueCount, server.MaxHeaderValueCount)
	assert.Zero(t, server.ReadTimeout)
	assert.Zero(t, server.WriteTimeout)
}

func TestEntryResetRulesClearsAttachedRules(t *testing.T) {
	entry := &_Entry{
		rules: []*_Rule{{
			redirectionSite: &_TestSite{name: "admin@demo.app"},
		}},
	}

	entry.SetOrUpdateRouting(nil, nil)

	assert.Empty(t, entry.rules)
}

func newTestRequest(target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func TestEntryTargetPathForwardingAndUpdate(t *testing.T) {
	for _, inproc := range []bool{false, true} {
		t.Run(fmt.Sprintf("inproc=%t", inproc), func(t *testing.T) {
			backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("X-Received-Authorization", r.Header.Get("Authorization"))
				w.Header().Set("X-Received-Method", r.Method)
				w.Header().Set("X-Received-Body", string(body))
				_, _ = io.WriteString(w, r.URL.RequestURI())
			})
			var endpoint string
			if inproc {
				endpoint = "link+inproc://vine/entry-target-path-test"
				ingressinproc.Register(endpoint, backend)
				t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
			} else {
				server := httptest.NewUnstartedServer(backend)
				server.Config.Protocols = httputil.UnencryptedHTTP2Protocols()
				server.Start()
				t.Cleanup(server.Close)
				endpoint = server.URL
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			values := map[string]string{
				watched.FormatDescriptorWebKey("demo.Web"):     vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.Web", AuthMode: skeldesc.AuthModeOff}),
				watched.FormatDescriptorActorKey("demo.Actor"): vcode.MustMarshalJsonS(watched.DescriptorActor{SkelName: "demo.Actor"}),
				watched.FormatPortalSiteKey("web"): vcode.MustMarshalJsonS(watched.PortalSite{
					Name: "web", Type: "WEBGW", ActorSkelName: "demo.Actor", WebgwConfig: &watched.PortalWebgwConfig{WebName: "demo.Web"},
				}),
				watched.FormatWebRegistrationKey("demo.Web", "demo", "instance"): vcode.MustMarshalJsonS(watched.WebRegistration{
					Endpoint: endpoint + "/web/proxy/in/instance/demo.Web", WebSkelName: "demo.Web", AppName: "demo", AppInstanceId: "instance",
				}),
			}
			endpoints := &epmgr.Manager{Context: ctx, Watch: watchtest.New(t, values)}
			endpoints.DIInit()
			descriptors := new(desc.Manager{Context: ctx, Watch: watchtest.New(t, values)})
			descriptors.DIInit()
			t.Cleanup(descriptors.AfterAppStop)
			accessManager := &access.Access{Context: ctx, Desc: descriptors, Epmgr: endpoints}
			accessManager.DIInit()
			t.Cleanup(accessManager.AfterAppStop)
			sites := &site.Manager{Context: ctx, CurrentApp: meta.MustNewApp("vine.portal", "1.2.3", "123e4567-e89b-12d3-a456-426614174099"), Watch: watchtest.New(t, values), Epmgr: endpoints, Access: accessManager}
			sites.DIInit()
			entry := newEntry(spec.SchemeHTTP, 80, nil)
			public := httptest.NewServer(entry)
			t.Cleanup(public.Close)
			for _, hostPattern := range []string{"shop.example.com", "*.example.com"} {
				for _, targetPath := range []string{"/internal", "/v2", ""} {
					rule, ok := newRule(watched.PortalRule{Name: "rule", ResolvedMatchPathPrefix: "/api", ResolvedRoutePathPrefix: targetPath, RouteType: "SITE", RouteSiteName: "web"}, watched.PortalEntry{EffectiveHTTP: watched.PortalEntryHTTP{Host: hostPattern}}, spec.SchemeHTTP, 80, sites)
					require.True(t, ok)
					entry.SetOrUpdateRouting([]string{hostPattern}, []*_Rule{rule})
					request, err := http.NewRequest(http.MethodPost, public.URL+"/api/a%2Fb/?q=%2F", strings.NewReader("payload"))
					require.NoError(t, err)
					request.Header.Set("Authorization", "Bearer native-token")
					request.Host = "shop.example.com"
					response, err := public.Client().Do(request)
					require.NoError(t, err)
					body, err := io.ReadAll(response.Body)
					_ = response.Body.Close()
					require.NoError(t, err)
					require.Equal(t, http.StatusOK, response.StatusCode, string(body))
					assert.Equal(t, "/web/proxy/in/instance/demo.Web"+targetPath+"/a%2Fb/?q=%2F", string(body))
					assert.Equal(t, "Bearer native-token", response.Header.Get("X-Received-Authorization"))
					assert.Equal(t, "POST", response.Header.Get("X-Received-Method"))
					assert.Equal(t, "payload", response.Header.Get("X-Received-Body"))
				}
			}
		})
	}
}

func TestEntryTargetPathDispatchesWithinRpcGateway(t *testing.T) {
	manager := newTestSiteManager(t, "rpc")
	gateway, found := manager.Site("rpc")
	require.True(t, found)
	gateway.Update(watched.PortalSite{Name: "rpc", Type: "RPCGW", RpcgwConfig: new(watched.PortalRpcgwConfig{InspectEnabled: true})})
	rule, ok := newRule(watched.PortalRule{Name: "rpc", ResolvedMatchPathPrefix: "/api", ResolvedRoutePathPrefix: "/inspect", RouteType: "SITE", RouteSiteName: "rpc"}, watched.PortalEntry{EffectiveHTTP: watched.PortalEntryHTTP{Host: "example.com"}}, spec.SchemeHTTP, 80, manager)
	require.True(t, ok)
	entry := newEntry(spec.SchemeHTTP, 80, nil)
	entry.SetOrUpdateRouting([]string{"example.com"}, []*_Rule{rule})
	response := httptest.NewRecorder()
	entry.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api", nil))
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"services":[]}`, response.Body.String())
}

func TestEntryDispatchesMcpAtApiPrefix(t *testing.T) {
	manager := newTestSiteManager(t, "rpc")
	gateway, found := manager.Site("rpc")
	require.True(t, found)
	gateway.Update(watched.PortalSite{Name: "rpc", Type: "RPCGW", RpcgwConfig: new(watched.PortalRpcgwConfig{McpEnabled: true})})
	rule, ok := newRule(watched.PortalRule{Name: "rpc", ResolvedMatchPathPrefix: "/api", ResolvedRoutePathPrefix: "/", RouteType: "SITE", RouteSiteName: "rpc"}, watched.PortalEntry{EffectiveHTTP: watched.PortalEntryHTTP{Host: "example.com"}}, spec.SchemeHTTP, 80, manager)
	require.True(t, ok)
	entry := newEntry(spec.SchemeHTTP, 80, nil)
	entry.SetOrUpdateRouting([]string{"example.com"}, []*_Rule{rule})
	request := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "tools/list")
	response := httptest.NewRecorder()
	entry.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"tools":[]`)
}

func TestEntryWildcardHostPrecedence(t *testing.T) {
	e := newEntry(spec.SchemeHTTP, 80, nil)
	e.SetOrUpdateRouting([]string{"", "*.example.com", "a.example.com"}, []*_Rule{
		{name: "fallback", matchScheme: spec.SchemeHTTP, matchPathPrefix: "/deep/path"},
		{name: "wildcard", matchHost: "*.example.com", matchPathPrefix: "/deep"},
		{name: "exact", matchHost: "a.example.com", matchPathPrefix: "/"},
		{name: "wildcard-long", matchHost: "*.example.com", matchPathPrefix: "/deep/path"},
	})
	for host, want := range map[string]string{"a.example.com": "exact", "b.example.com": "wildcard-long", "127.0.0.1": "fallback"} {
		rule, ok := e.route(httptest.NewRequest("GET", "http://"+host+"/deep/path", nil))
		require.True(t, ok)
		assert.Equal(t, want, rule.name)
	}
}

func TestEntryHTTPSelectsHostBeforeMatchingPaths(t *testing.T) {
	e := newEntry(spec.SchemeHTTP, 80, nil)
	e.SetOrUpdateRouting([]string{"", "*.example.com", "app.example.com", "127.0.0.1", "::1"}, []*_Rule{
		{name: "fallback", matchScheme: spec.SchemeHTTP, matchPathPrefix: "/"},
		{name: "wildcard", matchHost: "*.example.com", matchPathPrefix: "/"},
		{name: "exact", matchHost: "app.example.com", matchPathPrefix: "/own"},
		{name: "ip", matchHost: "127.0.0.1", matchPathPrefix: "/own"},
		{name: "ipv6", matchHost: "::1", matchPathPrefix: "/own"},
	})
	for _, test := range []struct {
		target string
		rule   string
	}{
		{"http://app.example.com/own", "exact"},
		{"http://APP.EXAMPLE.COM./own", "exact"},
		{"http://app.example.com/other", ""},
		{"http://other.example.com/other", "wildcard"},
		{"http://deep.other.example.com/other", ""},
		{"http://127.0.0.1/own", "ip"},
		{"http://127.0.0.1/other", ""},
		{"http://127.0.0.2/other", "fallback"},
		{"http://[::1]/own", "ipv6"},
		{"http://[0:0:0:0:0:0:0:1]/own", "ipv6"},
		{"http://[::1]/other", ""},
		{"http://[::2]/other", "fallback"},
		{"http://unknown.test/other", ""},
	} {
		t.Run(test.target, func(t *testing.T) {
			rule, ok := e.route(newTestRequest(test.target))
			require.Equal(t, test.rule != "", ok)
			if ok {
				require.Equal(t, test.rule, rule.name)
			}
		})
	}
}

func TestEntryExplicitIPListeners(t *testing.T) {
	for _, test := range []struct {
		ip      string
		network string
	}{{"127.0.0.1", "tcp4"}, {"::1", "tcp6"}} {
		t.Run(test.ip, func(t *testing.T) {
			if test.network == "tcp6" {
				probe, err := net.Listen("tcp6", "[::1]:0")
				if err != nil {
					t.Skipf("IPv6 unavailable: %v", err)
				}
				_ = probe.Close()
			}
			entry := newEntry(spec.SchemeHTTP, 0, nil)
			entry.listenIP = test.ip
			require.NoError(t, entry.Start())
			t.Cleanup(entry.Stop)
			require.NoError(t, entry.Start())
			host, _, err := net.SplitHostPort(entry.addr)
			require.NoError(t, err)
			assert.Equal(t, test.ip, host)
			connection, err := net.DialTimeout(test.network, entry.addr, time.Second)
			require.NoError(t, err)
			defer connection.Close()
			require.NoError(t, connection.SetDeadline(time.Now().Add(time.Second)))
			request, err := http.NewRequest(http.MethodGet, "http://example.com/missing", nil)
			require.NoError(t, err)
			require.NoError(t, request.Write(connection))
			response, err := http.ReadResponse(bufio.NewReader(connection), request)
			require.NoError(t, err)
			_ = response.Body.Close()
			assert.Equal(t, http.StatusNotFound, response.StatusCode)
		})
	}
}

func TestEntryDefaultKeepsTCPWildcard(t *testing.T) {
	previous := listenEntryTCP
	t.Cleanup(func() { listenEntryTCP = previous })
	listenEntryTCP = func(network string, address string) (net.Listener, error) {
		assert.Equal(t, "tcp", network)
		assert.Equal(t, "0.0.0.0:8080", address)
		return newTestListener(), nil
	}
	entry := newEntry(spec.SchemeHTTP, 8080, nil)
	require.NoError(t, entry.Start())
	t.Cleanup(entry.Stop)
}

func TestEntryTLSSelectsEntryBeforeCertificate(t *testing.T) {
	cert := testHTTPPortalCertificate(t, "*.example.com", "*.public.example.com")
	certificate, err := tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
	require.NoError(t, err)
	roots := x509.NewCertPool()
	roots.AddCert(certificate.Leaf)
	certificateVault := &vault.Vault{Context: context.Background(), Watch: watchtest.New(t, map[string]string{watched.FormatPortalCertKey("test"): vcode.MustMarshalJsonS(cert)})}
	certificateVault.DIInit()
	configs := testEntryConfigs(
		testEntryConfig("exact", "https", "allowed.example.com", 0, []string{"127.0.0.1"}),
		testEntryConfig("wildcard", "https", "*.public.example.com", 0, []string{"127.0.0.1"}),
		testEntryConfig("private", "https", "private.public.example.com", 0, []string{"127.0.0.1"}),
		testEntryConfig("other-port", "https", "other.example.com", 443, []string{"127.0.0.1"}),
		testEntryConfig("other-ip", "https", "ip.example.com", 0, []string{"127.0.0.2"}),
	)
	manager := &Manager{Vault: certificateVault, SiteManager: &site.Manager{}, entryConfigsByName: configs, entryRulesByName: map[string]watched.PortalRule{"wildcard": {Name: "wildcard", EntryName: "wildcard", RouteType: routeTypeSite, RouteSiteName: "unavailable"}}, entriesByKey: map[_Key]*_Entry{}}
	require.NoError(t, manager.reconcileEntriesLocked())
	ingress := manager.entriesByKey[_Key{listenIP: "127.0.0.1", scheme: spec.SchemeHTTPS, port: 0}]
	require.NoError(t, ingress.Start())
	t.Cleanup(ingress.Stop)
	request := func(sni string, host string, clientConfig *tls.Config) (int, bool, error) {
		config := clientConfig.Clone()
		config.ServerName = sni
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp4", ingress.addr, config)
		if err != nil {
			return 0, false, err
		}
		defer conn.Close()
		require.NoError(t, conn.SetDeadline(time.Now().Add(3*time.Second)))
		_, err = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host)
		if err != nil {
			return 0, false, err
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			return 0, false, err
		}
		defer response.Body.Close()
		_, err = io.Copy(io.Discard, response.Body)
		return response.StatusCode, conn.ConnectionState().DidResume, err
	}
	trusted := &tls.Config{RootCAs: roots}
	for _, test := range []struct {
		sni    string
		host   string
		status int
	}{
		{"allowed.example.com", "allowed.example.com", http.StatusNotFound},
		{"allowed.example.com", "ALLOWED.EXAMPLE.COM.:443", http.StatusNotFound},
		{"allowed.example.com", "a.public.example.com", http.StatusMisdirectedRequest},
		{"a.public.example.com", "b.public.example.com", http.StatusMisdirectedRequest},
		{"a.public.example.com", "a.public.example.com", http.StatusServiceUnavailable},
		{"private.public.example.com", "private.public.example.com", http.StatusNotFound},
	} {
		t.Run(test.sni+"/"+test.host, func(t *testing.T) {
			status, _, err := request(test.sni, test.host, trusted)
			require.NoError(t, err)
			require.Equal(t, test.status, status)
		})
	}
	// Skip client verification here so only server entry admission can reject these.
	for _, sni := range []string{"unknown.example.com", "other.example.com", "ip.example.com", "deep.a.public.example.com", "", "127.0.0.1"} {
		t.Run("reject/"+sni, func(t *testing.T) {
			_, _, err := request(sni, "allowed.example.com", &tls.Config{InsecureSkipVerify: true})
			require.Error(t, err)
		})
	}
	// Admission must also run when a previously accepted TLS session is resumed.
	for _, version := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
		t.Run(fmt.Sprint("resume/", version), func(t *testing.T) {
			config := &tls.Config{RootCAs: roots, MinVersion: version, MaxVersion: version, ClientSessionCache: tls.NewLRUClientSessionCache(4)}
			_, _, err := request("allowed.example.com", "allowed.example.com", config)
			require.NoError(t, err)
			_, resumed, err := request("allowed.example.com", "allowed.example.com", config)
			require.NoError(t, err)
			require.True(t, resumed)
			key := watched.FormatPortalEntryKey("exact")
			manager.handlePortalEntryEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: key})
			_, _, err = request("allowed.example.com", "allowed.example.com", config)
			require.Error(t, err)
			value := testEntryConfig("exact", "https", "allowed.example.com", 0, []string{"127.0.0.1"})
			manager.handlePortalEntryEvent(hubapiwatch.Event{Key: key, Value: vcode.MustMarshalJsonS(value)})
		})
	}
}

func TestEntryRejectsMissingHost(t *testing.T) {
	ingress := newEntry(spec.SchemeHTTP, 80, nil)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	request.Host = ""
	response := httptest.NewRecorder()
	ingress.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}
