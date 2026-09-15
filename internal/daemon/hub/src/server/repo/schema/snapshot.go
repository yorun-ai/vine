package schema

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// _SchemaSet is the per-kind view of the schema snapshot: the schemas selected
// for serving, every registered version, and a name index for lookups.
type _SchemaSet[T any] struct {
	selected []T
	versions []core.SchemaVersion[T]
	byName   map[string]T
}

func (s _SchemaSet[T]) Selected() []T {
	return append([]T{}, s.selected...)
}

func (s _SchemaSet[T]) Versions() []core.SchemaVersion[T] {
	return append([]core.SchemaVersion[T]{}, s.versions...)
}

func (s _SchemaSet[T]) Get(skelName string) T {
	return s.byName[skelName]
}

// _DomainSchemaSet is the domain-level view of the snapshot: the domain
// versions the Hub serves and the per-domain breakdown readers publish.
type _DomainSchemaSet struct {
	views        []core.DomainSchemaView
	vineHubViews []core.DomainSchemaView
}

func (s _DomainSchemaSet) Views() []core.DomainSchemaView {
	return append([]core.DomainSchemaView{}, s.views...)
}

func (s _DomainSchemaSet) VineHubViews() []core.DomainSchemaView {
	return append([]core.DomainSchemaView{}, s.vineHubViews...)
}

// _SchemaSnapshot is what the schema repository serves reads from. It is
// rebuilt whenever the registered schemas change.
type _SchemaSnapshot struct {
	Domains   _DomainSchemaSet
	Actors    _SchemaSet[*skel.ActorSchema]
	Configs   _SchemaSet[*skel.ConfigSchema]
	Data      _SchemaSet[*skel.DataSchema]
	Enums     _SchemaSet[*skel.EnumSchema]
	Events    _SchemaSet[*skel.EventSchema]
	Resources _SchemaSet[*skel.ResourceSchema]
	Services  _SchemaSet[*skel.ServiceSchema]
	Tasks     _SchemaSet[*skel.TaskSchema]
	Webs      _SchemaSet[*skel.WebSchema]
}

// _SchemaKindSpec describes one schema kind: where its declarations live in a
// domain schema, which of them the Hub serves, and how they are identified.
type _SchemaKindSpec[T any] struct {
	refs     func(schema *skel.DomainSchema) []_SchemaRef[T]
	selected func(schema *skel.DomainSchema) []T
	skelName func(item T) string
}

// simpleSchemaKind builds the spec of a kind whose declarations all live in one
// domain schema field, optionally hiding Vine's own declarations.
func simpleSchemaKind[T any](
	extract func(schema *skel.DomainSchema) []T,
	skelName func(item T) string,
	hash func(item T) string,
	serveVine bool,
) _SchemaKindSpec[T] {
	return _SchemaKindSpec[T]{
		refs: func(schema *skel.DomainSchema) []_SchemaRef[T] {
			return schemaRefs(extract(schema), skelName, hash)
		},
		selected: func(schema *skel.DomainSchema) []T {
			items := extract(schema)
			if !serveVine {
				items = filterNonVineSchemas(items, skelName)
			}
			return items
		},
		skelName: skelName,
	}
}

// refreshSnapshot rebuilds the snapshot the repository serves. Callers hold the
// repository lock.
func (r *SchemaRepo) refreshSnapshot() {
	r.snapshot = buildSnapshot(r.buildDomainSchemaVersions())
}

func buildSnapshot(domainVersions []core.DomainSchemaVersion) _SchemaSnapshot {
	mainSchemas := make([]*skel.DomainSchema, 0, len(domainVersions))
	for _, version := range domainVersions {
		if version.Main {
			mainSchemas = append(mainSchemas, version.Schema)
		}
	}
	return _SchemaSnapshot{
		Domains:   buildDomainSchemaSet(domainVersions),
		Actors:    buildSchemaSet(domainVersions, mainSchemas, _actorSchemaKind),
		Configs:   buildSchemaSet(domainVersions, mainSchemas, _configSchemaKind),
		Data:      buildSchemaSet(domainVersions, mainSchemas, _dataSchemaKind),
		Enums:     buildSchemaSet(domainVersions, mainSchemas, _enumSchemaKind),
		Events:    buildSchemaSet(domainVersions, mainSchemas, _eventSchemaKind),
		Resources: buildSchemaSet(domainVersions, mainSchemas, _resourceSchemaKind),
		Services:  buildSchemaSet(domainVersions, mainSchemas, _serviceSchemaKind),
		Tasks:     buildSchemaSet(domainVersions, mainSchemas, _taskSchemaKind),
		Webs:      buildSchemaSet(domainVersions, mainSchemas, _webSchemaKind),
	}
}

func buildDomainSchemaSet(domainVersions []core.DomainSchemaVersion) _DomainSchemaSet {
	return _DomainSchemaSet{
		views:        buildDomainSchemaViews(domainVersions),
		vineHubViews: buildVineHubDomainSchemaViews(domainVersions),
	}
}

func buildSchemaSet[T any](
	domainVersions []core.DomainSchemaVersion,
	mainSchemas []*skel.DomainSchema,
	spec _SchemaKindSpec[T],
) _SchemaSet[T] {
	selected := make([]T, 0)
	for _, schema := range mainSchemas {
		selected = append(selected, spec.selected(schema)...)
	}
	selected = sortedSchemasBySkelName(selected, spec.skelName)
	byName := make(map[string]T, len(selected))
	for _, item := range selected {
		byName[spec.skelName(item)] = item
	}
	return _SchemaSet[T]{
		selected: selected,
		versions: buildSchemaVersions(domainVersions, spec.refs),
		byName:   byName,
	}
}

