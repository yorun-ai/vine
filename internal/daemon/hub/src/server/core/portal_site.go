package core

import (
	"net/url"
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
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
	FieldSources  FieldSources
	Id            int
	Name          string
	Type          PortalSiteType
	ActorSkelName string
	ActorVia      string
	Cors          PortalCors
	WebName       string
	// WebMountPath is derived from the Web contract of the site and is empty when
	// the Web is not limited to a path.
	WebMountPath string
	// RpcgwServices is derived from the registered services of the site actor and
	// is empty for sites that do not forward Rpc traffic.
	RpcgwServices []string
	// Enabled decides whether Hub publishes the site to Portal.
	Enabled bool
}

type PortalSiteCreation struct {
	Name          string
	Type          PortalSiteType
	ActorSkelName string
	ActorVia      string
	Cors          PortalCors
	WebName       string
	// Enabled is optional and defaults to true.
	Enabled *bool
}

type PortalSiteUpdate struct {
	Name          *string
	Type          *PortalSiteType
	ActorSkelName *string
	ActorVia      *string
	Cors          *PortalCors
	WebName       *string
	Enabled       *bool
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

// PortalSiteRepo stores Portal sites. List and the lookups return entities the
// caller owns; storage is free to assemble them from more than one source.
type PortalSiteRepo interface {
	List() []*PortalSite
	GetById(id int) (*PortalSite, bool)
	GetByName(name string) (*PortalSite, bool)
	Save(entry *PortalSite)
	Remove(id int) bool
}

type PortalSiteCore struct {
	PortalSiteRepo PortalSiteRepo `inject:""`
	DescriptorRepo DescriptorRepo `inject:""`
}

func (m *PortalSiteCore) List() []*PortalSite {
	entries := m.PortalSiteRepo.List()
	ret := make([]*PortalSite, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, entry)
	}
	return ret
}

func (m *PortalSiteCore) ListOptions() PortalSiteOptions {
	return PortalSiteOptions{
		Actors:   toPortalSiteActorOptions(m.DescriptorRepo.ListActorDescriptors()),
		Services: toPortalSiteServiceOptions(m.DescriptorRepo.ListServiceDescriptors()),
		Webs:     toPortalSiteWebOptions(m.DescriptorRepo.ListWebDescriptors()),
	}
}

func (m *PortalSiteCore) Get(id int) *PortalSite {
	entry, ok := m.PortalSiteRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	return entry
}

// FindByName returns a complete portal site or false when no site uses the name.
func (m *PortalSiteCore) FindByName(name string) (*PortalSite, bool) {
	return m.PortalSiteRepo.GetByName(name)
}

func (m *PortalSiteCore) Create(creation PortalSiteCreation) *PortalSite {
	_, ok := m.PortalSiteRepo.GetByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("portal entry %q already exists", creation.Name))

	entry := PortalSite{
		Name:          creation.Name,
		Type:          creation.Type,
		ActorSkelName: creation.ActorSkelName,
		ActorVia:      creation.ActorVia,
		Cors:          creation.Cors,
		WebName:       creation.WebName,
		Enabled:       EnabledOrDefault(creation.Enabled),
	}
	entry = m.Validate(entry)
	m.PortalSiteRepo.Save(&entry)
	return &entry
}

func (m *PortalSiteCore) Update(id int, update PortalSiteUpdate) *PortalSite {
	entry, ok := m.PortalSiteRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))

	next := *entry
	next.FieldSources = cloneFieldSources(entry.FieldSources)
	if update.Name != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/name")
		if *update.Name != entry.Name {
			_, exists := m.PortalSiteRepo.GetByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("portal entry %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.Type != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/type")
		next.Type = *update.Type
	}
	if update.ActorSkelName != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/actorSkelName")
		next.ActorSkelName = *update.ActorSkelName
	}
	if update.ActorVia != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/actorVia")
		next.ActorVia = *update.ActorVia
	}
	if update.Cors != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/cors")
		next.Cors = *update.Cors
	}
	if update.WebName != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/webName")
		next.WebName = *update.WebName
	}
	if update.Enabled != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/enabled")
		next.Enabled = *update.Enabled
	}

	next = m.Validate(next)
	m.PortalSiteRepo.Save(&next)
	return &next
}

func (m *PortalSiteCore) Remove(id int) {
	_, ok := m.PortalSiteRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))

	ok = m.PortalSiteRepo.Remove(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
}

