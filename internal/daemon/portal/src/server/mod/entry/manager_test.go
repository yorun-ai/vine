package entry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/meta"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerReconcileEntriesBindsPortAndRules(t *testing.T) {
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "https", "demo.local", 8443, nil), testEntryConfig("home", "https", "demo.local", 8443, nil), testEntryConfig("redirect", "http", "demo.local", 8080, nil)),
		entryRulesByName:   map[string]watched.PortalRule{},
		entriesByKey:       map[_Key]*_Entry{},
		SiteManager:        newTestSiteManager(t, "admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:      "admin",
		EntryName: "admin",

		ResolvedMatchPathPrefix: "/admin",
		RouteType:               "SITE",
		RouteSiteName:           "admin@demo.app",
	}
	manager.entryRulesByName["home"] = watched.PortalRule{
		Name:      "home",
		EntryName: "home",

		ResolvedMatchPathPrefix: "/",
		RouteType:               "SITE",
		RouteSiteName:           "home@demo.app",
	}
	manager.entryRulesByName["redirect"] = watched.PortalRule{
		Name:      "redirect",
		EntryName: "redirect",

		ResolvedMatchPathPrefix: "/old",
		RouteType:               "PERMANENT_REDIRECT",
		RouteRedirectionPattern: "https://demo.local/new",
	}

	manager.reconcileEntriesLocked()

	httpsKey := _Key{scheme: spec.SchemeHTTPS, port: 8443}
	httpKey := _Key{scheme: spec.SchemeHTTP, port: 8080}
	assert.Len(t, manager.entriesByKey, 2)
	assert.Len(t, manager.entriesByKey[httpsKey].rules, 2)
	assert.Equal(t, "admin@demo.app", manager.entriesByKey[httpsKey].rules[0].routeSiteName)
	assert.Len(t, manager.entriesByKey[httpKey].rules, 1)
	assert.Equal(t, "redirection", manager.entriesByKey[httpKey].rules[0].redirectionSite.Name())
}

func TestManagerHTTPEntryDoesNotInheritWildcardRedirect(t *testing.T) {
	certificateVault := &vault.Vault{Context: context.Background(), Watch: watchtest.New(t, map[string]string{
		watched.FormatPortalCertKey("test"): vcode.MustMarshalJsonS(testHTTPPortalCertificate(t, "*.example.com")),
	})}
	certificateVault.DIInit()
	exact := testEntryConfig("exact", "http", "app.example.com", 8080, nil)
	wildcard := watched.PortalEntry{Name: "wildcard", Protocol: "http", EffectiveHTTP: watched.PortalEntryHTTP{
		Host: "*.example.com", HttpEnabled: true, HttpPort: 8080,
		HttpsEnabled: true, HttpsPort: 8443, AutoHTTPSEnabled: true,
	}}
	manager := &Manager{
		Vault:              certificateVault,
		entryConfigsByName: testEntryConfigs(exact, wildcard),
		entryRulesByName:   map[string]watched.PortalRule{},
		entriesByKey:       map[_Key]*_Entry{},
	}
	require.NoError(t, manager.reconcileEntriesLocked())
	ingress := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
	request := func(target string, status int, location string) {
		t.Helper()
		response := httptest.NewRecorder()
		ingress.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		require.Equal(t, status, response.Code)
		require.Equal(t, location, response.Header().Get("Location"))
	}
	request("http://app.example.com:8080/path", http.StatusNotFound, "")
	request("http://other.example.com:8080/path", http.StatusPermanentRedirect, "https://other.example.com:8443/path")

	rule := watched.PortalRule{Name: "exact-rule", EntryName: "exact", ResolvedMatchPathPrefix: "/own", RouteType: routeTypeTemporaryRedirect, RouteRedirectionPattern: "http://target.test"}
	manager.handlePortalRuleEvent(hubapiwatch.Event{Key: watched.FormatPortalRuleKey(rule.Name), Value: vcode.MustMarshalJsonS(rule)})
	request("http://app.example.com:8080/own", http.StatusTemporaryRedirect, "http://target.test")
	request("http://app.example.com:8080/other", http.StatusNotFound, "")

	// Removing the exact entry lets the wildcard own this host again.
	manager.handlePortalEntryEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: watched.FormatPortalEntryKey(exact.Name)})
	request("http://app.example.com:8080/path", http.StatusPermanentRedirect, "https://app.example.com:8443/path")
	manager.handlePortalEntryEvent(hubapiwatch.Event{Key: watched.FormatPortalEntryKey(exact.Name), Value: vcode.MustMarshalJsonS(exact)})
	request("http://app.example.com:8080/path", http.StatusNotFound, "")
}

