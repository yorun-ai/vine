package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestPortalEntryServiceMapsRulesAndTargetSites(t *testing.T) {
	ruleRepo := &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{
		"demo-rule": {
			Id:              1,
			Name:            "demo-rule",
			EntryId:         1,
			MatchPathPrefix: "/demo",
			RouteType:       core.PortalRuleRouteTypeSite,
			RouteSiteName:   "demo-site",
			FieldSources:    core.FieldSources{"/name": {Source: "app/default"}},
		},
	}}
	siteRepo := &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{
		"demo-site": {
			Id:            2,
			Name:          "demo-site",
			Type:          core.PortalSiteTypeWEBGW,
			ActorSkelName: "demo.Actor",
			ActorVia:      "client",
			WebName:       "demo.Web",
			WebMountPath:  "/demo",
			FieldSources:  core.FieldSources{"/name": {Source: "app/default"}},
		},
	}}
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{
		PortalEntryRepo: newTestPortalEntryRepoSpy(&core.PortalEntry{Id: 1, Name: "http:8080", Scheme: "http", Port: 8080}),
		PortalRuleRepo:  ruleRepo,
		PortalSiteRepo:  siteRepo,
	}}

	entries := service.List()

	require.Len(t, entries, 1)
	assert.Equal(t, 1, entries[0].Id)
	assert.Equal(t, "http:8080", entries[0].Name)
	require.Len(t, entries[0].Rules, 1)

	// The entry payload embeds the list shape of its rule and site: it carries the
	// values a list needs, never the seed provenance of the detail responses.
	assert.Equal(t, "demo-rule", entries[0].Rules[0].Rule.Name)
	require.NotNil(t, entries[0].Rules[0].Site)
	assert.Equal(t, "demo-site", entries[0].Rules[0].Site.Name)
	assert.Equal(t, "/demo", entries[0].Rules[0].Site.WebMountPath)
}

func TestPortalEntryServiceCreatesAndRemovesEntry(t *testing.T) {
	entryRepo := newTestPortalEntryRepoSpy()
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{}},
		PortalSiteRepo:  &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}},
	}}

	created := service.Create(skeled.PortalEntryCreation{Name: "web", Protocol: "http", Http: new(skeled.PortalEntryHttpUpdate{HttpPort: new(8080), HttpsEnabled: new(false), AutoHttps: new(false)}), Host: "", ListenIPs: []string{"127.0.0.1", "::1"}})

	// An entry routes no rule when the operator creates it.
	require.NotZero(t, created.Id)
	assert.Equal(t, "web", created.Name)
	assert.Equal(t, "http", created.Scheme)
	assert.Equal(t, 8080, created.Port)
	assert.Equal(t, []string{"127.0.0.1", "::1"}, created.ListenIPs)
	assert.Empty(t, created.Rules)

	updated := service.Update(created.Id, skeled.PortalEntryUpdate{Name: new("console"), ListenIPs: new([]string{"127.0.0.2"})})
	assert.Equal(t, "console", updated.Name)
	assert.Equal(t, 8080, updated.Port)
	assert.Equal(t, []string{"127.0.0.2"}, updated.ListenIPs)

	assert.Len(t, service.List(), 1)

	service.Remove(created.Id)

	assert.Empty(t, service.List())
}

func TestPortalEntryProtocolVocabulary(t *testing.T) {
	require.NotPanics(t, func() { validatePortalEntryProtocol(new("http")) })
	require.Panics(t, func() { validatePortalEntryProtocol(new("")) })
	require.Panics(t, func() { validatePortalEntryProtocol(new("tcp")) })
	patch := toCoreHTTPUpdate(new(skeled.PortalEntryHttpUpdate{HttpsEnabled: new(false), AutoHttps: new(false)}))
	config := patch.Apply(core.DefaultPortalEntryHTTP())
	require.True(t, config.HttpEnabled)
	require.False(t, config.HttpsEnabled)
	require.False(t, config.AutoHTTPS)
}

func TestPortalEntryServiceCreatesDefaultHTTPAndPatchesTransports(t *testing.T) {
	service := &PortalEntryApiServiceServerImpl{PortalEntryCore: &core.PortalEntryCore{
		PortalEntryRepo: newTestPortalEntryRepoSpy(), PortalRuleRepo: &_NamedPortalRuleRepoSpy{items: map[string]*core.PortalRule{}}, PortalSiteRepo: &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}},
	}}
	created := service.Create(skeled.PortalEntryCreation{Name: "dual", Protocol: "http", Host: "demo.local"})
	require.True(t, created.Http.HttpEnabled)
	require.True(t, created.Http.HttpsEnabled)
	require.True(t, created.Http.AutoHttps)
	require.Equal(t, 80, created.Http.HttpPort)
	require.Equal(t, 443, created.Http.HttpsPort)
	updated := service.Update(created.Id, skeled.PortalEntryUpdate{Protocol: new("http"), Http: new(skeled.PortalEntryHttpUpdate{HttpsPort: new(8443)})})
	require.Equal(t, created.Id, updated.Id)
	require.True(t, updated.Http.AutoHttps)
	require.Equal(t, 8443, updated.Http.HttpsPort)
	updated = service.Update(created.Id, skeled.PortalEntryUpdate{Protocol: new("http"), Http: new(skeled.PortalEntryHttpUpdate{HttpsEnabled: new(false), AutoHttps: new(false)})})
	require.True(t, updated.Http.HttpEnabled)
	require.False(t, updated.Http.HttpsEnabled)
	require.False(t, updated.Http.AutoHttps)
	require.Equal(t, 80, updated.Http.HttpPort)
	require.Panics(t, func() {
		service.Update(created.Id, skeled.PortalEntryUpdate{Protocol: new("http"), Http: new(skeled.PortalEntryHttpUpdate{AutoHttps: new(true)})})
	})
	require.False(t, service.List()[0].Http.AutoHttps)
}