func toPortalSiteActorOptions(descriptors []*skeldesc.Actor) []PortalSiteActorOption {
	options := make([]PortalSiteActorOption, 0, len(descriptors))
	for _, descriptor := range descriptors {
		actorVias := make([]string, 0, len(descriptor.Vias))
		for _, actorVia := range descriptor.Vias {
			actorVias = append(actorVias, string(actorVia))
		}
		options = append(options, PortalSiteActorOption{
			Name:      descriptor.Name,
			SkelName:  descriptor.SkelName,
			ActorVias: actorVias,
		})
	}
	return vslice.SortBy(options, func(a PortalSiteActorOption, b PortalSiteActorOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

func toPortalSiteServiceOptions(descriptors []*skeldesc.Service) []PortalSiteServiceOption {
	options := make([]PortalSiteServiceOption, 0, len(descriptors))
	for _, descriptor := range descriptors {
		options = append(options, PortalSiteServiceOption{
			Name:           descriptor.Name,
			SkelName:       descriptor.SkelName,
			ActorSkelNames: actorSkelNames(descriptor.Audiences),
		})
	}
	return vslice.SortBy(options, func(a PortalSiteServiceOption, b PortalSiteServiceOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

// MatchPortalSiteRpcgwServicesInDomainViews returns the Rpc services a site
// forwards to, matching the services registered for its actor and access mode.
func MatchPortalSiteRpcgwServicesInDomainViews(site PortalSite, views []DomainDescriptorView) []string {
	if site.Type != PortalSiteTypeRPCGW {
		return []string{}
	}

	serviceNames := make([]string, 0)
	seen := map[string]struct{}{}
	for _, view := range views {
		if !view.DomainVersion.Main {
			continue
		}
		for _, service := range view.DomainVersion.Descriptor.Services {
			if !service.HasAudience(site.ActorSkelName, skeldesc.ActorViaKind(site.ActorVia)) {
				continue
			}
			if _, ok := seen[service.SkelName]; !ok {
				seen[service.SkelName] = struct{}{}
				serviceNames = append(serviceNames, service.SkelName)
			}
		}
	}
	return vslice.Sort(serviceNames)
}

func toPortalSiteWebOptions(descriptors []*skeldesc.Web) []PortalSiteWebOption {
	options := make([]PortalSiteWebOption, 0, len(descriptors))
	for _, descriptor := range descriptors {
		options = append(options, PortalSiteWebOption{
			Name:           descriptor.Name,
			SkelName:       descriptor.SkelName,
			ActorSkelNames: actorSkelNames(descriptor.Audiences),
		})
	}
	return vslice.SortBy(options, func(a PortalSiteWebOption, b PortalSiteWebOption) bool {
		return cmpString(a.SkelName, b.SkelName) < 0
	})
}

func actorSkelNames(refs []*skeldesc.ActorAudience) []string {
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

func (s *PortalSite) normalizeAndValidate() {
	fail := func(ok bool, message string) {
		ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal site %q: %s", s.Name, message))
	}
	fail(strings.TrimSpace(s.Name) != "", "name is required")
	fail(s.Type == PortalSiteTypeRPCGW || s.Type == PortalSiteTypeWEBGW, "type must be RPCGW or WEBGW")
	fail(strings.TrimSpace(s.ActorSkelName) != "", "actorSkelName is required")
	fail(s.ActorVia == string(skeldesc.ActorViaClient) || s.ActorVia == string(skeldesc.ActorViaAgent) || s.ActorVia == string(skeldesc.ActorViaOpenAPI), "actorVia must be client, agent or openapi")
	if s.Type == PortalSiteTypeWEBGW {
		fail(strings.TrimSpace(s.WebName) != "", "webName is required for WEBGW")
	}
	s.Cors = NormalizePortalCors(s.Cors)
	fail(s.Cors.Mode == PortalCorsModeDisabled || s.Cors.Mode == PortalCorsModeSameDomain || s.Cors.Mode == PortalCorsModeStrict, "unsupported CORS mode")
	for _, origin := range s.Cors.AllowedOrigins {
		parsed, err := url.Parse(origin)
		fail(err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil && parsed.Path == "" && parsed.RawQuery == "" && !parsed.ForceQuery && parsed.Fragment == "", "CORS origins must be HTTP(S) origins without a path, query or fragment")
	}
}

// Validate checks and normalizes a user site without accessing storage.
func (*PortalSiteCore) Validate(site PortalSite) PortalSite {
	site.normalizeAndValidate()
	return site
}

// Save creates or replaces a user site by name, preserving an existing ID.
func (m *PortalSiteCore) Save(site PortalSite) *PortalSite {
	site = m.Validate(site)
	site.Id = 0
	if current, ok := m.PortalSiteRepo.GetByName(site.Name); ok {
		site.Id = current.Id
	}
	m.PortalSiteRepo.Save(&site)
	return &site
}