func TestManagerReconcileEntriesDeduplicatesBySchemeAndPort(t *testing.T) {
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "https", "demo.local", 8443, nil), testEntryConfig("home", "http", "", 8443, nil)),
		entryRulesByName:   map[string]watched.PortalRule{},
		entriesByKey:       map[_Key]*_Entry{},
		SiteManager:        newTestSiteManager(t, "admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:      "admin",
		EntryName: "admin",

		RouteType:     "SITE",
		RouteSiteName: "admin@demo.app",
	}
	manager.entryRulesByName["home"] = watched.PortalRule{
		Name:      "home",
		EntryName: "home",

		RouteType:     "SITE",
		RouteSiteName: "home@demo.app",
	}

	manager.reconcileEntriesLocked()

	httpsKey := _Key{scheme: spec.SchemeHTTPS, port: 8443}
	httpKey := _Key{scheme: spec.SchemeHTTP, port: 8443}
	assert.Len(t, manager.entriesByKey, 2)
	assert.Len(t, manager.entriesByKey[httpsKey].rules, 1)
	assert.Len(t, manager.entriesByKey[httpKey].rules, 1)
}

func TestManagerReconcileEntriesUpdatesExistingPortalRules(t *testing.T) {
	existing := newEntry(spec.SchemeHTTPS, 8443, nil)
	existing.SetOrUpdateRouting(nil, []*_Rule{{name: "old"}})
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "https", "demo.local", 8443, nil)),
		entryRulesByName:   map[string]watched.PortalRule{},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTPS, port: 8443}: existing,
		},
		SiteManager: newTestSiteManager(t, "admin@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:      "admin",
		EntryName: "admin",

		RouteType:     "SITE",
		RouteSiteName: "admin@demo.app",
	}

	manager.reconcileEntriesLocked()

	assert.Same(t, existing, manager.entriesByKey[_Key{scheme: spec.SchemeHTTPS, port: 8443}])
	assert.Len(t, existing.rules, 1)
	assert.Equal(t, "admin", existing.rules[0].name)
}

func TestManagerAfterAppStartStartsEntriesCreatedBeforeStart(t *testing.T) {
	prev := listenEntryTCP
	var listenAddress string
	listenEntryTCP = func(network string, address string) (net.Listener, error) {
		listenAddress = address
		return newTestListener(), nil
	}
	t.Cleanup(func() {
		listenEntryTCP = prev
	})

	existing := newEntry(spec.SchemeHTTP, 8080, nil)
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "http", "", 8080, nil)),
		entryRulesByName: map[string]watched.PortalRule{
			"admin": {
				Name:      "admin",
				EntryName: "admin",

				RouteType:     "SITE",
				RouteSiteName: "admin@demo.app",
			},
		},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTP, port: 8080}: existing,
		},
		SiteManager: newTestSiteManager(t, "admin@demo.app"),
	}

	manager.AfterAppStart()

	assert.True(t, existing.started)
	assert.Equal(t, "0.0.0.0:8080", listenAddress)
}

func newTestSiteManager(t *testing.T, names ...string) *site.Manager {
	valuesByKey := map[string]string{}
	for _, name := range names {
		valuesByKey[watched.FormatPortalSiteKey(name)] = vcode.MustMarshalJsonS(watched.PortalSite{
			Name: name,
			Type: "RPCGW",
			RpcgwConfig: &watched.PortalRpcgwConfig{
				Services: []watched.PortalRpcgwService{{SkelName: "demo.UserService"}},
			},
		})
	}
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Watch:   watchtest.New(t, valuesByKey),
	}
	epmgrManager.DIInit()
	descriptors := new(desc.Manager{Context: t.Context(), Watch: watchtest.New(t, nil)})
	descriptors.DIInit()
	t.Cleanup(descriptors.AfterAppStop)
	manager := &site.Manager{
		CurrentApp: meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099"),
		Context:    context.Background(),
		Watch:      watchtest.New(t, valuesByKey),
		Epmgr:      epmgrManager,
		Desc:       descriptors,
	}
	manager.DIInit()
	return manager
}

