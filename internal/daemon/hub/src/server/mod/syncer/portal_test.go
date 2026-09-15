package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleTargetPathWatchRoundTrip(t *testing.T) {
	rule := &core.PortalRule{Name: "mapped", RouteType: "SITE", MatchPathPrefix: "/api", RoutePathPrefix: "/internal"}
	wire := vcode.MustMarshalJsonS(ToWatchedPortalRule(rule))
	decoded := vcode.MustUnmarshalJsonS[*watched.PortalRule](wire)
	assert.Equal(t, "/internal", decoded.RoutePathPrefix)
	assert.Contains(t, wire, `"routePathPrefix":"/internal"`)
	assert.Contains(t, wire, `"matchPathPrefix":"/api"`)
	assert.NotContains(t, wire, `"targetPath"`)
	assert.NotContains(t, wire, `"targetType"`)
}

func TestPortalSitePublishesWebMountPath(t *testing.T) {
	for _, mountPath := range []string{"", "/", "/app"} {
		site := &core.PortalSite{Name: "web", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web", WebMountPath: mountPath}
		wire := vcode.MustMarshalJsonS(toWatchedPortalSite(site))
		decoded := vcode.MustUnmarshalJsonS[watched.PortalSite](wire)
		assert.Equal(t, mountPath, decoded.WebgwConfig.MountPath)
		site.BuiltIn = true
		assert.Empty(t, toWatchedPortalSite(site).WebgwConfig.MountPath, "built-in Dashboard access remains configurable")
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
			assert.Equal(t, "/configured", got.MatchPathPrefix)
			assert.Equal(t, "/backend", got.RoutePathPrefix)
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
	}
	target.SyncPortalRule(rule)
	target.SyncPortalSite(&core.PortalSite{Id: 1, Name: "web", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/old"})
	value, ok := watchServer.Get(watched.FormatPortalRuleKey(rule.Name))
	assert.True(t, ok)
	assert.Equal(t, "/old", vcode.MustUnmarshalJsonS[*watched.PortalRule](value).ResolvedMatchPathPrefix)

	target.SyncPortalSite(&core.PortalSite{Id: 1, Name: "web", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/new/"})
	value, ok = watchServer.Get(watched.FormatPortalRuleKey(rule.Name))
	assert.True(t, ok)
	resolved := vcode.MustUnmarshalJsonS[*watched.PortalRule](value)
	assert.Equal(t, "/new", resolved.ResolvedMatchPathPrefix)
	assert.Equal(t, "/new", resolved.ResolvedRoutePathPrefix)
	assert.Equal(t, "/configured", resolved.MatchPathPrefix)
	assert.Equal(t, "/backend", resolved.RoutePathPrefix)
}
