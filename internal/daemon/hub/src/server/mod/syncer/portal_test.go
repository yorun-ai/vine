package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// Two rules that match one request have no defined order in Portal, so Hub
// publishes the rule whose name sorts first and leaves the other out.
func TestSyncerPublishesOneRulePerRequest(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	entry := &core.PortalEntry{Id: 1, Name: "web", Scheme: "http", Port: 8099, Enabled: true}
	target.SyncPortalEntry(entry)
	first := &core.PortalRule{Id: 1, Name: "demo.app", EntryId: entry.Id, MatchPathPrefix: "/", Enabled: true}
	second := &core.PortalRule{Id: 2, Name: "demo.app-shadow", EntryId: entry.Id, MatchPathPrefix: "/", Enabled: true}
	target.SyncPortalRule(second)
	target.SyncPortalRule(first)

	_, ok := watchServer.Get(watched.FormatPortalRuleKey("demo.app"))
	assert.True(t, ok, "Hub publishes the rule whose name sorts first")
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("demo.app-shadow"))
	assert.False(t, ok, "Hub leaves the rule the other one supersedes out")

	// The rule that leaves lets the remaining one through again.
	target.RemovePortalRule(first)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("demo.app-shadow"))
	assert.True(t, ok, "the remaining rule is published once it serves the request alone")
}

func TestSyncerPublishesOnlyEnabledConfiguration(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	entry := &core.PortalEntry{Id: 1, Name: "web", Scheme: "http", Port: 8099, Enabled: true}
	target.SyncPortalEntry(entry)
	site := &core.PortalSite{Id: 1, Name: "web-site", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web", Enabled: true}
	target.SyncPortalSite(site)
	rule := &core.PortalRule{
		Id: 1, Name: "web", EntryId: entry.Id, RouteType: core.PortalRuleRouteTypeSite,
		RouteSiteName: site.Name, MatchPathPrefix: "/", Enabled: true,
	}
	target.SyncPortalRule(rule)
	target.SyncPortalCert(&core.PortalCert{Id: 1, Name: "web-cert", Enabled: true})

	for _, key := range []string{
		watched.FormatPortalRuleKey("web"),
		watched.FormatPortalSiteKey("web-site"),
		watched.FormatPortalCertKey("web-cert"),
	} {
		_, ok := watchServer.Get(key)
		assert.True(t, ok, "enabled configuration is published: %s", key)
	}

	// Disabling a rule hides it, and enabling it again publishes it.
	disabled := *rule
	disabled.Enabled = false
	target.SyncPortalRule(&disabled)
	_, ok := watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.False(t, ok, "a disabled rule is not published")
	target.SyncPortalRule(rule)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.True(t, ok)

	// Disabling the entry hides the rules it routes.
	disabledEntry := *entry
	disabledEntry.Enabled = false
	target.SyncPortalEntry(&disabledEntry)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.False(t, ok, "the rules of a disabled entry are not published")
	target.SyncPortalEntry(entry)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.True(t, ok)

	// Disabling the site hides the site and the SITE rules that target it.
	disabledSite := *site
	disabledSite.Enabled = false
	target.SyncPortalSite(&disabledSite)
	_, ok = watchServer.Get(watched.FormatPortalSiteKey("web-site"))
	assert.False(t, ok, "a disabled site is not published")
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.False(t, ok, "a rule of a disabled site is not published")
	target.SyncPortalSite(site)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("web"))
	assert.True(t, ok)

	// Disabling a certificate removes it from the Portal TLS configuration.
	target.SyncPortalCert(&core.PortalCert{Id: 1, Name: "web-cert", Enabled: false})
	_, ok = watchServer.Get(watched.FormatPortalCertKey("web-cert"))
	assert.False(t, ok, "a disabled certificate is not published")
}

func TestPortalRuleTargetPathWatchRoundTrip(t *testing.T) {
	rule := &core.PortalRule{Name: "mapped", RouteType: "SITE", MatchPathPrefix: "/api", RoutePathPrefix: "/internal"}
	wire := vcode.MustMarshalJsonS(ToWatchedPortalRule(rule, nil))
	decoded := vcode.MustUnmarshalJsonS[*watched.PortalRule](wire)
	assert.Equal(t, "/internal", decoded.ResolvedRoutePathPrefix)
	assert.Contains(t, wire, `"resolvedRoutePathPrefix":"/internal"`)
	assert.Contains(t, wire, `"resolvedMatchPathPrefix":"/api"`)
	assert.NotContains(t, wire, `"routePathPrefix"`)
	assert.NotContains(t, wire, `"matchPathPrefix"`)
	assert.NotContains(t, wire, `"targetPath"`)
	assert.NotContains(t, wire, `"targetType"`)
}

