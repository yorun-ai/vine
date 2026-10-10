package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
)

type PortalEntryRepo struct {
	Dao    *model.PortalEntryDao `inject:""`
	Syncer *syncer.Syncer        `inject:""`
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

func (s *PortalEntryRepo) GetByName(name string) (*core.PortalEntry, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return toCorePortalEntry(row), true
	}
	return nil, false
}

func (s *PortalEntryRepo) GetBySchemeHostPort(scheme string, host string, port int) (*core.PortalEntry, bool) {
	if row, ok := s.Dao.BySchemeHostPort(scheme, host, port); ok {
		return toCorePortalEntry(row), true
	}
	return nil, false
}

func (s *PortalEntryRepo) Save(entry *core.PortalEntry) {
	s.Access.CheckWrite()
	*entry = core.NormalizePortalEntry(*entry)
	row := toModelPortalEntry(entry)
	s.Dao.Save(row)
	entry.Id = row.Id
	s.Syncer.SyncPortalEntry(entry)
}

func (s *PortalEntryRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	row, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalEntry(toCorePortalEntry(row))
	return true
}

func toCorePortalEntry(row *model.PortalEntry) *core.PortalEntry {
	entry := &core.PortalEntry{
		Id:        row.Id,
		Name:      row.Name,
		Scheme:    row.Scheme,
		Host:      row.Host,
		Port:      row.Port,
		ListenIPs: vcode.MustUnmarshalJsonS[[]string](row.ListenIPs),
		Enabled:   row.Enabled,
	}
	if row.Protocol != "" && row.HTTPConfig != "" {
		entry.Protocol = row.Protocol
		entry.Http = vcode.MustUnmarshalJsonS[*core.PortalEntryHTTP](row.HTTPConfig)
	}
	return new(core.NormalizePortalEntry(*entry))
}

func toModelPortalEntry(entry *core.PortalEntry) *model.PortalEntry {
	entry = new(core.NormalizePortalEntry(*entry))
	return &model.PortalEntry{
		Id:         entry.Id,
		Name:       entry.Name,
		Protocol:   entry.Protocol,
		HTTPConfig: vcode.MustMarshalJsonS(entry.Http),
		Scheme:     entry.Scheme,
		Host:       entry.Host,
		Port:       entry.Port,
		ListenIPs:  vcode.MustMarshalJsonS(entry.ListenIPs),
		Enabled:    entry.Enabled,
	}
}
