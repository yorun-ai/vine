package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
)

type DBPortalSiteRepo struct {
	Dao        *model.PortalSiteDao `inject:""`
	SchemaRepo core.SchemaRepo      `inject:""`
	Syncer     *syncer.Syncer       `inject:""`
}

func (s *DBPortalSiteRepo) ListEntries() []core.PortalSite {
	rows := s.Dao.ListOrdered()
	entries := make([]core.PortalSite, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, *toCorePortalSite(row))
	}
	return entries
}

func (s *DBPortalSiteRepo) GetEntryById(id int) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return toCorePortalSite(row), true
	}
	return nil, false
}

func (s *DBPortalSiteRepo) GetEntryByName(name string) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return toCorePortalSite(row), true
	}
	return nil, false
}

func (s *DBPortalSiteRepo) SaveEntry(entry *core.PortalSite) {
	row := toDBPortalSite(entry)
	s.Dao.Save(row)
	entry.Id = row.Id

	s.Syncer.SyncPortalSiteWithRpcgwServices(entry, s.rpcgwServices(entry))
}

func (s *DBPortalSiteRepo) rpcgwServices(entry *core.PortalSite) []string {
	return core.MatchPortalSiteRpcgwServicesInDomainViews(*entry, s.SchemaRepo.ListDomainSchemaViews())
}

func (s *DBPortalSiteRepo) RemoveEntry(id int) bool {
	entry, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalSite(toCorePortalSite(entry))
	return true
}

func toCorePortalSite(row *model.PortalSite) *core.PortalSite {
	cors := core.NormalizePortalCors(core.PortalCors{
		Mode:           core.PortalCorsMode(row.CorsMode),
		AllowedOrigins: decodePortalCorsOrigins(row.CorsOrigins),
	})
	return &core.PortalSite{
		Id:            row.Id,
		Name:          row.Name,
		Type:          core.PortalSiteType(row.Type),
		ActorSkelName: row.ActorSkelName,
		ActorVia:      row.ActorVia,
		Cors:          cors,
		WebName:       row.WebName,
		BuiltIn:       row.BuiltIn,
	}
}

func toDBPortalSite(entry *core.PortalSite) *model.PortalSite {
	return &model.PortalSite{
		Model:         rdb.Model{Id: entry.Id},
		Name:          entry.Name,
		Type:          string(entry.Type),
		ActorSkelName: entry.ActorSkelName,
		ActorVia:      entry.ActorVia,
		CorsMode:      string(entry.Cors.Mode),
		CorsOrigins:   vcode.MustMarshalJsonS(entry.Cors.AllowedOrigins),
		WebName:       entry.WebName,
		BuiltIn:       entry.BuiltIn,
	}
}

func decodePortalCorsOrigins(value string) []string {
	if value == "" || value == "null" {
		return []string{}
	}
	origins := vcode.MustUnmarshalJsonS[[]string](value)
	if origins == nil {
		return []string{}
	}
	return origins
}
