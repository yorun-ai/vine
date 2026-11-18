package syncer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleSitePromotesFallbackInSameRevision(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	target.SyncPortalEntry(&core.PortalEntry{Id: 1, Name: "web", Protocol: "http", Enabled: true})
	site := &core.PortalSite{Id: 1, Name: "docs", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/docs", Enabled: true}
	target.SyncPortalSite(site)

	preferred := &core.PortalRule{Id: 1, Name: "a-docs", EntryId: 1, Enabled: true, MatchPathPrefix: "/configured", RouteType: core.PortalRuleRouteTypeSite, RouteSiteName: "docs"}
	fallback := &core.PortalRule{Id: 2, Name: "b-redirect", EntryId: 1, Enabled: true, MatchPathPrefix: "/docs", RouteType: core.PortalRuleRouteTypeTemporaryRedirect, RouteRedirectionPattern: "/v1"}
	revision := testWatchRevision(t, server)
	target.SyncPortalRules([]*core.PortalRule{fallback, preferred})
	require.Equal(t, revision+1, testWatchRevision(t, server))
	requirePublishedRule(t, server, preferred.Name, "/docs")
	requireMissingRule(t, server, fallback.Name)

	fallback.RouteRedirectionPattern = "/v2"
	target.SyncPortalRule(fallback)
	require.Equal(t, revision+1, testWatchRevision(t, server))
	site.Enabled = false
	target.SyncPortalSite(site)
	require.Equal(t, revision+2, testWatchRevision(t, server))
	_, ok := server.Get(watched.FormatPortalSiteKey(site.Name))
	require.False(t, ok)
	requireMissingRule(t, server, preferred.Name)
	require.Equal(t, "/v2", requirePublishedRule(t, server, fallback.Name, "/docs").RouteRedirectionPattern)

	site.Enabled = true
	target.SyncPortalSite(site)
	require.Equal(t, revision+3, testWatchRevision(t, server))
	requirePublishedRule(t, server, preferred.Name, "/docs")
	requireMissingRule(t, server, fallback.Name)

	site.WebMountPath = "/guide"
	target.SyncPortalSite(site)
	require.Equal(t, revision+4, testWatchRevision(t, server))
	requirePublishedRule(t, server, preferred.Name, "/guide")
	requirePublishedRule(t, server, fallback.Name, "/docs")

	// A named site that disappears from a non-wildcard entry restores the rule's
	// configured prefix until the site returns.
	site.Name, site.Enabled = "renamed", false
	target.SyncPortalSite(site)
	require.Equal(t, revision+5, testWatchRevision(t, server))
	_, ok = server.Get(watched.FormatPortalSiteKey("docs"))
	require.False(t, ok)
	requirePublishedRule(t, server, preferred.Name, "/configured")
}

func TestPortalRuleMoveBetweenConflictGroups(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	target.SyncPortalEntry(&core.PortalEntry{Id: 1, Name: "web", Protocol: "http", Enabled: true})
	newRule := func(id int, name, prefix string) *core.PortalRule {
		return &core.PortalRule{Id: id, Name: name, EntryId: 1, Enabled: true, MatchPathPrefix: prefix, RouteType: core.PortalRuleRouteTypeTemporaryRedirect, RouteRedirectionPattern: "/v1"}
	}
	a, b, c := newRule(1, "a", "/old"), newRule(2, "b", "/old"), newRule(3, "c", "/new")
	target.SyncPortalRules([]*core.PortalRule{c, b, a})
	revision := testWatchRevision(t, server)
	a.MatchPathPrefix = "/new"
	target.SyncPortalRule(a)
	require.Equal(t, revision+1, testWatchRevision(t, server))
	requirePublishedRule(t, server, "a", "/new")
	requirePublishedRule(t, server, "b", "/old")
	requireMissingRule(t, server, "c")

	a.Name = "z"
	target.SyncPortalRule(a)
	require.Equal(t, revision+2, testWatchRevision(t, server))
	requireMissingRule(t, server, "a")
	requireMissingRule(t, server, "z")
	requirePublishedRule(t, server, "c", "/new")
	target.RemovePortalRule(c)
	require.Equal(t, revision+3, testWatchRevision(t, server))
	requirePublishedRule(t, server, "z", "/new")
}

func requirePublishedRule(t *testing.T, server *watchserver.Server, name, prefix string) *watched.PortalRule {
	t.Helper()
	raw, ok := server.Get(watched.FormatPortalRuleKey(name))
	require.True(t, ok, "missing rule %q", name)
	rule := vcode.MustUnmarshalJsonS[*watched.PortalRule](raw)
	require.Equal(t, prefix, rule.ResolvedMatchPathPrefix)
	return rule
}

func requireMissingRule(t *testing.T, server *watchserver.Server, name string) {
	t.Helper()
	_, ok := server.Get(watched.FormatPortalRuleKey(name))
	require.False(t, ok, "unexpected rule %q", name)
}

func TestPortalRuleRefreshUsesFinalSiteNamesAndMounts(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	target.SyncPortalEntry(&core.PortalEntry{Id: 1, Name: "web", Protocol: "http", Enabled: true})
	first := &core.PortalSite{Id: 1, Name: "first", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/first", Enabled: true}
	second := &core.PortalSite{Id: 2, Name: "second", Type: core.PortalSiteTypeWEBGW, WebMountPath: "/second", Enabled: true}
	target.SyncPortalSites([]*core.PortalSite{first, second})
	target.SyncPortalRules([]*core.PortalRule{
		{Id: 1, Name: "a", EntryId: 1, RouteType: core.PortalRuleRouteTypeSite, RouteSiteName: "first", Enabled: true},
		{Id: 2, Name: "b", EntryId: 1, RouteType: core.PortalRuleRouteTypeSite, RouteSiteName: "second", Enabled: true},
	})

	revision := testWatchRevision(t, server)
	first.Name, second.Name = second.Name, first.Name
	target.SyncPortalSites([]*core.PortalSite{first, second})
	require.Equal(t, revision+1, testWatchRevision(t, server))
	requirePublishedRule(t, server, "a", "/second")
	requirePublishedRule(t, server, "b", "/first")
}
