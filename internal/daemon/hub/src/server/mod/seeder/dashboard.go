package seeder

import (
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

var DashboardRpcServices = deriveDashboardRpcServiceNames()

var DashboardRpcCoreEntry = core.PortalSite{
	Name:          core.DashboardRpcSiteName,
	Type:          core.PortalSiteTypeRPCGW,
	ActorSkelName: skeled.AdminActor{}.SkelName(),
	ActorVia:      string(skel.ActorViaClient),
	BuiltIn:       true,
}

var DashboardWebCoreEntry = core.PortalSite{
	Name:          core.DashboardWebSiteName,
	Type:          core.PortalSiteTypeWEBGW,
	ActorSkelName: skeled.AdminActor{}.SkelName(),
	ActorVia:      string(skel.ActorViaClient),
	WebName:       "vine.hub.admin.DashboardWeb",
	BuiltIn:       true,
}

const (
	dashboardApiRuleName = "vine.hub.admin-api"
	dashboardWebRuleName = "vine.hub.dashboard-web"
)

func (s *Seeder) seedDashboard() {
	s.saveDashboardSite(DashboardRpcCoreEntry)
	s.saveDashboardSite(DashboardWebCoreEntry)

	url := s.Flag.DashboardURL
	refreshAccess := s.Flag.DashboardURLSet || s.canMigrateLegacyDashboardAccess()

	s.saveDashboardRule(core.PortalRule{
		Name:            dashboardApiRuleName,
		MatchScheme:     url.Scheme,
		MatchHost:       url.Hostname(),
		MatchPort:       url.Port(),
		MatchPathPrefix: "/api",
		RouteType:       "SITE",
		RouteSiteName:   DashboardRpcCoreEntry.Name,
		BuiltIn:         true,
	}, refreshAccess)
	s.saveDashboardRule(core.PortalRule{
		Name:            dashboardWebRuleName,
		MatchScheme:     url.Scheme,
		MatchHost:       url.Hostname(),
		MatchPort:       url.Port(),
		MatchPathPrefix: url.EscapedPath(),
		RouteType:       "SITE",
		RouteSiteName:   DashboardWebCoreEntry.Name,
		BuiltIn:         true,
	}, refreshAccess)
}

// saveDashboardSite keeps the stable database id and refreshes built-in
// site fields on every startup.
func (s *Seeder) saveDashboardSite(site core.PortalSite) {
	s.SiteCore.EnsureDashboardSite(site)
}

// saveDashboardRule refreshes built-in rule fields on every startup. Access
// fields are refreshed only for an explicit dashboard-url or when safely
// migrating the legacy built-in HTTP defaults to the mTLS HTTPS defaults.
func (s *Seeder) saveDashboardRule(rule core.PortalRule, refreshAccess bool) {
	s.RuleCore.EnsureDashboardRule(rule, refreshAccess)
}

func (s *Seeder) canMigrateLegacyDashboardAccess() bool {
	if !s.Flag.DashboardURLMTLSDefault {
		return false
	}
	legacyRules := []struct {
		name       string
		pathPrefix string
	}{
		{name: dashboardApiRuleName, pathPrefix: "/api"},
		{name: dashboardWebRuleName, pathPrefix: "/"},
	}
	for _, legacy := range legacyRules {
		rule, ok := s.RuleRepo.GetRuleByName(legacy.name)
		if !ok {
			continue
		}
		if rule.MatchScheme != "http" || rule.MatchHost != "" || rule.MatchPort != 7099 || rule.MatchPathPrefix != legacy.pathPrefix {
			return false
		}
	}
	return true
}

func deriveDashboardRpcServiceNames() []string {
	var names []string
	adminActorSkelName := skeled.AdminActor{}.SkelName()
	for _, domainSchema := range skel.RegisteredDomainSchemas() {
		if domainSchema.Domain != "vine.hub.admin" {
			continue
		}
		for _, service := range domainSchema.Services {
			if service.Pub {
				continue
			}
			for _, actor := range service.Audiences {
				if actor.SkelName == adminActorSkelName {
					names = append(names, service.SkelName)
					break
				}
			}
		}
	}
	return vslice.Sort(names)
}
