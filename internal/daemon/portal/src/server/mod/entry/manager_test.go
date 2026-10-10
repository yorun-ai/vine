package entry

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/meta"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
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

func TestManagerReconcileEntriesDeduplicatesBySchemeAndPort(t *testing.T) {
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "https", "", 8443, nil), testEntryConfig("home", "http", "", 8443, nil)),
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
	existing.SetOrUpdateRules([]*_Rule{{name: "old"}})
	manager := &Manager{
		entryConfigsByName: testEntryConfigs(testEntryConfig("admin", "https", "", 8443, nil)),
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
	manager := &site.Manager{
		CurrentApp: meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099"),
		Context:    context.Background(),
		Watch:      watchtest.New(t, valuesByKey),
		Epmgr:      epmgrManager,
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
	config := watched.PortalEntry{Name: name, Protocol: "http", Host: host, ListenIPs: ips}
	if scheme == "http" {
		config.Http.HttpEnabled = true
		config.Http.HttpPort = port
	} else {
		config.Http.HttpsEnabled = true
		config.Http.HttpsPort = port
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