type _TestListener struct {
}

func newTestListener() *_TestListener {
	return &_TestListener{}
}

func (l *_TestListener) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

func (l *_TestListener) Close() error {
	return nil
}

func (*_TestListener) Addr() net.Addr {
	return &net.TCPAddr{Port: 8080}
}

func testEntryConfig(name string, scheme string, host string, port int, ips []string) watched.PortalEntry {
	config := watched.PortalEntry{Name: name, Protocol: "http", EffectiveHTTP: watched.PortalEntryHTTP{Host: host}, ListenIPs: ips}
	if scheme == "http" {
		config.EffectiveHTTP.HttpEnabled = true
		config.EffectiveHTTP.HttpPort = port
	} else {
		config.EffectiveHTTP.HttpsEnabled = true
		config.EffectiveHTTP.HttpsPort = port
	}
	return config
}
func testEntryConfigs(configs ...watched.PortalEntry) map[string]watched.PortalEntry {
	result := map[string]watched.PortalEntry{}
	for _, config := range configs {
		result[watched.FormatPortalEntryKey(config.Name)] = config
	}
	return result
}
func testRedirectRule(name string, entryName string) watched.PortalRule {
	return watched.PortalRule{Name: name, EntryName: entryName, RouteType: "PERMANENT_REDIRECT", RouteRedirectionPattern: "https://example.com"}
}
func TestManagerListenIPsShareAndUpdateBindings(t *testing.T) {
	first := testEntryConfig("first", "http", "first.local", 8080, []string{"127.0.0.1", "::1"})
	second := testEntryConfig("second", "http", "second.local", 8080, []string{"127.0.0.1"})
	manager := &Manager{entryRulesByName: map[string]watched.PortalRule{"first": testRedirectRule("first", "first"), "second": testRedirectRule("second", "second")}, entryConfigsByName: testEntryConfigs(first, second), entriesByKey: map[_Key]*_Entry{}}
	require.NoError(t, manager.reconcileEntriesLocked())
	v4 := _Key{scheme: spec.SchemeHTTP, port: 8080, listenIP: "127.0.0.1"}
	v6 := _Key{scheme: spec.SchemeHTTP, port: 8080, listenIP: "::1"}
	require.Len(t, manager.entriesByKey, 2)
	require.Len(t, manager.entriesByKey[v4].rules, 2)
	require.Len(t, manager.entriesByKey[v6].rules, 1)
	shared := manager.entriesByKey[v4]
	first.ListenIPs = []string{"127.0.0.2"}
	manager.entryConfigsByName[watched.FormatPortalEntryKey(first.Name)] = first
	require.NoError(t, manager.reconcileEntriesLocked())
	require.Same(t, shared, manager.entriesByKey[v4])
	require.Len(t, shared.rules, 1)
	require.NotContains(t, manager.entriesByKey, v6)
	require.Contains(t, manager.entriesByKey, _Key{scheme: spec.SchemeHTTP, port: 8080, listenIP: "127.0.0.2"})
}
func TestManagerListenIPsBindingFailureClosesPartialListeners(t *testing.T) {
	previous := listenEntryTCP
	t.Cleanup(func() { listenEntryTCP = previous })
	var opened net.Listener
	calls := 0
	listenEntryTCP = func(network string, address string) (net.Listener, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("address unavailable")
		}
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		opened = listener
		return listener, err
	}
	config := testEntryConfig("local", "http", "", 8080, []string{"127.0.0.1", "::1"})
	manager := &Manager{started: true, entryRulesByName: map[string]watched.PortalRule{"local": testRedirectRule("local", "local")}, entryConfigsByName: testEntryConfigs(config), entriesByKey: map[_Key]*_Entry{}}
	t.Cleanup(manager.AfterAppStop)
	require.Error(t, manager.reconcileEntriesLocked())
	require.Empty(t, manager.entriesByKey)
	require.NotNil(t, opened)
	connection, err := net.DialTimeout("tcp4", opened.Addr().String(), time.Second)
	if connection != nil {
		_ = connection.Close()
	}
	require.Error(t, err)
}
func TestManagerListenIPsFailedUpdateRestoresOldListener(t *testing.T) {
	previous := listenEntryTCP
	t.Cleanup(func() { listenEntryTCP = previous })
	listenEntryTCP = func(network string, address string) (net.Listener, error) {
		if address == "127.0.0.2:8080" {
			return nil, errors.New("address unavailable")
		}
		return net.Listen("tcp4", "127.0.0.1:0")
	}
	config := testEntryConfig("local", "http", "", 8080, []string{"127.0.0.1"})
	manager := &Manager{started: true, entriesByKey: map[_Key]*_Entry{}, entryRulesByName: map[string]watched.PortalRule{"local": testRedirectRule("local", "local")}, entryConfigsByName: testEntryConfigs(config)}
	t.Cleanup(manager.AfterAppStop)
	require.NoError(t, manager.reconcileEntriesLocked())
	key := _Key{scheme: spec.SchemeHTTP, port: 8080, listenIP: "127.0.0.1"}
	old := manager.entriesByKey[key]
	config.ListenIPs = []string{"127.0.0.2"}
	manager.entryConfigsByName[watched.FormatPortalEntryKey(config.Name)] = config
	require.Error(t, manager.reconcileEntriesLocked())
	require.Same(t, old, manager.entriesByKey[key])
	require.True(t, old.started)
	connection, err := net.DialTimeout("tcp4", old.addr, time.Second)
	require.NoError(t, err)
	_ = connection.Close()
}
func TestManagerWatchUpdatesConvergeFromWildcardToExplicitIPs(t *testing.T) {
	probe, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 unavailable: %v", err)
	}
	_ = probe.Close()
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	port := reservation.Addr().(*net.TCPAddr).Port
	require.NoError(t, reservation.Close())
	config := testEntryConfig("web", "http", "", port, nil)
	first, second := testRedirectRule("one", "web"), testRedirectRule("two", "web")
	first.ResolvedMatchPathPrefix = "/one"
	second.ResolvedMatchPathPrefix = "/two"
	manager := &Manager{entryRulesByName: map[string]watched.PortalRule{"one": first, "two": second}, entryConfigsByName: testEntryConfigs(config), entriesByKey: map[_Key]*_Entry{}}
	t.Cleanup(manager.AfterAppStop)
	manager.AfterAppStart()
	oldKey := _Key{scheme: spec.SchemeHTTP, port: port}
	old := manager.entriesByKey[oldKey]
	require.NotNil(t, old)
	config.ListenIPs = []string{"127.0.0.1", "::1"}
	manager.handlePortalEntryEvent(hubapiwatch.Event{Key: watched.FormatPortalEntryKey(config.Name), Value: vcode.MustMarshalJsonS(config)})
	require.False(t, old.started)
	require.Len(t, manager.entriesByKey, 2)
	for _, ip := range config.ListenIPs {
		entry := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: port, listenIP: ip}]
		require.NotNil(t, entry)
		require.True(t, entry.started)
		require.Len(t, entry.rules, 2)
		connection, err := net.DialTimeout("tcp", entry.addr, time.Second)
		require.NoError(t, err)
		_ = connection.Close()
	}
	manager.handlePortalRuleEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: "one"})
	manager.handlePortalRuleEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: "two"})
	require.Len(t, manager.entriesByKey, 2, "entries own listeners even without rules")
	manager.handlePortalEntryEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: watched.FormatPortalEntryKey(config.Name)})
	require.Empty(t, manager.entriesByKey)
}
func TestManagerIgnoresLegacyRulesWithoutEntryReference(t *testing.T) {
	legacy := vcode.MustUnmarshalJsonS[watched.PortalRule](`{"name":"legacy","matchScheme":"http","matchPort":8080,"routeType":"PERMANENT_REDIRECT","routeRedirectionPattern":"https://example.com"}`)
	manager := &Manager{entryRulesByName: map[string]watched.PortalRule{"legacy": legacy}, entryConfigsByName: map[string]watched.PortalEntry{}, entriesByKey: map[_Key]*_Entry{}}
	require.NoError(t, manager.reconcileEntriesLocked())
	require.Empty(t, manager.entriesByKey)
	manager.entryConfigsByName = testEntryConfigs(testEntryConfig("web", "http", "", 8080, nil))
	require.NoError(t, manager.reconcileEntriesLocked())
	require.Len(t, manager.entriesByKey, 1)
	require.Empty(t, manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}].rules)
}

