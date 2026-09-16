package admin

import (
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalSiteApiServiceServerImpl struct {
	skeled.DefaultPortalSiteApiServiceServer

	PortalSiteCore *core.PortalSiteCore `inject:""`
}

func (s *PortalSiteApiServiceServerImpl) List() []skeled.PortalSiteListItem {
	entries := s.PortalSiteCore.List()
	ret := make([]skeled.PortalSiteListItem, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, toServerPortalSiteListItem(entry))
	}
	return ret
}

func (s *PortalSiteApiServiceServerImpl) ListOptions() skeled.PortalSiteOptions {
	return toServerPortalSiteOptions(s.PortalSiteCore.ListOptions())
}

func (s *PortalSiteApiServiceServerImpl) Get(id int) skeled.PortalSite {
	entry := s.PortalSiteCore.Get(id)
	return toServerPortalSite(entry, toServerFieldSources(entry.FieldSources))
}

func (s *PortalSiteApiServiceServerImpl) Create(creation skeled.PortalSiteCreation) skeled.PortalSite {
	entry := s.PortalSiteCore.Create(core.PortalSiteCreation{
		Name:          creation.Name,
		Type:          core.PortalSiteType(creation.Type),
		ActorSkelName: creation.ActorSkelName,
		ActorVia:      creation.ActorVia,
		Cors:          toCorePortalCors(creation.Cors),
		WebName:       creation.WebName,
		Enabled:       creation.Enabled,
	})
	return toServerPortalSite(entry, toServerFieldSources(entry.FieldSources))
}

func (s *PortalSiteApiServiceServerImpl) Update(id int, update skeled.PortalSiteUpdate) skeled.PortalSite {
	entry := s.PortalSiteCore.Update(id, core.PortalSiteUpdate{
		Name:          update.Name,
		Type:          toCorePortalSiteTypePointer(update.Type),
		ActorSkelName: update.ActorSkelName,
		ActorVia:      update.ActorVia,
		Cors:          toCorePortalCorsPointer(update.Cors),
		WebName:       update.WebName,
		Enabled:       update.Enabled,
	})
	return toServerPortalSite(entry, toServerFieldSources(entry.FieldSources))
}

func (s *PortalSiteApiServiceServerImpl) Remove(id int) {
	s.PortalSiteCore.Remove(id)
}

func toServerPortalSite(entry *core.PortalSite, fieldSources []skeled.FieldSource) skeled.PortalSite {
	return skeled.PortalSite{
		Enabled:       entry.Enabled,
		Id:            entry.Id,
		Name:          entry.Name,
		Type:          skeled.PortalSiteType(entry.Type),
		ActorSkelName: entry.ActorSkelName,
		ActorVia:      entry.ActorVia,
		RpcgwServices: entry.RpcgwServices,
		Cors:          toServerPortalCors(entry.Cors),
		WebName:       entry.WebName,
		WebMountPath:  entry.WebMountPath,
		FieldSources:  fieldSources,
	}
}

// toServerPortalSiteListItem maps a site for list responses, which carry the
// entity values without its seed provenance.
func toServerPortalSiteListItem(entry *core.PortalSite) skeled.PortalSiteListItem {
	detail := toServerPortalSite(entry, nil)
	return skeled.PortalSiteListItem{
		Enabled:       detail.Enabled,
		Id:            detail.Id,
		Name:          detail.Name,
		Type:          detail.Type,
		ActorSkelName: detail.ActorSkelName,
		ActorVia:      detail.ActorVia,
		RpcgwServices: detail.RpcgwServices,
		Cors:          detail.Cors,
		WebName:       detail.WebName,
		WebMountPath:  detail.WebMountPath,
	}
}

func toCorePortalSiteTypePointer(value *skeled.PortalSiteType) *core.PortalSiteType {
	if value == nil {
		return nil
	}
	ret := core.PortalSiteType(*value)
	return &ret
}

func toCorePortalCors(value *skeled.PortalCors) core.PortalCors {
	if value == nil {
		return core.PortalCors{}
	}
	return core.PortalCors{
		Mode:           toCorePortalCorsMode(value.Mode),
		AllowedOrigins: append([]string{}, value.AllowedOrigins...),
	}
}

func toCorePortalCorsPointer(value *skeled.PortalCors) *core.PortalCors {
	if value == nil {
		return nil
	}
	ret := toCorePortalCors(value)
	return &ret
}

func toServerPortalCors(value core.PortalCors) *skeled.PortalCors {
	return &skeled.PortalCors{
		Mode:           toServerPortalCorsMode(value.Mode),
		AllowedOrigins: append([]string{}, value.AllowedOrigins...),
	}
}

func toCorePortalCorsMode(value skeled.PortalCorsMode) core.PortalCorsMode {
	switch value {
	case skeled.PortalCorsModeDisabled:
		return core.PortalCorsModeDisabled
	case skeled.PortalCorsModeSameDomain:
		return core.PortalCorsModeSameDomain
	case skeled.PortalCorsModeStrict:
		return core.PortalCorsModeStrict
	default:
		return core.PortalCorsMode(value)
	}
}

func toServerPortalCorsMode(value core.PortalCorsMode) skeled.PortalCorsMode {
	switch value {
	case core.PortalCorsModeDisabled:
		return skeled.PortalCorsModeDisabled
	case core.PortalCorsModeSameDomain:
		return skeled.PortalCorsModeSameDomain
	case core.PortalCorsModeStrict:
		return skeled.PortalCorsModeStrict
	default:
		return skeled.PortalCorsMode(value)
	}
}

func toServerPortalSiteOptions(options core.PortalSiteOptions) skeled.PortalSiteOptions {
	return skeled.PortalSiteOptions{
		Actors:   toServerPortalSiteActorOptions(options.Actors),
		Services: toServerPortalSiteServiceOptions(options.Services),
		Webs:     toServerPortalSiteWebOptions(options.Webs),
	}
}

func toServerPortalSiteActorOptions(options []core.PortalSiteActorOption) []skeled.PortalSiteActorOption {
	ret := make([]skeled.PortalSiteActorOption, 0, len(options))
	for _, option := range options {
		ret = append(ret, skeled.PortalSiteActorOption{
			Name:      option.Name,
			SkelName:  option.SkelName,
			ActorVias: option.ActorVias,
		})
	}
	return ret
}

func toServerPortalSiteServiceOptions(options []core.PortalSiteServiceOption) []skeled.PortalSiteServiceOption {
	ret := make([]skeled.PortalSiteServiceOption, 0, len(options))
	for _, option := range options {
		ret = append(ret, skeled.PortalSiteServiceOption{
			Name:           option.Name,
			SkelName:       option.SkelName,
			ActorSkelNames: option.ActorSkelNames,
		})
	}
	return ret
}

func toServerPortalSiteWebOptions(options []core.PortalSiteWebOption) []skeled.PortalSiteWebOption {
	ret := make([]skeled.PortalSiteWebOption, 0, len(options))
	for _, option := range options {
		ret = append(ret, skeled.PortalSiteWebOption{
			Name:           option.Name,
			SkelName:       option.SkelName,
			ActorSkelNames: option.ActorSkelNames,
		})
	}
	return ret
}
