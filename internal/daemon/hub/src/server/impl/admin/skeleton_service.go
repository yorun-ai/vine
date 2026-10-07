package admin

import (
	"cmp"
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type SkeletonApiServiceServerImpl struct {
	skeled.DefaultSkeletonApiServiceServer

	DescriptorRepo core.DescriptorRepo `inject:""`
}

func (s *SkeletonApiServiceServerImpl) ListDomains() []skeled.SkeletonDomain {
	views := s.DescriptorRepo.ListDomainDescriptorViews()
	serviceVersionsByKey := skeletonDescriptorVersionsByKey(s.DescriptorRepo.ListServiceDescriptorVersions())
	webVersionsByKey := skeletonDescriptorVersionsByKey(s.DescriptorRepo.ListWebDescriptorVersions())
	ret := make([]skeled.SkeletonDomain, 0, len(views))
	for _, view := range views {
		domain := toServerSkeletonDomain(view, serviceVersionsByKey, webVersionsByKey)
		if domain.Total == 0 {
			continue
		}
		ret = append(ret, domain)
	}
	return sortedSkeletonDomains(ret)
}

func (s *SkeletonApiServiceServerImpl) ListActors() []skeled.SkeletonActorItem {
	views := s.DescriptorRepo.ListDomainDescriptorViews()
	serviceVersionsByKey := skeletonDescriptorVersionsByKey(s.DescriptorRepo.ListServiceDescriptorVersions())
	webVersionsByKey := skeletonDescriptorVersionsByKey(s.DescriptorRepo.ListWebDescriptorVersions())
	ret := make([]skeled.SkeletonActorItem, 0)
	for _, version := range s.DescriptorRepo.ListActorDescriptorVersions() {
		actor := toServerSkeletonActorItem(toSkeletonVersionFields(version), version.Descriptor)
		fillServerSkeletonActorItemAccess(&actor, views, serviceVersionsByKey, webVersionsByKey)
		ret = append(ret, actor)
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListConfigs() []skeled.SkeletonConfigItem {
	ret := make([]skeled.SkeletonConfigItem, 0)
	for _, version := range s.DescriptorRepo.ListConfigDescriptorVersions() {
		ret = append(ret, toServerSkeletonConfigItem(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListServices() []skeled.SkeletonServiceItem {
	ret := make([]skeled.SkeletonServiceItem, 0)
	for _, version := range s.DescriptorRepo.ListServiceDescriptorVersions() {
		ret = append(ret, toServerSkeletonServiceItem(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListResources() []skeled.SkeletonResourceItem {
	ret := make([]skeled.SkeletonResourceItem, 0)
	for _, version := range s.DescriptorRepo.ListResourceDescriptorVersions() {
		ret = append(ret, toServerSkeletonResourceItem(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListWebs() []skeled.SkeletonWebItem {
	ret := make([]skeled.SkeletonWebItem, 0)
	for _, version := range s.DescriptorRepo.ListWebDescriptorVersions() {
		ret = append(ret, toServerSkeletonWebItem(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListTasks() []skeled.SkeletonTask {
	ret := make([]skeled.SkeletonTask, 0)
	for _, version := range s.DescriptorRepo.ListTaskDescriptorVersions() {
		ret = append(ret, toServerSkeletonTask(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListEvents() []skeled.SkeletonEventItem {
	ret := make([]skeled.SkeletonEventItem, 0)
	for _, version := range s.DescriptorRepo.ListEventDescriptorVersions() {
		ret = append(ret, toServerSkeletonEventItem(toSkeletonVersionFields(version), version.Descriptor))
	}
	return ret
}

func (s *SkeletonApiServiceServerImpl) ListData() []skeled.SkeletonData {
	ret := make([]skeled.SkeletonData, 0)
	for _, version := range s.DescriptorRepo.ListDataDescriptorVersions() {
		ret = append(ret, toServerSkeletonData(toSkeletonVersionFields(version), version.Descriptor))
	}
	for _, version := range s.DescriptorRepo.ListEnumDescriptorVersions() {
		ret = append(ret, toServerSkeletonEnumData(toSkeletonVersionFields(version), version.Descriptor))
	}
	return sortedSkeletonData(ret)
}

func sortedSkeletonActorItems(items []skeled.SkeletonActorItem) []skeled.SkeletonActorItem {
	return vslice.SortBy(items, func(a skeled.SkeletonActorItem, b skeled.SkeletonActorItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonConfigItems(items []skeled.SkeletonConfigItem) []skeled.SkeletonConfigItem {
	return vslice.SortBy(items, func(a skeled.SkeletonConfigItem, b skeled.SkeletonConfigItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonServiceItems(items []skeled.SkeletonServiceItem) []skeled.SkeletonServiceItem {
	return vslice.SortBy(items, func(a skeled.SkeletonServiceItem, b skeled.SkeletonServiceItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonResourceItems(items []skeled.SkeletonResourceItem) []skeled.SkeletonResourceItem {
	return vslice.SortBy(items, func(a skeled.SkeletonResourceItem, b skeled.SkeletonResourceItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonWebItems(items []skeled.SkeletonWebItem) []skeled.SkeletonWebItem {
	return vslice.SortBy(items, func(a skeled.SkeletonWebItem, b skeled.SkeletonWebItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonTasks(items []skeled.SkeletonTask) []skeled.SkeletonTask {
	return vslice.SortBy(items, func(a skeled.SkeletonTask, b skeled.SkeletonTask) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonEventItems(items []skeled.SkeletonEventItem) []skeled.SkeletonEventItem {
	return vslice.SortBy(items, func(a skeled.SkeletonEventItem, b skeled.SkeletonEventItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonData(items []skeled.SkeletonData) []skeled.SkeletonData {
	return vslice.SortBy(items, func(a skeled.SkeletonData, b skeled.SkeletonData) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.DescriptorHash, b.SkelName, b.IsMain, b.DescriptorHash) < 0
	})
}

func sortedSkeletonDomains(items []skeled.SkeletonDomain) []skeled.SkeletonDomain {
	return vslice.SortBy(items, func(a skeled.SkeletonDomain, b skeled.SkeletonDomain) bool {
		if a.Domain != b.Domain {
			return cmp.Compare(a.Domain, b.Domain) < 0
		}
		if a.IsMain != b.IsMain {
			return a.IsMain
		}
		return cmp.Compare(b.DescriptorHash, a.DescriptorHash) < 0
	})
}

func compareSkeletonItemVersion(aName string, aIsMain bool, aHash string, bName string, bIsMain bool, bHash string) int {
	if order := cmp.Compare(aName, bName); order != 0 {
		return order
	}
	if aIsMain != bIsMain {
		if aIsMain {
			return -1
		}
		return 1
	}
	return cmp.Compare(bHash, aHash)
}

type _SkeletonVersionFields struct {
	Domain               string
	DescriptorHash       string
	MainDescriptorHash   string
	IsMultiVersion       bool
	IsMain               bool
	DomainDescriptorHash string
}

func toSkeletonVersionFields[T any](version core.DescriptorVersion[T]) _SkeletonVersionFields {
	return _SkeletonVersionFields{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.MultiVersion,
		IsMain:               version.Main,
		DomainDescriptorHash: version.DomainDescriptorHash,
	}
}

func toServerSkeletonDomain(
	view core.DomainDescriptorView,
	serviceVersionsByKey map[string]core.DescriptorVersion[*skeldesc.Service],
	webVersionsByKey map[string]core.DescriptorVersion[*skeldesc.Web],
) skeled.SkeletonDomain {
	version := view.DomainVersion
	domain := skeled.SkeletonDomain{
		Domain:             version.Descriptor.Name,
		DescriptorHash:     version.Descriptor.Hash,
		MainDescriptorHash: version.MainDescriptorHash,
		IsMultiVersion:     version.MultiVersion,
		IsMain:             version.Main,
		Actors:             make([]skeled.SkeletonActorItem, 0, len(view.Actors)),
		Configs:            make([]skeled.SkeletonConfigItem, 0, len(view.Configs)),
		Services:           make([]skeled.SkeletonServiceItem, 0, len(view.Services)),
		Resources:          make([]skeled.SkeletonResourceItem, 0, len(view.Resources)),
		Data:               make([]skeled.SkeletonData, 0, len(view.Data)+len(view.Enums)),
		Webs:               make([]skeled.SkeletonWebItem, 0, len(view.Webs)),
		Tasks:              make([]skeled.SkeletonTask, 0, len(view.Tasks)),
		Events:             make([]skeled.SkeletonEventItem, 0, len(view.Events)),
	}
	for _, item := range view.Actors {
		actor := toServerSkeletonActorItem(toSkeletonVersionFields(item), item.Descriptor)
		fillServerSkeletonActorItemAccess(&actor, []core.DomainDescriptorView{view}, serviceVersionsByKey, webVersionsByKey)
		domain.Actors = append(domain.Actors, actor)
	}
	for _, item := range view.Configs {
		domain.Configs = append(domain.Configs, toServerSkeletonConfigItem(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Services {
		domain.Services = append(domain.Services, toServerSkeletonServiceItem(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Resources {
		domain.Resources = append(domain.Resources, toServerSkeletonResourceItem(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Data {
		domain.Data = append(domain.Data, toServerSkeletonData(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Enums {
		domain.Data = append(domain.Data, toServerSkeletonEnumData(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Webs {
		domain.Webs = append(domain.Webs, toServerSkeletonWebItem(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Tasks {
		domain.Tasks = append(domain.Tasks, toServerSkeletonTask(toSkeletonVersionFields(item), item.Descriptor))
	}
	for _, item := range view.Events {
		domain.Events = append(domain.Events, toServerSkeletonEventItem(toSkeletonVersionFields(item), item.Descriptor))
	}
	domain.Actors = sortedSkeletonActorItems(domain.Actors)
	domain.Configs = sortedSkeletonConfigItems(domain.Configs)
	domain.Services = sortedSkeletonServiceItems(domain.Services)
	domain.Resources = sortedSkeletonResourceItems(domain.Resources)
	domain.Data = sortedSkeletonData(domain.Data)
	domain.Webs = sortedSkeletonWebItems(domain.Webs)
	domain.Tasks = sortedSkeletonTasks(domain.Tasks)
	domain.Events = sortedSkeletonEventItems(domain.Events)
	domain.Total = len(domain.Actors) + len(domain.Configs) + len(domain.Services) + len(domain.Resources) + len(domain.Data) + len(domain.Webs) + len(domain.Tasks) + len(domain.Events)
	return domain
}

func toServerSkeletonActorItem(version _SkeletonVersionFields, descriptor *skeldesc.Actor) skeled.SkeletonActorItem {
	vias := make([]string, 0, len(descriptor.Vias))
	for _, via := range descriptor.Vias {
		vias = append(vias, string(via))
	}
	result := skeled.SkeletonActorItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		ActorVias:            vias,
		AuthEnabled:          (descriptor.Auth != nil),
		PermEnabled:          (descriptor.Permission != nil),
		Services:             []skeled.SkeletonServiceItem{},
		Webs:                 []skeled.SkeletonWebItem{},
	}

	if descriptor.Auth != nil {
		result.IdentifierField = descriptor.Auth.IdentifierField
		result.Credential = toServerSkeletonActorData(version, descriptor.Auth.Credential)
		result.Info = toServerSkeletonActorData(version, descriptor.Auth.Info)
		result.AuthService = toServerSkeletonActorService(version, descriptor.Auth.Service)
	}
	if descriptor.Permission != nil {
		result.PermService = toServerSkeletonActorService(version, descriptor.Permission.Service)
		result.PermMethod = toServerSkeletonActorMethod(descriptor.Permission.Method())
	}
	return result
}

func toServerSkeletonActorData(actorVersion _SkeletonVersionFields, descriptor *skeldesc.Data) *skeled.SkeletonData {
	if descriptor == nil {
		return nil
	}
	item := toServerSkeletonData(toSkeletonDerivedVersionFields(actorVersion, descriptor.Hash), descriptor)
	return &item
}

func toServerSkeletonActorService(actorVersion _SkeletonVersionFields, descriptor *skeldesc.Service) *skeled.SkeletonServiceItem {
	if descriptor == nil {
		return nil
	}
	item := toServerSkeletonServiceItem(toSkeletonDerivedVersionFields(actorVersion, descriptor.Hash), descriptor)
	return &item
}

func toServerSkeletonActorMethod(descriptor *skeldesc.Method) *skeled.SkeletonMethod {
	if descriptor == nil {
		return nil
	}
	item := toServerSkeletonMethod(descriptor)
	return &item
}

func toSkeletonDerivedVersionFields(parent _SkeletonVersionFields, descriptorHash string) _SkeletonVersionFields {
	return _SkeletonVersionFields{
		Domain:               parent.Domain,
		DescriptorHash:       descriptorHash,
		MainDescriptorHash:   descriptorHash,
		IsMultiVersion:       false,
		IsMain:               true,
		DomainDescriptorHash: parent.DomainDescriptorHash,
	}
}

func fillServerSkeletonActorItemAccess(
	actor *skeled.SkeletonActorItem,
	views []core.DomainDescriptorView,
	serviceVersionsByKey map[string]core.DescriptorVersion[*skeldesc.Service],
	webVersionsByKey map[string]core.DescriptorVersion[*skeldesc.Web],
) {
	serviceKeys := map[string]struct{}{}
	webKeys := map[string]struct{}{}

	for _, view := range views {
		if !domainDescriptorCanReferenceActorVersion(view.DomainVersion.Descriptor, actor.SkelName, actor.DescriptorHash) {
			continue
		}
		for _, descriptor := range view.DomainVersion.Descriptor.Services {
			if !skeletonActorRefsContain(descriptor.Audiences, actor.SkelName) {
				continue
			}
			key := skeletonDescriptorVersionKey(descriptor.SkelName, descriptor.Hash)
			if _, ok := serviceKeys[key]; ok {
				continue
			}
			version, ok := serviceVersionsByKey[key]
			if !ok {
				continue
			}
			serviceKeys[key] = struct{}{}
			actor.Services = append(actor.Services, toServerSkeletonServiceItem(toSkeletonVersionFields(version), version.Descriptor))
		}
		for _, descriptor := range view.DomainVersion.Descriptor.Webs {
			if !skeletonActorRefsContain(descriptor.Audiences, actor.SkelName) {
				continue
			}
			key := skeletonDescriptorVersionKey(descriptor.SkelName, descriptor.Hash)
			if _, ok := webKeys[key]; ok {
				continue
			}
			version, ok := webVersionsByKey[key]
			if !ok {
				continue
			}
			webKeys[key] = struct{}{}
			actor.Webs = append(actor.Webs, toServerSkeletonWebItem(toSkeletonVersionFields(version), version.Descriptor))
		}
	}

	actor.Services = sortedSkeletonServiceItems(actor.Services)
	actor.Webs = sortedSkeletonWebItems(actor.Webs)
}

func domainDescriptorCanReferenceActorVersion(descriptor *skeldesc.Domain, skelName string, descriptorHash string) bool {
	hasActor := false
	for _, actor := range descriptor.Actors {
		if actor.SkelName != skelName {
			continue
		}
		hasActor = true
		if actor.Hash != descriptorHash {
			continue
		}
		return true
	}
	return !hasActor
}

func skeletonActorRefsContain(refs []*skeldesc.ActorAudience, skelName string) bool {
	for _, ref := range refs {
		if ref.SkelName == skelName {
			return true
		}
	}
	return false
}

func toServerSkeletonServiceItem(version _SkeletonVersionFields, descriptor *skeldesc.Service) skeled.SkeletonServiceItem {
	return skeled.SkeletonServiceItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Pub:                  descriptor.Pub,
		Api:                  descriptor.Api,
		Ext:                  descriptor.Ext,
		AuthMode:             string(descriptor.AuthMode),
		Require:              toServerSkeletonPermExpr(descriptor.Require),
		Actors:               toServerSkeletonActorRefs(descriptor.Audiences),
		Methods:              toServerSkeletonMethods(descriptor.Methods),
	}
}

func toServerSkeletonResourceItem(version _SkeletonVersionFields, descriptor *skeldesc.Resource) skeled.SkeletonResourceItem {
	return skeled.SkeletonResourceItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Checks:               toServerSkeletonResourceChecks(descriptor, descriptor.Checks),
		Actions:              toServerSkeletonResourceActions(descriptor, descriptor.Actions),
		CheckService:         toServerSkeletonResourceCheckService(version, descriptor.CheckService),
	}
}

func toServerSkeletonResourceCheckService(resourceVersion _SkeletonVersionFields, descriptor *skeldesc.Service) *skeled.SkeletonServiceItem {
	if descriptor == nil {
		return nil
	}
	item := toServerSkeletonServiceItem(toSkeletonDerivedVersionFields(resourceVersion, descriptor.Hash), descriptor)
	return &item
}

func toServerSkeletonConfigItem(version _SkeletonVersionFields, descriptor *skeldesc.Config) skeled.SkeletonConfigItem {
	return skeled.SkeletonConfigItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Pub:                  descriptor.Pub,
		Sensitive:            descriptor.Sensitive,
		Lifecycle:            string(descriptor.Lifecycle),
		Fields:               toServerSkeletonFields(descriptor.Members),
	}
}

func toServerSkeletonWebItem(version _SkeletonVersionFields, descriptor *skeldesc.Web) skeled.SkeletonWebItem {
	return skeled.SkeletonWebItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		AuthMode:             string(descriptor.AuthMode),
		Actors:               toServerSkeletonActorRefs(descriptor.Audiences),
	}
}

func toServerSkeletonTask(version _SkeletonVersionFields, descriptor *skeldesc.Task) skeled.SkeletonTask {
	triggers := make([]skeled.SkeletonTrigger, 0, len(descriptor.Triggers))
	for _, trigger := range descriptor.Triggers {
		triggers = append(triggers, skeled.SkeletonTrigger{
			Name:               trigger.Name,
			SkelName:           trigger.SkelName,
			Description:        trigger.Description,
			Deprecated:         trigger.Deprecated,
			DeprecatedReason:   trigger.DeprecatedReason,
			InputDescription:   trigger.InputDescription,
			Arguments:          toServerSkeletonFields(trigger.Arguments),
			ArgumentsSensitive: trigger.ArgumentsSensitive,
		})
	}
	return skeled.SkeletonTask{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Triggers:             triggers,
	}
}

func toServerSkeletonEventItem(version _SkeletonVersionFields, descriptor *skeldesc.Event) skeled.SkeletonEventItem {
	return skeled.SkeletonEventItem{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Pub:                  descriptor.Pub,
		Ext:                  descriptor.Ext,
		Sensitive:            descriptor.Sensitive,
		Fields:               toServerSkeletonFields(descriptor.Members),
	}
}

func toServerSkeletonData(version _SkeletonVersionFields, descriptor *skeldesc.Data) skeled.SkeletonData {
	return skeled.SkeletonData{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Enum:                 false,
		Sensitive:            descriptor.Sensitive,
		TypeParameters:       append([]string{}, descriptor.TypeParameters...),
		Fields:               toServerSkeletonFields(descriptor.Members),
		EnumItems:            []skeled.SkeletonEnumItem{},
	}
}

func toServerSkeletonEnumData(version _SkeletonVersionFields, descriptor *skeldesc.Enum) skeled.SkeletonData {
	return skeled.SkeletonData{
		Domain:               version.Domain,
		DescriptorHash:       version.DescriptorHash,
		MainDescriptorHash:   version.MainDescriptorHash,
		IsMultiVersion:       version.IsMultiVersion,
		IsMain:               version.IsMain,
		DomainDescriptorHash: version.DomainDescriptorHash,
		Name:                 descriptor.Name,
		SkelName:             descriptor.SkelName,
		Description:          descriptor.Description,
		Deprecated:           descriptor.Deprecated,
		DeprecatedReason:     descriptor.DeprecatedReason,
		Enum:                 true,
		TypeParameters:       []string{},
		Fields:               []skeled.SkeletonField{},
		EnumItems:            toServerSkeletonEnumItems(descriptor.Items),
	}
}

func toServerSkeletonEnumItems(descriptors []*skeldesc.EnumItem) []skeled.SkeletonEnumItem {
	ret := make([]skeled.SkeletonEnumItem, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, skeled.SkeletonEnumItem{
			Name:             descriptor.Name,
			Description:      descriptor.Description,
			Deprecated:       descriptor.Deprecated,
			DeprecatedReason: descriptor.DeprecatedReason,
		})
	}
	return ret
}

func toServerSkeletonActorRefs(refs []*skeldesc.ActorAudience) []skeled.SkeletonActorRef {
	ret := make([]skeled.SkeletonActorRef, 0, len(refs))
	for _, ref := range refs {
		ret = append(ret, skeled.SkeletonActorRef{
			Name:     ref.Name,
			SkelName: ref.SkelName,
			Via:      string(ref.Via),
		})
	}
	return ret
}

func toServerSkeletonMethods(descriptors []*skeldesc.Method) []skeled.SkeletonMethod {
	ret := make([]skeled.SkeletonMethod, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, toServerSkeletonMethod(descriptor))
	}
	return ret
}

func toServerSkeletonMethod(descriptor *skeldesc.Method) skeled.SkeletonMethod {
	return skeled.SkeletonMethod{
		Name:               descriptor.Name,
		SkelName:           descriptor.SkelName,
		Description:        descriptor.Description,
		Deprecated:         descriptor.Deprecated,
		DeprecatedReason:   descriptor.DeprecatedReason,
		InputDescription:   descriptor.InputDescription,
		OutputDescription:  descriptor.OutputDescription,
		Example:            descriptor.Example,
		AuthMode:           string(descriptor.AuthMode),
		Require:            toServerSkeletonPermExpr(descriptor.Require),
		OutputExample:      descriptor.OutputExample,
		Arguments:          toServerSkeletonFields(descriptor.Arguments),
		ArgumentsSensitive: descriptor.ArgumentsSensitive,
		ResultType:         formatSkeletonType(descriptor.ResultType),
		ResultSensitive:    descriptor.ResultSensitive,
	}
}

func toServerSkeletonPermExpr(descriptor *skeldesc.PermissionRequire) *skeled.SkeletonPermExpr {
	if descriptor == nil {
		return nil
	}
	return toServerSkeletonPermExprNode(descriptor.Expression)
}

func toServerSkeletonPermExprNode(descriptor *skeldesc.PermissionExpression) *skeled.SkeletonPermExpr {
	if descriptor == nil {
		return nil
	}
	children := make([]skeled.SkeletonPermExpr, 0, len(descriptor.Children))
	for _, child := range descriptor.Children {
		childExpr := toServerSkeletonPermExprNode(child)
		if childExpr == nil {
			continue
		}
		children = append(children, *childExpr)
	}
	return &skeled.SkeletonPermExpr{
		Mode:     string(descriptor.Mode),
		Code:     descriptor.Code,
		Check:    toServerSkeletonPermCheck(descriptor.Check),
		Children: children,
	}
}

func toServerSkeletonPermCheck(descriptor *skeldesc.PermissionCheckInvocation) *skeled.SkeletonPermCheck {
	if descriptor == nil {
		return nil
	}
	return &skeled.SkeletonPermCheck{
		ResourceSkelName: descriptor.ResourceSkelName,
		ActionName:       descriptor.ActionName,
		CheckName:        descriptor.CheckName,
		ServiceSkelName:  descriptor.ServiceSkelName,
		MethodSkelName:   descriptor.MethodSkelName,
		Arguments:        toServerSkeletonPermCheckArguments(descriptor.Arguments),
	}
}

func toServerSkeletonPermCheckArguments(descriptors []*skeldesc.PermissionCheckArgument) []skeled.SkeletonPermCheckArgument {
	ret := make([]skeled.SkeletonPermCheckArgument, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, skeled.SkeletonPermCheckArgument{
			Name:     descriptor.Name,
			JsonPath: descriptor.JsonPath,
			Type:     formatSkeletonType(descriptor.Type),
		})
	}
	return ret
}

func toServerSkeletonResourceActions(resource *skeldesc.Resource, descriptors []*skeldesc.ResourceAction) []skeled.SkeletonResourceAction {
	ret := make([]skeled.SkeletonResourceAction, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, skeled.SkeletonResourceAction{
			Name:             descriptor.Name,
			PermissionCode:   descriptor.PermissionCode,
			Description:      descriptor.Description,
			Deprecated:       descriptor.Deprecated,
			DeprecatedReason: descriptor.DeprecatedReason,
			Checks:           toServerSkeletonResourceChecks(resource, descriptor.Checks),
		})
	}
	return ret
}

func toServerSkeletonResourceChecks(resource *skeldesc.Resource, descriptors []*skeldesc.ResourceCheck) []skeled.SkeletonResourceCheck {
	ret := make([]skeled.SkeletonResourceCheck, 0, len(descriptors))
	for _, descriptor := range descriptors {
		method := resource.CheckMethod(descriptor)
		ret = append(ret, skeled.SkeletonResourceCheck{
			Name:               descriptor.Name,
			Deprecated:         descriptor.Deprecated,
			DeprecatedReason:   descriptor.DeprecatedReason,
			MethodName:         method.Name,
			MethodSkelName:     method.SkelName,
			Arguments:          toServerSkeletonFields(method.Arguments),
			ArgumentsSensitive: method.ArgumentsSensitive,
		})
	}
	return ret
}

func toServerSkeletonFields(descriptors []*skeldesc.Member) []skeled.SkeletonField {
	ret := make([]skeled.SkeletonField, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ret = append(ret, skeled.SkeletonField{
			Name:             descriptor.Name,
			Type:             formatSkeletonType(descriptor.Type),
			Description:      descriptor.Description,
			Deprecated:       descriptor.Deprecated,
			DeprecatedReason: descriptor.DeprecatedReason,
			Example:          descriptor.Example,
			Sensitive:        descriptor.Sensitive,
		})
	}
	return ret
}

func formatSkeletonType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor == nil {
		return ""
	}
	var ret string
	switch typeDescriptor.Kind {
	case skeldesc.TypeKindScalar:
		ret = string(typeDescriptor.Scalar)
	case skeldesc.TypeKindEnum, skeldesc.TypeKindData, skeldesc.TypeKindConfig, skeldesc.TypeKindEvent:
		ret = formatSkeletonNamedType(typeDescriptor)
		if len(typeDescriptor.TypeArguments) > 0 {
			args := make([]string, 0, len(typeDescriptor.TypeArguments))
			for _, arg := range typeDescriptor.TypeArguments {
				args = append(args, formatSkeletonType(arg))
			}
			ret += "<" + strings.Join(args, ", ") + ">"
		}
	case skeldesc.TypeKindTypeParameter:
		ret = typeDescriptor.Name
		if ret == "" {
			ret = shortSkelName(typeDescriptor.SkelName)
		}
	case skeldesc.TypeKindList:
		ret = "list<" + formatSkeletonType(typeDescriptor.Element) + ">"
	case skeldesc.TypeKindMap:
		ret = "map<" + formatSkeletonType(typeDescriptor.Key) + ", " + formatSkeletonType(typeDescriptor.Value) + ">"
	default:
		ret = string(typeDescriptor.Kind)
	}
	if typeDescriptor.Nullable {
		ret += "?"
	}
	return ret
}

func formatSkeletonNamedType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor.SkelName != "" {
		return typeDescriptor.SkelName
	}
	if typeDescriptor.Name != "" {
		return typeDescriptor.Name
	}
	return shortSkelName(typeDescriptor.SkelName)
}

func shortSkelName(skelName string) string {
	if _, shortName, ok := strings.CutLast(skelName, "."); ok {
		return shortName
	}
	return skelName
}

func skeletonDescriptorVersionKey(skelName string, descriptorHash string) string {
	return skelName + "\x00" + descriptorHash
}

func skeletonDescriptorVersionsByKey[T any](versions []core.DescriptorVersion[T]) map[string]core.DescriptorVersion[T] {
	ret := make(map[string]core.DescriptorVersion[T], len(versions))
	for _, version := range versions {
		ret[skeletonDescriptorVersionKey(version.SkelName, version.DescriptorHash)] = version
	}
	return ret
}
