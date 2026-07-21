package core

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

type PortalSiteType string

const (
	PortalSiteTypeRPCGW PortalSiteType = "RPCGW"
	PortalSiteTypeWEBGW PortalSiteType = "WEBGW"
)

type PortalCorsMode string

const (
	PortalCorsModeDisabled   PortalCorsMode = "DISABLED"
	PortalCorsModeSameDomain PortalCorsMode = "SAME_DOMAIN"
	PortalCorsModeStrict     PortalCorsMode = "STRICT"
)

type PortalCors struct {
	Mode           PortalCorsMode
	AllowedOrigins []string
}

func NormalizePortalCors(cors PortalCors) PortalCors {
	if cors.Mode == "" {
		cors.Mode = PortalCorsModeSameDomain
	}
	if cors.AllowedOrigins == nil {
		cors.AllowedOrigins = []string{}
	}
	return cors
}

type PortalSite struct {
	Id            int
	Name          string
	Type          PortalSiteType
	ActorSkelName string
	ActorVia      string
	Cors          PortalCors
	WebName       string
	BuiltIn       bool
}

type PortalSiteCreation struct {
	Name          string
	Type          PortalSiteType
	ActorSkelName string
	ActorVia      string
	Cors          PortalCors
	WebName       string
}

type PortalSiteUpdate struct {
	Name          *string
	Type          *PortalSiteType
	ActorSkelName *string
	ActorVia      *string
	Cors          *PortalCors
	WebName       *string
}

type PortalSiteActorOption struct {
	Name      string
	SkelName  string
	ActorVias []string
}

type PortalSiteServiceOption struct {
	Name           string
	SkelName       string
	ActorSkelNames []string
}

type PortalSiteWebOption struct {
	Name           string
	SkelName       string
	ActorSkelNames []string
}

type PortalSiteOptions struct {
	Actors   []PortalSiteActorOption
	Services []PortalSiteServiceOption
	Webs     []PortalSiteWebOption
}

type PortalSiteRepo interface {
	ListEntries() []PortalSite
	GetEntryById(id int) (*PortalSite, bool)
	GetEntryByName(name string) (*PortalSite, bool)
	SaveEntry(entry *PortalSite)
	RemoveEntry(id int) bool
}

type PortalSiteCore struct {
	PortalSiteRepo PortalSiteRepo `inject:""`
	SchemaRepo     SchemaRepo     `inject:""`
}

func (m *PortalSiteCore) List() []PortalSite {
	entries := m.PortalSiteRepo.ListEntries()
	ret := make([]PortalSite, 0, len(entries))
	for _, entry := range entries {
		if !entry.BuiltIn {
			ret = append(ret, entry)
		}
	}
	return ret
}

func (m *PortalSiteCore) ListOptions() PortalSiteOptions {
	return PortalSiteOptions{
		Actors:   toPortalSiteActorOptions(m.SchemaRepo.ListActorSchemas()),
		Services: toPortalSiteServiceOptions(m.SchemaRepo.ListServiceSchemas()),
		Webs:     toPortalSiteWebOptions(m.SchemaRepo.ListWebSchemas()),
	}
}

func (m *PortalSiteCore) Get(id int) PortalSite {
	entry, ok := m.PortalSiteRepo.GetEntryById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	return *entry
}

func (m *PortalSiteCore) RpcgwServices(site PortalSite) []string {
	if m.SchemaRepo == nil {
		return []string{}
	}
	return MatchPortalSiteRpcgwServicesInDomainViews(site, m.SchemaRepo.ListDomainSchemaViews())
}

func (m *PortalSiteCore) Create(creation PortalSiteCreation) PortalSite {
	_, ok := m.PortalSiteRepo.GetEntryByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("portal entry %q already exists", creation.Name))

	entry := PortalSite{
		Name:          creation.Name,
		Type:          creation.Type,
		ActorSkelName: creation.ActorSkelName,
		ActorVia:      creation.ActorVia,
		Cors:          creation.Cors,
		WebName:       creation.WebName,
	}
	m.PortalSiteRepo.SaveEntry(&entry)
	return entry
}

