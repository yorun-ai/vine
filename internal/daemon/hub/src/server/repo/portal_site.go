package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
)

type PortalSiteRepo struct {
	Dao        *model.PortalSiteDao `inject:""`
	SchemaRepo core.SchemaRepo      `inject:""`
	Syncer     *syncer.Syncer       `inject:""`
	Access     *configaccess.Access `inject:""`
}

func (s *PortalSiteRepo) List() []*core.PortalSite {
	schemas := s.schemas()
	rows := s.Dao.ListOrdered()
	entries := make([]*core.PortalSite, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, s.toCorePortalSite(row, schemas))
	}
	return entries
}

func (s *PortalSiteRepo) GetById(id int) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return s.toCorePortalSite(row, s.schemas()), true
	}
	return nil, false
}

func (s *PortalSiteRepo) GetByName(name string) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return s.toCorePortalSite(row, s.schemas()), true
	}
	return nil, false
}

func (s *PortalSiteRepo) Save(entry *core.PortalSite) {
	s.Access.CheckWrite()
	row := toModelPortalSite(entry)
	s.Dao.Save(row)

	saved := s.toCorePortalSite(row, s.schemas())
	*entry = *saved
	s.Syncer.SyncPortalSite(saved)
}

func (s *PortalSiteRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	entry, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalSite(s.toCorePortalSite(entry, s.schemas()))
	return true
}

// toCorePortalSite reconstitutes a portal site from its row and the schemas the
// applications registered: the Web mount path and the Rpc services the site
// forwards to are assembled here so the domain and the API read complete sites.
func (s *PortalSiteRepo) toCorePortalSite(row *model.PortalSite, schemas _PortalSiteSchemas) *core.PortalSite {
	cors := core.NormalizePortalCors(core.PortalCors{
		Mode:           core.PortalCorsMode(row.CorsMode),
		AllowedOrigins: decodePortalCorsOrigins(row.CorsOrigins),
	})
	entry := &core.PortalSite{
		FieldSources:  decodeFieldSources(row.FieldSources),
		Id:            row.Id,
		Name:          row.Name,
		Type:          core.PortalSiteType(row.Type),
		ActorSkelName: row.ActorSkelName,
		ActorVia:      row.ActorVia,
		Cors:          cors,
		WebName:       row.WebName,
		Enabled:       row.Enabled,
	}
	entry.WebMountPath = schemas.mountPath(entry)
	entry.RpcgwServices = core.MatchPortalSiteRpcgwServicesInDomainViews(*entry, schemas.views)
	return entry
}

// _PortalSiteSchemas is the schema state one repository call assembles sites
// from: the declared Web mount paths and the registered domain views.
type _PortalSiteSchemas struct {
	mountPaths map[string]string
	views      []core.DomainSchemaView
}

func (s *PortalSiteRepo) schemas() _PortalSiteSchemas {
	webs := s.SchemaRepo.ListWebSchemas()
	mountPaths := make(map[string]string, len(webs))
	for _, web := range webs {
		if web.MountPath != "" {
			mountPaths[web.SkelName] = web.MountPath
		}
	}
	return _PortalSiteSchemas{mountPaths: mountPaths, views: s.SchemaRepo.ListDomainSchemaViews()}
}

func (schemas _PortalSiteSchemas) mountPath(entry *core.PortalSite) string {
	if entry.Type != core.PortalSiteTypeWEBGW || entry.WebName == "" {
		return ""
	}
	return schemas.mountPaths[entry.WebName]
}

func toModelPortalSite(entry *core.PortalSite) *model.PortalSite {
	return &model.PortalSite{
		FieldSources:  encodeFieldSources(entry.FieldSources),
		Id:            entry.Id,
		Name:          entry.Name,
		Type:          string(entry.Type),
		ActorSkelName: entry.ActorSkelName,
		ActorVia:      entry.ActorVia,
		CorsMode:      string(entry.Cors.Mode),
		CorsOrigins:   vcode.MustMarshalJsonS(entry.Cors.AllowedOrigins),
		WebName:       entry.WebName,
		Enabled:       entry.Enabled,
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
