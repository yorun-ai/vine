package impl

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalSiteServiceServerImpl struct {
	skeled.DefaultPortalSiteServiceServer

	PortalSiteCore *core.PortalSiteCore `inject:""`
}

func (s *PortalSiteServiceServerImpl) List() []skeled.PortalSite {
	entries := s.PortalSiteCore.List()
	ret := make([]skeled.PortalSite, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, s.toServerPortalSite(entry))
	}
	return ret
}

func (s *PortalSiteServiceServerImpl) ListOptions() skeled.PortalSiteOptions {
	return toServerPortalSiteOptions(s.PortalSiteCore.ListOptions())
}

func (s *PortalSiteServiceServerImpl) Get(id int) skeled.PortalSite {
	return s.toServerPortalSite(s.PortalSiteCore.Get(id))
}

func (s *PortalSiteServiceServerImpl) Create(creation skeled.PortalSiteCreation) skeled.PortalSite {
	return s.toServerPortalSite(s.PortalSiteCore.Create(core.PortalSiteCreation{
		Name:          creation.Name,
		Type:          core.PortalSiteType(creation.Type),
		ActorSkelName: creation.ActorSkelName,
		ActorVia:      creation.ActorVia,
		Cors:          toCorePortalCors(creation.Cors),
		WebName:       creation.WebName,
	}))
}

func (s *PortalSiteServiceServerImpl) Update(id int, update skeled.PortalSiteUpdate) skeled.PortalSite {
	return s.toServerPortalSite(s.PortalSiteCore.Update(id, core.PortalSiteUpdate{
		Name:          update.Name,
		Type:          toCorePortalSiteTypePointer(update.Type),
		ActorSkelName: update.ActorSkelName,
		ActorVia:      update.ActorVia,
		Cors:          toCorePortalCorsPointer(update.Cors),
		WebName:       update.WebName,
	}))
}

func (s *PortalSiteServiceServerImpl) Remove(id int) {
	s.PortalSiteCore.Remove(id)
}

func (s *PortalSiteServiceServerImpl) toServerPortalSite(entry core.PortalSite) skeled.PortalSite {
	return toServerPortalSite(entry, s.PortalSiteCore.RpcgwServices(entry))
}

func toServerPortalSite(entry core.PortalSite, rpcgwServices []string) skeled.PortalSite {
	return skeled.PortalSite{
		Id:            entry.Id,
		Name:          entry.Name,
		Type:          skeled.PortalSiteType(entry.Type),
		ActorSkelName: entry.ActorSkelName,
		ActorVia:      entry.ActorVia,
		RpcgwServices: rpcgwServices,
		Cors:          toServerPortalCors(entry.Cors),
		WebName:       entry.WebName,
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