func (m *PortalSiteCore) Update(id int, update PortalSiteUpdate) PortalSite {
	entry, ok := m.PortalSiteRepo.GetEntryById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	ex.PanicNewIfNot(!entry.BuiltIn, ex.OperationFailed, ex.F("built-in portal entry %q cannot be updated", entry.Name))

	next := *entry
	if update.Name != nil {
		if *update.Name != entry.Name {
			_, exists := m.PortalSiteRepo.GetEntryByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("portal entry %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.Type != nil {
		next.Type = *update.Type
	}
	if update.ActorSkelName != nil {
		next.ActorSkelName = *update.ActorSkelName
	}
	if update.ActorVia != nil {
		next.ActorVia = *update.ActorVia
	}
	if update.Cors != nil {
		next.Cors = *update.Cors
	}
	if update.WebName != nil {
		next.WebName = *update.WebName
	}

	m.PortalSiteRepo.SaveEntry(&next)
	return next
}

func (m *PortalSiteCore) Remove(id int) {
	entry, ok := m.PortalSiteRepo.GetEntryById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	ex.PanicNewIfNot(!entry.BuiltIn, ex.OperationFailed, ex.F("built-in portal entry %q cannot be removed", entry.Name))

	ok = m.PortalSiteRepo.RemoveEntry(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
}

func toPortalSiteActorOptions(schemas []*skel.ActorSchema) []PortalSiteActorOption {
	options := make([]PortalSiteActorOption, 0, len(schemas))
	for _, schema := range schemas {
		actorVias := make([]string, 0, len(schema.Vias))
		for _, actorVia := range schema.Vias {
			actorVias = append(actorVias, string(actorVia))
		}
		options = append(options, PortalSiteActorOption{
			Name:      schema.Name,
			SkelName:  schema.SkelName,
			ActorVias: actorVias,
		})
	}
	return vslice.SortBy(options, func(a PortalSiteActorOption, b PortalSiteActorOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

func toPortalSiteServiceOptions(schemas []*skel.ServiceSchema) []PortalSiteServiceOption {
	options := make([]PortalSiteServiceOption, 0, len(schemas))
	for _, schema := range schemas {
		options = append(options, PortalSiteServiceOption{
			Name:           schema.Name,
			SkelName:       schema.SkelName,
			ActorSkelNames: actorSkelNames(schema.Audiences),
		})
	}
	return vslice.SortBy(options, func(a PortalSiteServiceOption, b PortalSiteServiceOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

func MatchPortalSiteRpcgwServicesInDomainViews(site PortalSite, views []DomainSchemaView) []string {
	if site.Type != PortalSiteTypeRPCGW {
		return []string{}
	}

	serviceNames := make([]string, 0)
	seen := map[string]struct{}{}
	for _, view := range views {
		if !view.DomainVersion.Main {
			continue
		}
		for _, schema := range view.DomainVersion.Schema.Services {
			if !schema.HasAudience(site.ActorSkelName, skel.ActorVia(site.ActorVia)) {
				continue
			}
			if _, ok := seen[schema.SkelName]; !ok {
				seen[schema.SkelName] = struct{}{}
				serviceNames = append(serviceNames, schema.SkelName)
			}
		}
	}
	return vslice.Sort(serviceNames)
}

func toPortalSiteWebOptions(schemas []*skel.WebSchema) []PortalSiteWebOption {
	options := make([]PortalSiteWebOption, 0, len(schemas))
	for _, schema := range schemas {
		options = append(options, PortalSiteWebOption{
			Name:           schema.Name,
			SkelName:       schema.SkelName,
			ActorSkelNames: actorSkelNames(schema.Audiences),
		})
	}
	return vslice.SortBy(options, func(a PortalSiteWebOption, b PortalSiteWebOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

func actorSkelNames(refs []*skel.ActorAudienceSchema) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.SkelName)
	}
	return names
}

func cmpString(a string, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
