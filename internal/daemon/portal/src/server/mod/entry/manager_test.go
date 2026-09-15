package entry

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/meta"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerReconcileEntriesBindsPortAndRules(t *testing.T) {
	manager := &Manager{
		entryRulesByName: map[string]watched.PortalRule{},
		entriesByKey:     map[_Key]*_Entry{},
		SiteManager:      newTestSiteManager("admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:            "admin",
		MatchScheme:     string(spec.SchemeHTTPS),
		MatchHost:       "demo.local",
		MatchPort:       8443,
		MatchPathPrefix: "/admin",
		RouteType:       "SITE",
		RouteSiteName:   "admin@demo.app",
	}
	manager.entryRulesByName["home"] = watched.PortalRule{
		Name:            "home",
		MatchScheme:     string(spec.SchemeHTTPS),
		MatchHost:       "demo.local",
		MatchPort:       8443,
		MatchPathPrefix: "/",
		RouteType:       "SITE",
		RouteSiteName:   "home@demo.app",
	}
	manager.entryRulesByName["redirect"] = watched.PortalRule{
		Name:                    "redirect",
		MatchScheme:             string(spec.SchemeHTTP),
		MatchHost:               "demo.local",
		MatchPort:               8080,
		MatchPathPrefix:         "/old",
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
		entryRulesByName: map[string]watched.PortalRule{},
		entriesByKey:     map[_Key]*_Entry{},
		SiteManager:      newTestSiteManager("admin@demo.app", "home@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:          "admin",
		MatchScheme:   string(spec.SchemeHTTPS),
		MatchPort:     8443,
		RouteType:     "SITE",
		RouteSiteName: "admin@demo.app",
	}
	manager.entryRulesByName["home"] = watched.PortalRule{
		Name:          "home",
		MatchScheme:   string(spec.SchemeHTTP),
		MatchPort:     8443,
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
		entryRulesByName: map[string]watched.PortalRule{},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTPS, port: 8443}: existing,
		},
		SiteManager: newTestSiteManager("admin@demo.app"),
	}

	manager.entryRulesByName["admin"] = watched.PortalRule{
		Name:          "admin",
		MatchScheme:   string(spec.SchemeHTTPS),
		MatchPort:     8443,
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
		entryRulesByName: map[string]watched.PortalRule{
			"admin": {
				Name:          "admin",
				MatchScheme:   string(spec.SchemeHTTP),
				MatchPort:     8080,
				RouteType:     "SITE",
				RouteSiteName: "admin@demo.app",
			},
		},
		entriesByKey: map[_Key]*_Entry{
			{scheme: spec.SchemeHTTP, port: 8080}: existing,
		},
		SiteManager: newTestSiteManager("admin@demo.app"),
	}

	manager.AfterAppStart()

	assert.True(t, existing.started)
	assert.Equal(t, "0.0.0.0:8080", listenAddress)
}

