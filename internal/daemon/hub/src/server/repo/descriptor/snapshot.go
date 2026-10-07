package descriptor

import (
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// _DescriptorSet is the per-kind view of the descriptor snapshot: the descriptors selected
// for serving, every registered version, and a name index for lookups.
type _DescriptorSet[T any] struct {
	selected []T
	versions []core.DescriptorVersion[T]
	byName   map[string]T
}

func (s _DescriptorSet[T]) Selected() []T {
	return append([]T{}, s.selected...)
}

func (s _DescriptorSet[T]) Versions() []core.DescriptorVersion[T] {
	return append([]core.DescriptorVersion[T]{}, s.versions...)
}

func (s _DescriptorSet[T]) Get(skelName string) T {
	return s.byName[skelName]
}

// _DomainDescriptorSet is the domain-level view of the snapshot: the domain
// versions the Hub serves and the per-domain breakdown readers publish.
type _DomainDescriptorSet struct {
	views        []core.DomainDescriptorView
	vineHubViews []core.DomainDescriptorView
}

func (s _DomainDescriptorSet) Views() []core.DomainDescriptorView {
	return append([]core.DomainDescriptorView{}, s.views...)
}

func (s _DomainDescriptorSet) VineHubViews() []core.DomainDescriptorView {
	return append([]core.DomainDescriptorView{}, s.vineHubViews...)
}

// _DescriptorSnapshot is what the descriptor repository serves reads from. It is
// rebuilt whenever the registered descriptors change.
type _DescriptorSnapshot struct {
	Domains   _DomainDescriptorSet
	Actors    _DescriptorSet[*skeldesc.Actor]
	Configs   _DescriptorSet[*skeldesc.Config]
	Data      _DescriptorSet[*skeldesc.Data]
	Enums     _DescriptorSet[*skeldesc.Enum]
	Events    _DescriptorSet[*skeldesc.Event]
	Resources _DescriptorSet[*skeldesc.Resource]
	Services  _DescriptorSet[*skeldesc.Service]
	Tasks     _DescriptorSet[*skeldesc.Task]
	Webs      _DescriptorSet[*skeldesc.Web]
}

// _DescriptorKindSpec describes one descriptor kind: where its declarations live in a
// domain descriptor, which of them the Hub serves, and how they are identified.
type _DescriptorKindSpec[T any] struct {
	refs     func(descriptor *skeldesc.Domain) []_DescriptorRef[T]
	selected func(descriptor *skeldesc.Domain) []T
	skelName func(item T) string
}

// simpleDescriptorKind builds the spec of a kind whose declarations all live in one
// domain descriptor field, optionally hiding Vine's own declarations.
func simpleDescriptorKind[T any](
	extract func(descriptor *skeldesc.Domain) []T,
	skelName func(item T) string,
	hash func(item T) string,
	serveVine bool,
) _DescriptorKindSpec[T] {
	return _DescriptorKindSpec[T]{
		refs: func(descriptor *skeldesc.Domain) []_DescriptorRef[T] {
			return descriptorRefs(extract(descriptor), skelName, hash)
		},
		selected: func(descriptor *skeldesc.Domain) []T {
			items := extract(descriptor)
			if !serveVine {
				items = filterNonVineDescriptors(items, skelName)
			}
			return items
		},
		skelName: skelName,
	}
}

// refreshSnapshot rebuilds the snapshot the repository serves. Callers hold the
// repository lock.
func (r *DescriptorRepo) refreshSnapshot() {
	r.snapshot = buildSnapshot(r.buildDomainDescriptorVersions())
}

func buildSnapshot(domainVersions []core.DomainDescriptorVersion) _DescriptorSnapshot {
	mainDescriptors := make([]*skeldesc.Domain, 0, len(domainVersions))
	for _, version := range domainVersions {
		if version.Main {
			mainDescriptors = append(mainDescriptors, version.Descriptor)
		}
	}
	return _DescriptorSnapshot{
		Domains:   buildDomainDescriptorSet(domainVersions),
		Actors:    buildDescriptorSet(domainVersions, mainDescriptors, _actorDescriptorKind),
		Configs:   buildDescriptorSet(domainVersions, mainDescriptors, _configDescriptorKind),
		Data:      buildDescriptorSet(domainVersions, mainDescriptors, _dataDescriptorKind),
		Enums:     buildDescriptorSet(domainVersions, mainDescriptors, _enumDescriptorKind),
		Events:    buildDescriptorSet(domainVersions, mainDescriptors, _eventDescriptorKind),
		Resources: buildDescriptorSet(domainVersions, mainDescriptors, _resourceDescriptorKind),
		Services:  buildDescriptorSet(domainVersions, mainDescriptors, _serviceDescriptorKind),
		Tasks:     buildDescriptorSet(domainVersions, mainDescriptors, _taskDescriptorKind),
		Webs:      buildDescriptorSet(domainVersions, mainDescriptors, _webDescriptorKind),
	}
}

func buildDomainDescriptorSet(domainVersions []core.DomainDescriptorVersion) _DomainDescriptorSet {
	return _DomainDescriptorSet{
		views:        buildDomainDescriptorViews(domainVersions),
		vineHubViews: buildVineHubDomainDescriptorViews(domainVersions),
	}
}

func buildDescriptorSet[T any](
	domainVersions []core.DomainDescriptorVersion,
	mainDescriptors []*skeldesc.Domain,
	spec _DescriptorKindSpec[T],
) _DescriptorSet[T] {
	selected := make([]T, 0)
	for _, descriptor := range mainDescriptors {
		selected = append(selected, spec.selected(descriptor)...)
	}
	selected = sortedDescriptorsBySkelName(selected, spec.skelName)
	byName := make(map[string]T, len(selected))
	for _, item := range selected {
		byName[spec.skelName(item)] = item
	}
	return _DescriptorSet[T]{
		selected: selected,
		versions: buildDescriptorVersions(domainVersions, spec.refs),
		byName:   byName,
	}
}

var (
	_actorDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Actor { return descriptor.Actors },
		func(item *skeldesc.Actor) string { return item.SkelName },
		func(item *skeldesc.Actor) string { return item.Hash },
		false,
	)
	_configDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Config { return descriptor.Configs },
		func(item *skeldesc.Config) string { return item.SkelName },
		func(item *skeldesc.Config) string { return item.Hash },
		true,
	)
	_enumDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Enum { return descriptor.Enums },
		func(item *skeldesc.Enum) string { return item.SkelName },
		func(item *skeldesc.Enum) string { return item.Hash },
		true,
	)
	_eventDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Event { return descriptor.Events },
		func(item *skeldesc.Event) string { return item.SkelName },
		func(item *skeldesc.Event) string { return item.Hash },
		true,
	)
	_resourceDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Resource { return descriptor.Resources },
		func(item *skeldesc.Resource) string { return item.SkelName },
		func(item *skeldesc.Resource) string { return item.Hash },
		false,
	)
	_taskDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Task { return descriptor.Tasks },
		func(item *skeldesc.Task) string { return item.SkelName },
		func(item *skeldesc.Task) string { return item.Hash },
		true,
	)
	_webDescriptorKind = simpleDescriptorKind(
		func(descriptor *skeldesc.Domain) []*skeldesc.Web { return descriptor.Webs },
		func(item *skeldesc.Web) string { return item.SkelName },
		func(item *skeldesc.Web) string { return item.Hash },
		false,
	)
	_dataDescriptorKind = _DescriptorKindSpec[*skeldesc.Data]{
		refs:     dataDescriptorRefs,
		selected: func(descriptor *skeldesc.Domain) []*skeldesc.Data { return descriptor.Data },
		skelName: func(item *skeldesc.Data) string { return item.SkelName },
	}
	_serviceDescriptorKind = _DescriptorKindSpec[*skeldesc.Service]{
		refs: serviceDescriptorRefs,
		selected: func(descriptor *skeldesc.Domain) []*skeldesc.Service {
			return filterNonVineDescriptors(descriptor.Services, func(item *skeldesc.Service) string { return item.SkelName })
		},
		skelName: func(item *skeldesc.Service) string { return item.SkelName },
	}
)

