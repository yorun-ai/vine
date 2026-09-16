package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestPortalEntryServiceMapsRulesAndTargetSites(t *testing.T) {
	ruleRepo := &_MaintenanceServicePortalRuleRepo{items: map[string]*core.PortalRule{
		"demo-rule": {
			Id:              1,
			Name:            "demo-rule",
			EntryId:         1,
			MatchScheme:     "http",
			MatchPort:       8080,
			MatchPathPrefix: "/demo",
			RouteType:       core.PortalRuleRouteTypeSite,
			RouteSiteName:   "demo-site",
			FieldSources:    core.FieldSources{"/name": {Source: "app/default"}},
		},
	}}
	siteRepo := &_MaintenanceServicePortalSiteRepo{items: map[string]*core.PortalSite{
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
	assert.Equal(t, "http:8080", entries[0].Name)
	require.Len(t, entries[0].Rules, 1)

	// The entry payload embeds the list shape of its rule and site: it carries the
	// values a list needs, never the seed provenance of the detail responses.
	assert.Equal(t, "demo-rule", entries[0].Rules[0].Rule.Name)
	require.NotNil(t, entries[0].Rules[0].Site)
	assert.Equal(t, "demo-site", entries[0].Rules[0].Site.Name)
	assert.Equal(t, "/demo", entries[0].Rules[0].Site.WebMountPath)
}
