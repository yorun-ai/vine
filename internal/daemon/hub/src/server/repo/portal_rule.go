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
	PortalSiteRepo  *PortalSiteRepo      `inject:""`
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

// ListViews resolves entry transports and target sites once per distinct reference.
func (s *PortalRuleRepo) ListViews() []core.PortalRuleView {
	rows := s.Dao.ListWithTargets()
	certs := make([]*core.PortalCert, 0, len(rows.Certs))
	for _, row := range rows.Certs {
		certs = append(certs, toCorePortalCert(row))
	}
	entries := make(map[int]*core.PortalEntry, len(rows.Entries))
	for _, row := range rows.Entries {
		entry := toCorePortalEntry(row)
		resolveEntryHTTP(entry, certs)
		entries[entry.Id] = entry
	}
	sites := make(map[string]*core.PortalSite, len(rows.Sites))
	for _, site := range s.PortalSiteRepo.sitesFromRows(rows.Sites) {
		sites[site.Name] = site
	}
	views := make([]core.PortalRuleView, 0, len(rows.Rules))
	for _, row := range rows.Rules {
		entry := entries[row.EntryId]
		views = append(views, core.PortalRuleView{
			Rule:  toCorePortalRule(row, entry),
			Entry: entry,
			Site:  sites[row.RouteSiteName],
		})
	}
	return views
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

func (s *PortalRuleRepo) Save(rules ...*core.PortalRule) {
	s.Access.PortalRouteWrites.Lock()
	defer s.Access.PortalRouteWrites.Unlock()
	s.Access.CheckWrite()
	// Each DAO save commits independently. If a later save fails, Watch must
	// still reflect every committed row, including any newly assigned IDs.
	saved := 0
	defer func() {
		if saved > 0 {
			s.Syncer.SyncPortalRules(rules[:saved])
		}
	}()
	for _, rule := range rules {
		entry := s.entryById(rule.EntryId)
		ex.PanicNewIfNot(entry != nil, ex.OperationFailed,
			ex.F("portal rule %q references missing portal entry %d", rule.Name, rule.EntryId))
		row := toModelPortalRule(rule)
		s.Dao.Save(row)
		rule.Id = row.Id
		saved++
	}
}

func (s *PortalRuleRepo) Remove(id int) bool {
	s.Access.PortalRouteWrites.Lock()
	defer s.Access.PortalRouteWrites.Unlock()
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

func toModelPortalRule(rule *core.PortalRule) *model.PortalRule {
	return &model.PortalRule{
		FieldSources:            encodeFieldSources(rule.FieldSources),
		Id:                      rule.Id,
		Name:                    rule.Name,
		EntryId:                 rule.EntryId,
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