func newTestSiteManager(names ...string) *site.Manager {
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
		Watch:   hubwatch.NewTestClient(valuesByKey),
	}
	epmgrManager.DIInit()
	manager := &site.Manager{
		App:     meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099"),
		Context: context.Background(),
		Watch:   hubwatch.NewTestClient(valuesByKey),
		Epmgr:   epmgrManager,
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

func TestManagerWebMountPathsFollowSiteEvents(t *testing.T) {
	siteKey := watched.FormatPortalSiteKey("web")
	ruleKey := watched.FormatPortalRuleKey("web-rule")
	siteValue := func(path string) string {
		return vcode.MustMarshalJsonS(watched.PortalSite{Name: "web", Type: "WEBGW",
			WebgwConfig: &watched.PortalWebgwConfig{WebName: "demo.Web", MountPath: path}})
	}
	rule := watched.PortalRule{Name: "web-rule", MatchScheme: "http", MatchPort: 8080,
		RouteType: "SITE", RouteSiteName: "web", MatchPathPrefix: "/configured", RoutePathPrefix: "/backend"}
	manager := &Manager{Context: t.Context(), SiteManager: new(site.Manager), Watch: hubwatch.NewTestClient(map[string]string{
		siteKey: siteValue("/initial/"), ruleKey: vcode.MustMarshalJsonS(rule),
	})}
	manager.DIInit()
	t.Cleanup(manager.AfterAppStop)
	entry := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
	checkRoute := func(path string, rewritten string) {
		t.Helper()
		request := httptest.NewRequest("GET", "http://example.com"+path, nil)
		matched, ok := entry.route(request)
		require.True(t, ok)
		require.Equal(t, rewritten, matched.rewritePath(request).URL.RequestURI())
	}
	checkRoute("/initial/a%2Fb?q=1", "/initial/a%2Fb?q=1")
	checkRoute("/initial", "/initial")
	_, ok := entry.route(httptest.NewRequest("GET", "http://example.com/initially", nil))
	require.False(t, ok)
	_, ok = entry.route(httptest.NewRequest("GET", "http://example.com/configured", nil))
	require.False(t, ok)

	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindUpsert, Key: siteKey, Value: siteValue("/next")})
	checkRoute("/next/a", "/next/a")
	_, ok = entry.route(httptest.NewRequest("GET", "http://example.com/initial/a", nil))
	require.False(t, ok)

	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindUpsert, Key: siteKey, Value: siteValue("/")})
	checkRoute("/any/a%2Fb?q=1", "/any/a%2Fb?q=1")
	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindUpsert, Key: siteKey, Value: siteValue("")})
	checkRoute("/configured/a%2Fb?q=1", "/backend/a%2Fb?q=1")
	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindUpsert, Key: siteKey, Value: siteValue("/again")})
	checkRoute("/again/a", "/again/a")
	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: siteKey})
	checkRoute("/configured/a", "/backend/a")
	require.Equal(t, rule, manager.entryRulesByName[ruleKey], "site changes must not rewrite stored rule configuration")
}

func TestManagerMountChangesReorderRoutesOnLiveListener(t *testing.T) {
	originalListen := listenEntryTCP
	listenEntryTCP = func(network string, address string) (net.Listener, error) {
		return net.Listen(network, "127.0.0.1:0")
	}
	t.Cleanup(func() { listenEntryTCP = originalListen })
	siteKey := watched.FormatPortalSiteKey("web")
	siteValue := func(path string) string {
		return vcode.MustMarshalJsonS(watched.PortalSite{Name: "web", Type: "WEBGW",
			WebgwConfig: &watched.PortalWebgwConfig{MountPath: path}})
	}
	manager := &Manager{Context: t.Context(), SiteManager: new(site.Manager), Watch: hubwatch.NewTestClient(map[string]string{
		siteKey: siteValue("/app"),
		watched.FormatPortalRuleKey("web"): vcode.MustMarshalJsonS(watched.PortalRule{
			Name: "web", MatchScheme: "http", MatchPort: 8080, RouteType: "SITE", RouteSiteName: "web"}),
		watched.FormatPortalRuleKey("fallback"): vcode.MustMarshalJsonS(watched.PortalRule{
			Name: "fallback", MatchScheme: "http", MatchPort: 8080, MatchPathPrefix: "/", RouteType: "TEMPORARY_REDIRECT", RouteRedirectionPattern: "https://example.com"}),
	})}
	manager.DIInit()
	manager.AfterAppStart()
	t.Cleanup(manager.AfterAppStop)
	entry := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	checkStatus := func(path string, status int) {
		t.Helper()
		response, err := client.Get("http://" + entry.addr + path)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, status, response.StatusCode)
	}
	// No gateway is registered: 503 identifies the SITE rule, 307 the fallback.
	checkStatus("/app/x", http.StatusServiceUnavailable)
	checkStatus("/other", http.StatusTemporaryRedirect)
	manager.handlePortalSiteEvent(hubapiwatch.Event{Kind: hubapiwatch.EventKindUpsert, Key: siteKey, Value: siteValue("/new")})
	checkStatus("/new/x", http.StatusServiceUnavailable)
	checkStatus("/app/x", http.StatusTemporaryRedirect)
}