func TestPortalSitePublishesWebMountPath(t *testing.T) {
	for _, mountPath := range []string{"", "/", "/app"} {
		site := &core.PortalSite{Name: "web", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web", WebMountPath: mountPath}
		wire := vcode.MustMarshalJsonS(toWatchedPortalSite(site))
		decoded := vcode.MustUnmarshalJsonS[watched.PortalSite](wire)
		assert.Equal(t, mountPath, decoded.WebgwConfig.MountPath)
	}
}

func TestPortalRuleResolvedPrefixesFollowWebMountPath(t *testing.T) {
	target := testSyncer(watchserver.NewServerForTest())
	tests := []struct {
		name      string
		mountPath string
		match     string
		route     string
	}{
		{name: "empty", mountPath: "", match: "/configured", route: "/backend"},
		{name: "root", mountPath: "/", match: "/", route: ""},
		{name: "trim trailing slash", mountPath: "/app/", match: "/app", route: "/app"},
		{name: "nested", mountPath: "/app/admin", match: "/app/admin", route: "/app/admin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &core.PortalRule{RouteType: string(core.PortalRuleRouteTypeSite), RouteSiteName: "web", MatchPathPrefix: "/configured", RoutePathPrefix: "/backend"}
			site := &core.PortalSite{Type: core.PortalSiteTypeWEBGW, Name: "web", WebMountPath: tt.mountPath}
			got := target.toWatchedPortalRule(rule, site)
			assert.Equal(t, tt.match, got.ResolvedMatchPathPrefix)
			assert.Equal(t, tt.route, got.ResolvedRoutePathPrefix)
		})
	}
	target.WatchServer.AfterAppStop()
}

func TestPortalSiteUpdateRepublishesResolvedRule(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)
	rule := &core.PortalRule{
		Id:              1,
		Name:            "web-rule",
		RouteType:       string(core.PortalRuleRouteTypeSite),
		RouteSiteName:   "web",
		MatchPathPrefix: "/configured",
		RoutePathPrefix: "/backend",
		Enabled:         true,
	}
	target.SyncPortalRule(rule)
	target.SyncPortalSite(&core.PortalSite{Id: 1, Name: "web", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/old", Enabled: true})
	value, ok := watchServer.Get(watched.FormatPortalRuleKey(rule.Name))
	assert.True(t, ok)
	assert.Equal(t, "/old", vcode.MustUnmarshalJsonS[*watched.PortalRule](value).ResolvedMatchPathPrefix)

	target.SyncPortalSite(&core.PortalSite{Id: 1, Name: "web", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/new/", Enabled: true})
	value, ok = watchServer.Get(watched.FormatPortalRuleKey(rule.Name))
	assert.True(t, ok)
	resolved := vcode.MustUnmarshalJsonS[*watched.PortalRule](value)
	assert.Equal(t, "/new", resolved.ResolvedMatchPathPrefix)
	assert.Equal(t, "/new", resolved.ResolvedRoutePathPrefix)
}

func TestSyncerWithdrawsWildcardRuleWhenSiteBecomesRpc(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)
	target := testSyncer(watchServer)
	target.SyncPortalEntry(&core.PortalEntry{Id: 1, Name: "wildcard", Scheme: "http", Host: "*.example.com", Port: 80, Enabled: true})
	site := &core.PortalSite{Id: 1, Name: "target", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web", Enabled: true}
	target.SyncPortalSite(site)
	target.SyncPortalRule(&core.PortalRule{Id: 1, Name: "wildcard", EntryId: 1, RouteType: "SITE", RouteSiteName: "target", Enabled: true})
	_, ok := watchServer.Get(watched.FormatPortalRuleKey("wildcard"))
	assert.True(t, ok)
	site.Type = core.PortalSiteTypeRPCGW
	target.SyncPortalSite(site)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("wildcard"))
	assert.False(t, ok)
	site.Type = core.PortalSiteTypeWEBGW
	target.SyncPortalSite(site)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("wildcard"))
	assert.True(t, ok)
	target.RemovePortalSite(site)
	_, ok = watchServer.Get(watched.FormatPortalRuleKey("wildcard"))
	assert.False(t, ok)
}