func TestEntryFollowsEffectiveTransportUpdates(t *testing.T) {
	certificateVault := &vault.Vault{Context: context.Background(), Watch: watchtest.New(t, map[string]string{watched.FormatPortalCertKey("test"): vcode.MustMarshalJsonS(testHTTPPortalCertificate(t))})}
	certificateVault.DIInit()
	manager := &Manager{Vault: certificateVault, entryRulesByName: map[string]watched.PortalRule{}, entryConfigsByName: map[string]watched.PortalEntry{}, entriesByKey: map[_Key]*_Entry{}}
	config := watched.PortalEntry{Name: "web", Protocol: "http", EffectiveHTTP: watched.PortalEntryHTTP{Host: "example.com", HttpEnabled: true, HttpPort: 8080, HttpsEnabled: true, HttpsPort: 8443, AutoHTTPSEnabled: true}}
	for _, test := range []struct {
		name         string
		httpEnabled  bool
		httpsEnabled bool
		autoHTTPS    bool
		status       int
	}{
		{"both enabled", true, true, true, http.StatusPermanentRedirect},
		{"HTTPS disabled", true, false, false, http.StatusNotFound},
		{"HTTPS restored", true, true, true, http.StatusPermanentRedirect},
		{"HTTP disabled", false, true, false, 0},
		{"HTTP restored", true, true, true, http.StatusPermanentRedirect},
		{"redirect disabled", true, true, false, http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			config.EffectiveHTTP.HttpEnabled = test.httpEnabled
			config.EffectiveHTTP.HttpsEnabled = test.httpsEnabled
			config.EffectiveHTTP.AutoHTTPSEnabled = test.autoHTTPS
			manager.handlePortalEntryEvent(hubapiwatch.Event{Key: watched.FormatPortalEntryKey("web"), Value: vcode.MustMarshalJsonS(config)})
			ingress := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
			if !test.httpEnabled {
				require.Nil(t, ingress)
				return
			}
			require.NotNil(t, ingress)
			response := httptest.NewRecorder()
			ingress.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.com/a", nil))
			require.Equal(t, test.status, response.Code)
			if test.status == http.StatusPermanentRedirect {
				require.Equal(t, "https://example.com:8443/a", response.Header().Get("Location"))
			} else {
				require.Empty(t, response.Header().Get("Location"))
			}
		})
	}
}

func TestEntryEmptyHostEffectiveHTTPOnly(t *testing.T) {
	config := watched.PortalEntry{Name: "ip", Protocol: "http", EffectiveHTTP: watched.PortalEntryHTTP{HttpEnabled: true, HttpPort: 8080, HttpsEnabled: false, HttpsPort: 8080, AutoHTTPSEnabled: false}}
	manager := &Manager{entryConfigsByName: testEntryConfigs(config), entryRulesByName: map[string]watched.PortalRule{"ip": testRedirectRule("ip", "ip")}, entriesByKey: map[_Key]*_Entry{}}
	require.NoError(t, manager.reconcileEntriesLocked())
	require.Len(t, manager.entriesByKey, 1)
	ingress := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
	require.NotNil(t, ingress)
	require.Len(t, ingress.rules, 1)
	require.Zero(t, ingress.rules[0].autoHTTPSPort)
	for _, host := range []string{"127.0.0.1:8080", "[::1]:8080", "example.com:8080", "[::1", "::1]", "[[::1]]"} {
		request := newTestRequest("/")
		request.Host = host
		_, ok := ingress.route(request)
		require.Equal(t, host == "127.0.0.1:8080" || host == "[::1]:8080", ok)
	}
}