func filterNonVineDescriptors[T any](descriptors []T, skelNameOf func(T) string) []T {
	ret := make([]T, 0)
	for _, descriptor := range descriptors {
		if !isVineDescriptorRef(skelNameOf(descriptor)) {
			ret = append(ret, descriptor)
		}
	}
	return ret
}

func buildDomainDescriptorViews(
	domainVersions []core.DomainDescriptorVersion,
) []core.DomainDescriptorView {
	return buildDomainDescriptorViewsWithFilter(domainVersions, func(skelName string) bool {
		return !isVineDescriptorRef(skelName)
	})
}

func buildVineHubDomainDescriptorViews(
	domainVersions []core.DomainDescriptorVersion,
) []core.DomainDescriptorView {
	views := buildDomainDescriptorViewsWithFilter(domainVersions, isVineHubDescriptorRef)
	ret := make([]core.DomainDescriptorView, 0, len(views))
	for _, view := range views {
		if isVineHubDomain(view.DomainVersion.Descriptor.Name) {
			ret = append(ret, view)
		}
	}
	return ret
}

func buildDomainDescriptorViewsWithFilter(
	domainVersions []core.DomainDescriptorVersion,
	include func(skelName string) bool,
) []core.DomainDescriptorView {
	actorStates := descriptorVersionStates(domainVersions, actorDescriptorRefs, include)
	configStates := descriptorVersionStates(domainVersions, configDescriptorRefs, include)
	dataStates := descriptorVersionStates(domainVersions, dataDescriptorRefs, include)
	enumStates := descriptorVersionStates(domainVersions, enumDescriptorRefs, include)
	eventStates := descriptorVersionStates(domainVersions, eventDescriptorRefs, include)
	resourceStates := descriptorVersionStates(domainVersions, resourceDescriptorRefs, include)
	serviceStates := descriptorVersionStates(domainVersions, serviceDescriptorRefs, include)
	taskStates := descriptorVersionStates(domainVersions, taskDescriptorRefs, include)
	webStates := descriptorVersionStates(domainVersions, webDescriptorRefs, include)
	views := make([]core.DomainDescriptorView, 0, len(domainVersions))
	for _, domainVersion := range domainVersions {
		views = append(views, core.DomainDescriptorView{
			DomainVersion: domainVersion,
			Actors:        buildDomainDescriptorItemVersions(domainVersion, actorStates, actorDescriptorRefs, include),
			Configs:       buildDomainDescriptorItemVersions(domainVersion, configStates, configDescriptorRefs, include),
			Data:          buildDomainDescriptorItemVersions(domainVersion, dataStates, dataDescriptorRefs, include),
			Enums:         buildDomainDescriptorItemVersions(domainVersion, enumStates, enumDescriptorRefs, include),
			Events:        buildDomainDescriptorItemVersions(domainVersion, eventStates, eventDescriptorRefs, include),
			Resources:     buildDomainDescriptorItemVersions(domainVersion, resourceStates, resourceDescriptorRefs, include),
			Services:      buildDomainDescriptorItemVersions(domainVersion, serviceStates, serviceDescriptorRefs, include),
			Tasks:         buildDomainDescriptorItemVersions(domainVersion, taskStates, taskDescriptorRefs, include),
			Webs:          buildDomainDescriptorItemVersions(domainVersion, webStates, webDescriptorRefs, include),
		})
	}
	return views
}
