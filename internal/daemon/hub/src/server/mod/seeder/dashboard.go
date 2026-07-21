package seeder

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

var DashboardRpcServices = deriveDashboardRpcServiceNames()

var DashboardRpcCoreEntry = core.PortalSite{
	Name:          "vine.hub.AdminActor-client-rpc",
	Type:          core.PortalSiteTypeRPCGW,
	ActorSkelName: skeled.AdminActor{}.SkelName(),
	ActorVia:      string(skel.ActorViaClient),
	BuiltIn:       true,
}

var DashboardWebCoreEntry = core.PortalSite{
	Name:          "vine.hub.DashboardWeb-web",
	Type:          core.PortalSiteTypeWEBGW,
	ActorSkelName: skeled.AdminActor{}.SkelName(),
	ActorVia:      string(skel.ActorViaClient),
	WebName:       "vine.hub.DashboardWeb",
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

	s.saveDashboardRule(core.PortalRule{
		Name:       dashboardApiRuleName,
		Scheme:     url.Scheme,
		Host:       url.Hostname(),
		Port:       url.Port(),
		PathPrefix: "/api",
		TargetType: "SITE",
		SiteName:   DashboardRpcCoreEntry.Name,
		BuiltIn:    true,
	})
	s.saveDashboardRule(core.PortalRule{
		Name:       dashboardWebRuleName,
		Scheme:     url.Scheme,
		Host:       url.Hostname(),
		Port:       url.Port(),
		PathPrefix: url.EscapedPath(),
		TargetType: "SITE",
		SiteName:   DashboardWebCoreEntry.Name,
		BuiltIn:    true,
	})
}

// saveDashboardSite keeps the stable database id and refreshes built-in
// site fields on every startup.
func (s *Seeder) saveDashboardSite(site core.PortalSite) {
	if oldEntry, ok := s.EntryRepo.GetEntryByName(site.Name); ok {
		site.Id = oldEntry.Id
	}
	s.EntryRepo.SaveEntry(&site)
}

// saveDashboardRule refreshes built-in rule fields on every startup.
// Access fields are refreshed only when dashboard-url is explicit.
func (s *Seeder) saveDashboardRule(rule core.PortalRule) {
	if oldRule, ok := s.RuleRepo.GetRuleByName(rule.Name); ok {
		rule.Id = oldRule.Id
		if !s.Flag.DashboardURLSet {
			rule.Scheme = oldRule.Scheme
			rule.Host = oldRule.Host
			rule.Port = oldRule.Port
			rule.PathPrefix = oldRule.PathPrefix
		}
	}
	s.RuleRepo.SaveRule(&rule)
}

func deriveDashboardRpcServiceNames() []string {
	var names []string
	adminActorSkelName := skeled.AdminActor{}.SkelName()
	for _, domainSchema := range skel.RegisteredDomainSchemas() {
		if domainSchema.Domain != "vine.hub" {
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
