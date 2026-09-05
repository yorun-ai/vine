package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type PortalRuleServiceServerImpl struct {
	skeled.DefaultPortalRuleServiceServer

	PortalRuleCore *core.PortalRuleCore `inject:""`
	Flag           *flag.Flag           `inject:""`
}

func (s *PortalRuleServiceServerImpl) List() []skeled.PortalRule {
	rules := s.PortalRuleCore.List()
	ret := make([]skeled.PortalRule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, toServerPortalRule(rule))
	}
	return ret
}

func (s *PortalRuleServiceServerImpl) Get(id int) skeled.PortalRule {
	return toServerPortalRule(s.PortalRuleCore.Get(id))
}

func (s *PortalRuleServiceServerImpl) Create(creation skeled.PortalRuleCreation) skeled.PortalRule {
	var routePathPrefix string
	if creation.RoutePathPrefix != nil {
		routePathPrefix = *creation.RoutePathPrefix
	}
	return toServerPortalRule(s.PortalRuleCore.Create(core.PortalRuleCreation{
		Name:                    creation.Name,
		MatchScheme:             creation.MatchScheme,
		MatchHost:               creation.MatchHost,
		MatchPort:               creation.MatchPort,
		MatchPathPrefix:         creation.MatchPathPrefix,
		RouteType:               creation.RouteType,
		RouteSiteName:           creation.RouteSiteName,
		RouteRedirectionPattern: creation.RouteRedirectionPattern,
		RoutePathPrefix:         routePathPrefix,
	}))
}

func (s *PortalRuleServiceServerImpl) Update(id int, update skeled.PortalRuleUpdate) skeled.PortalRule {
	return toServerPortalRule(s.PortalRuleCore.Update(id, core.PortalRuleUpdate{
		Name:                    update.Name,
		MatchScheme:             update.MatchScheme,
		MatchHost:               update.MatchHost,
		MatchPort:               update.MatchPort,
		MatchPathPrefix:         update.MatchPathPrefix,
		RouteType:               update.RouteType,
		RouteSiteName:           update.RouteSiteName,
		RouteRedirectionPattern: update.RouteRedirectionPattern,
		RoutePathPrefix:         update.RoutePathPrefix,
	}))
}

func (s *PortalRuleServiceServerImpl) Remove(id int) {
	s.PortalRuleCore.Remove(id)
}

func (s *PortalRuleServiceServerImpl) GetDashboardAccess() skeled.PortalDashboardAccess {
	access := s.PortalRuleCore.DashboardAccess()
	return skeled.PortalDashboardAccess{
		Scheme:     access.Scheme,
		Host:       access.Host,
		Port:       access.Port,
		PathPrefix: access.PathPrefix,
		CanUpdate:  !s.Flag.DashboardURLSet,
	}
}

func (s *PortalRuleServiceServerImpl) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) []skeled.PortalRule {
	ex.PanicNewIfNot(!s.Flag.DashboardURLSet, ex.OperationFailed, "dashboard access is configured by dashboard-url")
	rules := s.PortalRuleCore.UpdateDashboardAccess(scheme, host, port, pathPrefix)
	ret := make([]skeled.PortalRule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, toServerPortalRule(rule))
	}
	return ret
}

func toServerPortalRule(rule core.PortalRule) skeled.PortalRule {
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
	}
}
