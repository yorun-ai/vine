package impl

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalEntryServiceServerImpl struct {
	skeled.DefaultPortalEntryServiceServer

	PortalEntryCore *core.PortalEntryCore `inject:""`
	PortalSiteCore  *core.PortalSiteCore  `inject:""`
}

func (s *PortalEntryServiceServerImpl) List() []skeled.PortalEntry {
	entries := s.PortalEntryCore.List()
	ret := make([]skeled.PortalEntry, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, s.toServerPortalEntry(entry))
	}
	return ret
}

func (s *PortalEntryServiceServerImpl) UpdateAccess(scheme string, host string, port int, update skeled.PortalEntryAccessUpdate) skeled.PortalEntry {
	entry := s.PortalEntryCore.UpdateAccess(scheme, host, port, core.PortalEntryAccessUpdate{
		Scheme: update.Scheme,
		Host:   update.Host,
		Port:   update.Port,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryServiceServerImpl) toServerPortalEntry(entry core.PortalEntry) skeled.PortalEntry {
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

func (s *PortalEntryServiceServerImpl) toServerPortalEntryRule(rule core.PortalEntryRule) skeled.PortalEntryRule {
	var site *skeled.PortalSite
	if rule.Site != nil {
		value := toServerPortalSite(*rule.Site, s.PortalSiteCore.RpcgwServices(*rule.Site))
		site = &value
	}
	return skeled.PortalEntryRule{
		Rule: toServerPortalRule(*rule.Rule),
		Site: site,
	}
}
