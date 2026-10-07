package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
)

type PortalSiteRepo struct {
	Dao            *model.PortalSiteDao `inject:""`
	DescriptorRepo core.DescriptorRepo  `inject:""`
	Syncer         *syncer.Syncer       `inject:""`
	Access         *configaccess.Access `inject:""`
}

func (s *PortalSiteRepo) List() []*core.PortalSite {
	descriptors := s.descriptors()
	rows := s.Dao.ListOrdered()
	entries := make([]*core.PortalSite, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, s.toCorePortalSite(row, descriptors))
	}
	return entries
}

func (s *PortalSiteRepo) GetById(id int) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return s.toCorePortalSite(row, s.descriptors()), true
	}
	return nil, false
}

func (s *PortalSiteRepo) GetByName(name string) (*core.PortalSite, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return s.toCorePortalSite(row, s.descriptors()), true
	}
	return nil, false
}

func (s *PortalSiteRepo) Save(entry *core.PortalSite) {
	s.Access.CheckWrite()
	row := toModelPortalSite(entry)
	s.Dao.Save(row)

	saved := s.toCorePortalSite(row, s.descriptors())
	*entry = *saved
	s.Syncer.SyncPortalSite(saved)
}

func (s *PortalSiteRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	entry, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalSite(s.toCorePortalSite(entry, s.descriptors()))
	return true
}

// toCorePortalSite reconstitutes a portal site from its row and the descriptors the
// applications registered: the Web mount path and the Rpc services the site
// forwards to are assembled here so the domain and the API read complete sites.
func (s *PortalSiteRepo) toCorePortalSite(row *model.PortalSite, descriptors _PortalSiteDescriptors) *core.PortalSite {
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
	entry.WebMountPath = descriptors.mountPath(entry)
	entry.RpcgwServices = core.MatchPortalSiteRpcgwServicesInDomainViews(*entry, descriptors.views)
	return entry
}

// _PortalSiteDescriptors is the descriptor state one repository call assembles sites
// from: the declared Web mount paths and the registered domain views.
type _PortalSiteDescriptors struct {
	mountPaths map[string]string
	views      []core.DomainDescriptorView
}

func (s *PortalSiteRepo) descriptors() _PortalSiteDescriptors {
	webs := s.DescriptorRepo.ListWebDescriptors()
	mountPaths := make(map[string]string, len(webs))
	for _, web := range webs {
		if web.MountPath != "" {
			mountPaths[web.SkelName] = web.MountPath
		}
	}
	return _PortalSiteDescriptors{
		mountPaths: mountPaths,
		views:      s.DescriptorRepo.ListDomainDescriptorViews(),
	}
}

func (descriptors _PortalSiteDescriptors) mountPath(entry *core.PortalSite) string {
	if entry.Type != core.PortalSiteTypeWEBGW || entry.WebName == "" {
		return ""
	}
	return descriptors.mountPaths[entry.WebName]
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
