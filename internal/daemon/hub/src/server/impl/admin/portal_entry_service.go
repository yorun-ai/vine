package admin

import (
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalEntryApiServiceServerImpl struct {
	skeled.DefaultPortalEntryApiServiceServer

	PortalEntryCore *core.PortalEntryCore `inject:""`
}

func (s *PortalEntryApiServiceServerImpl) List() []skeled.PortalEntry {
	entries := s.PortalEntryCore.List()
	ret := make([]skeled.PortalEntry, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, s.toServerPortalEntry(entry))
	}
	return ret
}

func (s *PortalEntryApiServiceServerImpl) UpdateAccess(scheme string, host string, port int, update skeled.PortalEntryAccessUpdate) skeled.PortalEntry {
	entry := s.PortalEntryCore.UpdateAccess(scheme, host, port, core.PortalEntryAccessUpdate{
		Scheme: update.Scheme,
		Host:   update.Host,
		Port:   update.Port,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Create(creation skeled.PortalEntryCreation) skeled.PortalEntry {
	entry := s.PortalEntryCore.Create(core.PortalEntryCreation{
		Scheme: creation.Scheme,
		Host:   creation.Host,
		Port:   creation.Port,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Remove(scheme string, host string, port int) {
	s.PortalEntryCore.Remove(scheme, host, port)
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntry(entry core.PortalEntryView) skeled.PortalEntry {
	rules := make([]skeled.PortalEntryRule, 0, len(entry.Rules))
	for _, rule := range entry.Rules {
		rules = append(rules, s.toServerPortalEntryRule(rule))
	}
	return skeled.PortalEntry{
		Name:   entry.Name,
		Scheme: entry.Scheme,
		Host:   entry.Host,
		Port:   entry.Port,
		Rules:  rules,
	}
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntryRule(rule core.PortalEntryRule) skeled.PortalEntryRule {
	var site *skeled.PortalSiteListItem
	if rule.Site != nil {
		value := toServerPortalSiteListItem(rule.Site)
		site = &value
	}
	return skeled.PortalEntryRule{
		Rule: toServerPortalRuleListItem(rule.Rule, nil, rule.Site),
		Site: site,
	}
}
