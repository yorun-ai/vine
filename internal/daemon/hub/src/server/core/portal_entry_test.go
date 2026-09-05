package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

func TestPortalEntryCoreListMergesRulesBySchemeHostAndPort(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "admin", MatchScheme: "https", MatchPathPrefix: "/admin", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "admin-site", BuiltIn: true},
			2: {Id: 2, Name: "home", MatchScheme: "https", MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "home-site"},
			3: {Id: 3, Name: "api", MatchScheme: "http", MatchPort: 8080, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypePermanentRedirect, RouteRedirectionPattern: "https://demo.local"},
			4: {Id: 4, Name: "ignored", MatchScheme: "http", MatchPort: 8080, MatchPathPrefix: "/", RouteType: "UNSUPPORTED"},
			5: {Id: 5, Name: "hosted", MatchScheme: "http", MatchHost: "demo.local", MatchPort: 8080, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "home-site"},
		},
	}
	siteRepo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "admin-site"},
			2: {Id: 2, Name: "home-site"},
		},
	}
	core := &PortalEntryCore{PortalRuleRepo: repo, PortalSiteRepo: siteRepo}

	entries := core.List()

	require.Len(t, entries, 3)
	assert.Equal(t, "https:443", entries[0].Name)
	assert.Equal(t, "https", entries[0].Scheme)
	assert.Equal(t, "", entries[0].Host)
	assert.Equal(t, 443, entries[0].Port)
	assert.False(t, entries[0].BuiltIn)
	require.Len(t, entries[0].Rules, 1)
	assert.Equal(t, "home", entries[0].Rules[0].Rule.Name)
	assert.Equal(t, 2, entries[0].Rules[0].Site.Id)

	assert.Equal(t, "http:8080", entries[1].Name)
	assert.Equal(t, "http", entries[1].Scheme)
	assert.Equal(t, "", entries[1].Host)
	assert.Equal(t, 8080, entries[1].Port)
	assert.False(t, entries[1].BuiltIn)
	require.Len(t, entries[1].Rules, 1)
	assert.Equal(t, "api", entries[1].Rules[0].Rule.Name)

	assert.Equal(t, "http:demo.local:8080", entries[2].Name)
	assert.Equal(t, "http", entries[2].Scheme)
	assert.Equal(t, "demo.local", entries[2].Host)
	assert.Equal(t, 8080, entries[2].Port)
	require.Len(t, entries[2].Rules, 1)
	assert.Equal(t, "hosted", entries[2].Rules[0].Rule.Name)
	assert.Equal(t, []string{"ListRules"}, repo.calls)
}

func TestPortalEntryCoreListSkipsBuiltInRules(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, MatchScheme: "http", MatchPort: 7099, RouteType: PortalRuleRouteTypeSite, BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, MatchScheme: "http", MatchPort: 7099, RouteType: PortalRuleRouteTypeSite, BuiltIn: true},
			3: {Id: 3, Name: "demo", MatchScheme: "https", RouteType: PortalRuleRouteTypeSite},
		},
	}
	core := &PortalEntryCore{PortalRuleRepo: repo, PortalSiteRepo: &portalSiteRepoSpy{}}

	entries := core.List()

	require.Len(t, entries, 1)
	assert.Equal(t, "https:443", entries[0].Name)
	require.Len(t, entries[0].Rules, 1)
	assert.Equal(t, "demo", entries[0].Rules[0].Rule.Name)
}

func TestPortalEntryCoreListRejectsUnknownScheme(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "tcp", MatchScheme: "tcp", MatchPort: 9000, RouteType: PortalRuleRouteTypeSite},
		},
	}
	core := &PortalEntryCore{PortalRuleRepo: repo}

	panicValue := capturePanic(func() {
		core.List()
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalEntryCoreUpdateAccessUpdatesGroupedRules(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "web", MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
			2: {Id: 2, Name: "api", MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "rpc-site"},
			3: {Id: 3, Name: "other-host", MatchScheme: "http", MatchHost: "demo.local", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
			4: {Id: 4, Name: "redirect", MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/old", RouteType: PortalRuleRouteTypePermanentRedirect, RouteRedirectionPattern: "https://demo.local"},
			5: {Id: 5, Name: "vine", MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/vine", RouteType: PortalRuleRouteTypeSite, BuiltIn: true},
		},
	}
	core := &PortalEntryCore{PortalRuleRepo: repo, PortalSiteRepo: &portalSiteRepoSpy{}}

	entry := core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{
		Scheme: "https",
		Host:   "app.example.com",
		Port:   8443,
	})

	assert.Equal(t, "https:app.example.com:8443", entry.Name)
	assert.Equal(t, "https", entry.Scheme)
	assert.Equal(t, "app.example.com", entry.Host)
	assert.Equal(t, 8443, entry.Port)
	require.Len(t, entry.Rules, 3)
	assert.Equal(t, "https", repo.rules[1].MatchScheme)
	assert.Equal(t, "app.example.com", repo.rules[1].MatchHost)
	assert.Equal(t, 8443, repo.rules[1].MatchPort)
	assert.Equal(t, "https", repo.rules[2].MatchScheme)
	assert.Equal(t, "app.example.com", repo.rules[2].MatchHost)
	assert.Equal(t, 8443, repo.rules[2].MatchPort)
	assert.Equal(t, "http", repo.rules[3].MatchScheme)
	assert.Equal(t, "demo.local", repo.rules[3].MatchHost)
	assert.Equal(t, "https", repo.rules[4].MatchScheme)
	assert.Equal(t, "http", repo.rules[5].MatchScheme)
	assert.Equal(t, []string{"ListRules", "SaveRule", "SaveRule", "SaveRule", "ListRules"}, repo.calls)
}

func TestPortalEntryCoreUpdateAccessRejectsMissingEntry(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "web", MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		},
	}
	core := &PortalEntryCore{PortalRuleRepo: repo}

	panicValue := capturePanic(func() {
		core.UpdateAccess("http", "missing.local", 7088, PortalEntryAccessUpdate{Scheme: "http", Port: 8080})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"ListRules"}, repo.calls)
}
