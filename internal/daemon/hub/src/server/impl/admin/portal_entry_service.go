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

func (s *PortalEntryApiServiceServerImpl) Update(id int, update skeled.PortalEntryUpdate) skeled.PortalEntry {
	entry := s.PortalEntryCore.Update(id, core.PortalEntryUpdate{
		Name:    update.Name,
		Scheme:  update.Scheme,
		Host:    update.Host,
		Port:    update.Port,
		Enabled: update.Enabled,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Create(creation skeled.PortalEntryCreation) skeled.PortalEntry {
	entry := s.PortalEntryCore.Create(core.PortalEntryCreation{
		Name:    creation.Name,
		Scheme:  creation.Scheme,
		Host:    creation.Host,
		Port:    creation.Port,
		Enabled: creation.Enabled,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Remove(id int) {
	s.PortalEntryCore.Remove(id)
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntry(entry core.PortalEntryView) skeled.PortalEntry {
	rules := make([]skeled.PortalEntryRule, 0, len(entry.Rules))
	for _, rule := range entry.Rules {
		rules = append(rules, s.toServerPortalEntryRule(entry.PortalEntry, rule))
	}
	return skeled.PortalEntry{
		Id:      entry.Id,
		Name:    entry.Name,
		Scheme:  entry.Scheme,
		Host:    entry.Host,
		Port:    entry.Port,
		Enabled: entry.Enabled,
		Rules:   rules,
	}
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntryRule(entry core.PortalEntry, rule core.PortalEntryRule) skeled.PortalEntryRule {
	var site *skeled.PortalSiteListItem
	if rule.Site != nil {
		value := toServerPortalSiteListItem(rule.Site)
		site = &value
	}
	return skeled.PortalEntryRule{
		Rule: toServerPortalRuleListItem(&entry, rule.Rule, nil, rule.Site),
		Site: site,
	}
}
