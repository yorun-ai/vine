package impl

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
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
	return toServerPortalRule(s.PortalRuleCore.Create(core.PortalRuleCreation{
		Name:               creation.Name,
		Scheme:             creation.Scheme,
		Host:               creation.Host,
		Port:               creation.Port,
		PathPrefix:         creation.PathPrefix,
		TargetType:         creation.TargetType,
		SiteName:           creation.SiteName,
		RedirectionPattern: creation.RedirectionPattern,
	}))
}

func (s *PortalRuleServiceServerImpl) Update(id int, update skeled.PortalRuleUpdate) skeled.PortalRule {
	return toServerPortalRule(s.PortalRuleCore.Update(id, core.PortalRuleUpdate{
		Name:               update.Name,
		Scheme:             update.Scheme,
		Host:               update.Host,
		Port:               update.Port,
		PathPrefix:         update.PathPrefix,
		TargetType:         update.TargetType,
		SiteName:           update.SiteName,
		RedirectionPattern: update.RedirectionPattern,
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
		Id:                 rule.Id,
		Name:               rule.Name,
		Scheme:             rule.Scheme,
		Host:               rule.Host,
		Port:               rule.Port,
		PathPrefix:         rule.PathPrefix,
		TargetType:         rule.TargetType,
		SiteName:           rule.SiteName,
		RedirectionPattern: rule.RedirectionPattern,
	}
}
