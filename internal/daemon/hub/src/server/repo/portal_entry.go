package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
)

type PortalEntryRepo struct {
	Dao    *model.PortalEntryDao `inject:""`
	Access *configaccess.Access  `inject:""`
}

func (s *PortalEntryRepo) List() []*core.PortalEntry {
	rows := s.Dao.ListOrdered()
	entries := make([]*core.PortalEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, toCorePortalEntry(row))
	}
	return entries
}

func (s *PortalEntryRepo) GetById(id int) (*core.PortalEntry, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return toCorePortalEntry(row), true
	}
	return nil, false
}

func (s *PortalEntryRepo) GetByAccess(scheme string, host string, port int) (*core.PortalEntry, bool) {
	if row, ok := s.Dao.ByAccess(scheme, host, port); ok {
		return toCorePortalEntry(row), true
	}
	return nil, false
}

func (s *PortalEntryRepo) GetBuiltIn() (*core.PortalEntry, bool) {
	if row, ok := s.Dao.BuiltIn(); ok {
		return toCorePortalEntry(row), true
	}
	return nil, false
}

func (s *PortalEntryRepo) Save(entry *core.PortalEntry) {
	s.Access.CheckWrite()
	row := toModelPortalEntry(entry)
	s.Dao.Save(row)
	entry.Id = row.Id
}

func (s *PortalEntryRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	if _, ok := s.Dao.DeleteById(id); !ok {
		return false
	}
	return true
}

func toCorePortalEntry(row *model.PortalEntry) *core.PortalEntry {
	// BuiltIn is carried through so the rule repository recognizes the rules of
	// Hub's own entry; the entry list filters built-in entries out.
	return &core.PortalEntry{
		Id:      row.Id,
		Name:    core.PortalEntryName(row.Scheme, row.Host, row.Port),
		Scheme:  row.Scheme,
		Host:    row.Host,
		Port:    row.Port,
		BuiltIn: row.BuiltIn,
	}
}

func toModelPortalEntry(entry *core.PortalEntry) *model.PortalEntry {
	return &model.PortalEntry{
		Id:      entry.Id,
		Scheme:  entry.Scheme,
		Host:    entry.Host,
		Port:    entry.Port,
		BuiltIn: entry.BuiltIn,
	}
}
