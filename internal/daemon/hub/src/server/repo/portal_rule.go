package repo

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
)

type PortalRuleRepo struct {
	Dao             *model.PortalRuleDao `inject:""`
	Syncer          *syncer.Syncer       `inject:""`
	Access          *configaccess.Access `inject:""`
	PortalSiteRepo  core.PortalSiteRepo  `inject:""`
	PortalEntryRepo core.PortalEntryRepo `inject:""`
}

func (s *PortalRuleRepo) List() []*core.PortalRule {
	entries := s.entriesById()
	rows := s.Dao.ListOrdered()
	rules := make([]*core.PortalRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, toCorePortalRule(row, entries[row.EntryId]))
	}
	return rules
}

func (s *PortalRuleRepo) GetById(id int) (*core.PortalRule, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return toCorePortalRule(row, s.entryById(row.EntryId)), true
	}
	return nil, false
}

func (s *PortalRuleRepo) GetByName(name string) (*core.PortalRule, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return toCorePortalRule(row, s.entryById(row.EntryId)), true
	}
	return nil, false
}

func (s *PortalRuleRepo) Save(rule *core.PortalRule) {
	s.Access.CheckWrite()
	entry := s.entryById(rule.EntryId)
	ex.PanicNewIfNot(entry != nil, ex.OperationFailed,
		ex.F("portal rule %q references missing portal entry %d", rule.Name, rule.EntryId))
	row := toModelPortalRule(rule, entry)
	s.Dao.Save(row)
	rule.Id = row.Id

	var site *core.PortalSite
	if s.PortalSiteRepo != nil && rule.RouteSiteName != "" {
		site, _ = s.PortalSiteRepo.GetByName(rule.RouteSiteName)
	}
	s.Syncer.SyncPortalRule(rule, site)
}

func (s *PortalRuleRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	rule, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalRule(toCorePortalRule(rule, s.entryById(rule.EntryId)))
	return true
}

// toCorePortalRule reconstitutes a rule from its row. The rule owns the entry it
// belongs to, never the access the entry serves.
func toCorePortalRule(row *model.PortalRule, entry *core.PortalEntry) *core.PortalRule {
	ex.PanicNewIfNot(entry != nil, ex.OperationFailed,
		ex.F("portal rule %q references missing portal entry %d", row.Name, row.EntryId))
	return &core.PortalRule{
		FieldSources:            decodeFieldSources(row.FieldSources),
		Id:                      row.Id,
		Name:                    row.Name,
		EntryId:                 row.EntryId,
		MatchPathPrefix:         row.MatchPathPrefix,
		RouteType:               row.RouteType,
		RouteSiteName:           row.RouteSiteName,
		RouteRedirectionPattern: row.RouteRedirectionPattern,
		RoutePathPrefix:         row.RoutePathPrefix,
		Enabled:                 row.Enabled,
	}
}

// toModelPortalRule fills the access columns older Hub versions still read.
func toModelPortalRule(rule *core.PortalRule, entry *core.PortalEntry) *model.PortalRule {
	return &model.PortalRule{
		FieldSources:            encodeFieldSources(rule.FieldSources),
		Id:                      rule.Id,
		Name:                    rule.Name,
		EntryId:                 rule.EntryId,
		MatchScheme:             entry.Scheme,
		MatchHost:               entry.Host,
		MatchPort:               entry.Port,
		MatchPathPrefix:         rule.MatchPathPrefix,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		RoutePathPrefix:         rule.RoutePathPrefix,
		Enabled:                 rule.Enabled,
	}
}

func (s *PortalRuleRepo) entriesById() map[int]*core.PortalEntry {
	entries := map[int]*core.PortalEntry{}
	for _, entry := range s.PortalEntryRepo.List() {
		entries[entry.Id] = entry
	}
	return entries
}

func (s *PortalRuleRepo) entryById(id int) *core.PortalEntry {
	entry, ok := s.PortalEntryRepo.GetById(id)
	if !ok {
		return nil
	}
	return entry
}