var (
	_actorSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.ActorSchema { return schema.Actors },
		func(item *skel.ActorSchema) string { return item.SkelName },
		func(item *skel.ActorSchema) string { return item.Hash },
		false,
	)
	_configSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.ConfigSchema { return schema.Configs },
		func(item *skel.ConfigSchema) string { return item.SkelName },
		func(item *skel.ConfigSchema) string { return item.Hash },
		true,
	)
	_enumSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.EnumSchema { return schema.Enums },
		func(item *skel.EnumSchema) string { return item.SkelName },
		func(item *skel.EnumSchema) string { return item.Hash },
		true,
	)
	_eventSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.EventSchema { return schema.Events },
		func(item *skel.EventSchema) string { return item.SkelName },
		func(item *skel.EventSchema) string { return item.Hash },
		true,
	)
	_resourceSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.ResourceSchema { return schema.Resources },
		func(item *skel.ResourceSchema) string { return item.SkelName },
		func(item *skel.ResourceSchema) string { return item.Hash },
		false,
	)
	_taskSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.TaskSchema { return schema.Tasks },
		func(item *skel.TaskSchema) string { return item.SkelName },
		func(item *skel.TaskSchema) string { return item.Hash },
		true,
	)
	_webSchemaKind = simpleSchemaKind(
		func(schema *skel.DomainSchema) []*skel.WebSchema { return schema.Webs },
		func(item *skel.WebSchema) string { return item.SkelName },
		func(item *skel.WebSchema) string { return item.Hash },
		false,
	)
	_dataSchemaKind = _SchemaKindSpec[*skel.DataSchema]{
		refs:     dataSchemaRefs,
		selected: func(schema *skel.DomainSchema) []*skel.DataSchema { return schema.Data },
		skelName: func(item *skel.DataSchema) string { return item.SkelName },
	}
	_serviceSchemaKind = _SchemaKindSpec[*skel.ServiceSchema]{
		refs: serviceSchemaRefs,
		selected: func(schema *skel.DomainSchema) []*skel.ServiceSchema {
			return filterNonVineSchemas(schema.Services, func(item *skel.ServiceSchema) string { return item.SkelName })
		},
		skelName: func(item *skel.ServiceSchema) string { return item.SkelName },
	}
)

func filterNonVineSchemas[T any](schemas []T, skelNameOf func(T) string) []T {
	ret := make([]T, 0)
	for _, schema := range schemas {
		if !isVineSchemaRef(skelNameOf(schema)) {
			ret = append(ret, schema)
		}
	}
	return ret
}

func buildDomainSchemaViews(
	domainVersions []core.DomainSchemaVersion,
) []core.DomainSchemaView {
	return buildDomainSchemaViewsWithFilter(domainVersions, func(skelName string) bool {
		return !isVineSchemaRef(skelName)
	})
}

func buildVineHubDomainSchemaViews(
	domainVersions []core.DomainSchemaVersion,
) []core.DomainSchemaView {
	views := buildDomainSchemaViewsWithFilter(domainVersions, isVineHubSchemaRef)
	ret := make([]core.DomainSchemaView, 0, len(views))
	for _, view := range views {
		if isVineHubDomain(view.DomainVersion.Schema.Domain) {
			ret = append(ret, view)
		}
	}
	return ret
}

func buildDomainSchemaViewsWithFilter(
	domainVersions []core.DomainSchemaVersion,
	include func(skelName string) bool,
) []core.DomainSchemaView {
	actorStates := schemaVersionStates(domainVersions, actorSchemaRefs, include)
	configStates := schemaVersionStates(domainVersions, configSchemaRefs, include)
	dataStates := schemaVersionStates(domainVersions, dataSchemaRefs, include)
	enumStates := schemaVersionStates(domainVersions, enumSchemaRefs, include)
	eventStates := schemaVersionStates(domainVersions, eventSchemaRefs, include)
	resourceStates := schemaVersionStates(domainVersions, resourceSchemaRefs, include)
	serviceStates := schemaVersionStates(domainVersions, serviceSchemaRefs, include)
	taskStates := schemaVersionStates(domainVersions, taskSchemaRefs, include)
	webStates := schemaVersionStates(domainVersions, webSchemaRefs, include)
	views := make([]core.DomainSchemaView, 0, len(domainVersions))
	for _, domainVersion := range domainVersions {
		views = append(views, core.DomainSchemaView{
			DomainVersion: domainVersion,
			Actors:        buildDomainSchemaItemVersions(domainVersion, actorStates, actorSchemaRefs, include),
			Configs:       buildDomainSchemaItemVersions(domainVersion, configStates, configSchemaRefs, include),
			Data:          buildDomainSchemaItemVersions(domainVersion, dataStates, dataSchemaRefs, include),
			Enums:         buildDomainSchemaItemVersions(domainVersion, enumStates, enumSchemaRefs, include),
			Events:        buildDomainSchemaItemVersions(domainVersion, eventStates, eventSchemaRefs, include),
			Resources:     buildDomainSchemaItemVersions(domainVersion, resourceStates, resourceSchemaRefs, include),
			Services:      buildDomainSchemaItemVersions(domainVersion, serviceStates, serviceSchemaRefs, include),
			Tasks:         buildDomainSchemaItemVersions(domainVersion, taskStates, taskSchemaRefs, include),
			Webs:          buildDomainSchemaItemVersions(domainVersion, webStates, webSchemaRefs, include),
		})
	}
	return views
}
