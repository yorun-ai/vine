package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type PortalRuleApiServiceServerImpl struct {
	skeled.DefaultPortalRuleApiServiceServer

	PortalRuleCore *core.PortalRuleCore `inject:""`
	PortalSiteRepo core.PortalSiteRepo  `inject:""`
	Flag           *flag.Flag           `inject:""`
}

func (s *PortalRuleApiServiceServerImpl) List() []skeled.PortalRuleListItem {
	rules := s.PortalRuleCore.List()
	ret := make([]skeled.PortalRuleListItem, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, toServerPortalRuleListItem(rule, s.PortalSiteRepo))
	}
	return ret
}

func (s *PortalRuleApiServiceServerImpl) Get(id int) skeled.PortalRule {
	rule := s.PortalRuleCore.Get(id)
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Create(creation skeled.PortalRuleCreation) skeled.PortalRule {
	var routePathPrefix string
	if creation.RoutePathPrefix != nil {
		routePathPrefix = *creation.RoutePathPrefix
	}
	rule := s.PortalRuleCore.Create(core.PortalRuleCreation{
		Name:                    creation.Name,
		EntryName:               creation.EntryName,
		MatchPathPrefix:         creation.MatchPathPrefix,
		RouteType:               creation.RouteType,
		RouteSiteName:           creation.RouteSiteName,
		RouteRedirectionPattern: creation.RouteRedirectionPattern,
		RoutePathPrefix:         routePathPrefix,
	})
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Update(id int, update skeled.PortalRuleUpdate) skeled.PortalRule {
	rule := s.PortalRuleCore.Update(id, core.PortalRuleUpdate{
		Name:                    update.Name,
		MatchPathPrefix:         update.MatchPathPrefix,
		RouteType:               update.RouteType,
		RouteSiteName:           update.RouteSiteName,
		RouteRedirectionPattern: update.RouteRedirectionPattern,
		RoutePathPrefix:         update.RoutePathPrefix,
	})
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Remove(id int) {
	s.PortalRuleCore.Remove(id)
}

func (s *PortalRuleApiServiceServerImpl) GetDashboardAccess() skeled.PortalDashboardAccess {
	access := s.PortalRuleCore.DashboardAccess()
	return skeled.PortalDashboardAccess{
		Scheme:     access.Scheme,
		Host:       access.Host,
		Port:       access.Port,
		PathPrefix: access.PathPrefix,
		CanUpdate:  !s.Flag.DashboardURLSet,
	}
}

func (s *PortalRuleApiServiceServerImpl) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) []skeled.PortalRule {
	ex.PanicNewIfNot(!s.Flag.DashboardURLSet, ex.OperationFailed, "dashboard access is configured by dashboard-url")
	rules := s.PortalRuleCore.UpdateDashboardAccess(scheme, host, port, pathPrefix)
	ret := make([]skeled.PortalRule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, s.toServerPortalRule(rule, nil))
	}
	return ret
}

func toServerPortalRule(rule *core.PortalRule, fieldSources []skeled.FieldSource) skeled.PortalRule {
	resolvedMatchPathPrefix, resolvedRoutePathPrefix := resolvePortalRulePaths(rule, nil)
	return skeled.PortalRule{
		Id:                      rule.Id,
		Name:                    rule.Name,
		MatchScheme:             rule.MatchScheme,
		MatchHost:               rule.MatchHost,
		MatchPort:               rule.MatchPort,
		MatchPathPrefix:         rule.MatchPathPrefix,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		RoutePathPrefix:         rule.RoutePathPrefix,
		ResolvedMatchPathPrefix: resolvedMatchPathPrefix,
		ResolvedRoutePathPrefix: resolvedRoutePathPrefix,
		FieldSources:            fieldSources,
	}
}

func (s *PortalRuleApiServiceServerImpl) toServerPortalRule(rule *core.PortalRule, fieldSources []skeled.FieldSource) skeled.PortalRule {
	ret := toServerPortalRule(rule, fieldSources)
	if s.PortalSiteRepo != nil {
		if site, ok := s.PortalSiteRepo.GetByName(rule.RouteSiteName); ok {
			ret.ResolvedMatchPathPrefix, ret.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, site)
		}
	}
	return ret
}

func resolvePortalRulePaths(rule *core.PortalRule, site *core.PortalSite) (string, string) {
	return core.ResolvePortalRulePaths(rule, site)
}

// toServerPortalRuleListItem maps a rule for list responses, which carry the
// entity values without its seed provenance.
func toServerPortalRuleListItem(rule *core.PortalRule, siteRepo core.PortalSiteRepo, sites ...*core.PortalSite) skeled.PortalRuleListItem {
	detail := toServerPortalRule(rule, nil)
	if len(sites) > 0 {
		detail.ResolvedMatchPathPrefix, detail.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, sites[0])
	} else if siteRepo != nil {
		if site, ok := siteRepo.GetByName(rule.RouteSiteName); ok {
			detail.ResolvedMatchPathPrefix, detail.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, site)
		}
	}
	return skeled.PortalRuleListItem{
		Id:                      detail.Id,
		Name:                    detail.Name,
		MatchScheme:             detail.MatchScheme,
		MatchHost:               detail.MatchHost,
		MatchPort:               detail.MatchPort,
		MatchPathPrefix:         detail.MatchPathPrefix,
		RouteType:               detail.RouteType,
		RouteSiteName:           detail.RouteSiteName,
		RouteRedirectionPattern: detail.RouteRedirectionPattern,
		RoutePathPrefix:         detail.RoutePathPrefix,
		ResolvedMatchPathPrefix: detail.ResolvedMatchPathPrefix,
		ResolvedRoutePathPrefix: detail.ResolvedRoutePathPrefix,
	}
}
