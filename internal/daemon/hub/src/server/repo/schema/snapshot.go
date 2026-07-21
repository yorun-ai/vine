package schema

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func memoryRefreshSchemaSnapshot() {
	domainVersions := memoryBuildDomainSchemaVersions()
	schemas := make([]*skel.DomainSchema, 0, len(domainVersions))
	for _, version := range domainVersions {
		if version.Main {
			schemas = append(schemas, version.Schema)
		}
	}
	actorVersions := memoryBuildSchemaVersions(domainVersions, memoryActorSchemaRefs)
	configVersions := memoryBuildSchemaVersions(domainVersions, memoryConfigSchemaRefs)
	dataVersions := memoryBuildSchemaVersions(domainVersions, memoryDataSchemaRefs)
	enumVersions := memoryBuildSchemaVersions(domainVersions, memoryEnumSchemaRefs)
	eventVersions := memoryBuildSchemaVersions(domainVersions, memoryEventSchemaRefs)
	resourceVersions := memoryBuildSchemaVersions(domainVersions, memoryResourceSchemaRefs)
	serviceVersions := memoryBuildSchemaVersions(domainVersions, memoryServiceSchemaRefs)
	taskVersions := memoryBuildSchemaVersions(domainVersions, memoryTaskSchemaRefs)
	webVersions := memoryBuildSchemaVersions(domainVersions, memoryWebSchemaRefs)
	memorySchemaSnapshot = _MemorySchemaSnapshot{
		DomainSchemas:  schemas,
		DomainVersions: domainVersions,
		DomainViews:    memoryBuildDomainSchemaViews(domainVersions),
		VineHubViews:   memoryBuildVineHubDomainSchemaViews(domainVersions),
		ConfigSchemas:  memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.ConfigSchema { return schema.Configs }, sortedConfigSchemas),
		ActorSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.ActorSchema {
			return memoryFilterNonVineSchemas(schema.Actors, func(item *skel.ActorSchema) string { return item.SkelName })
		}, sortedActorSchemas),
		DataSchemas:  memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.DataSchema { return schema.Data }, sortedDataSchemas),
		EnumSchemas:  memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.EnumSchema { return schema.Enums }, sortedEnumSchemas),
		EventSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.EventSchema { return schema.Events }, sortedEventSchemas),
		ResourceSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.ResourceSchema {
			return memoryFilterNonVineSchemas(schema.Resources, func(item *skel.ResourceSchema) string { return item.SkelName })
		}, sortedResourceSchemas),
		ServiceSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.ServiceSchema {
			return memoryFilterNonVineSchemas(schema.Services, func(item *skel.ServiceSchema) string { return item.SkelName })
		}, sortedServiceSchemas),
		TaskSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.TaskSchema { return schema.Tasks }, sortedTaskSchemas),
		WebSchemas: memoryBuildLatestSchemas(schemas, func(schema *skel.DomainSchema) []*skel.WebSchema {
			return memoryFilterNonVineSchemas(schema.Webs, func(item *skel.WebSchema) string { return item.SkelName })
		}, sortedWebSchemas),
		ActorVersions:    actorVersions,
		ConfigVersions:   configVersions,
		DataVersions:     dataVersions,
		EnumVersions:     enumVersions,
		EventVersions:    eventVersions,
		ResourceVersions: resourceVersions,
		ServiceVersions:  serviceVersions,
		TaskVersions:     taskVersions,
		WebVersions:      webVersions,
	}
}

func memoryBuildLatestSchemas[T any](
	domainSchemas []*skel.DomainSchema,
	getSchemas func(schema *skel.DomainSchema) []T,
	sortedSchemas func([]T) []T,
) []T {
	schemas := make([]T, 0)
	for _, domainSchema := range domainSchemas {
		schemas = append(schemas, getSchemas(domainSchema)...)
	}
	return sortedSchemas(schemas)
}

func memoryFilterNonVineSchemas[T any](schemas []T, skelNameOf func(T) string) []T {
	ret := make([]T, 0)
	for _, schema := range schemas {
		if !memoryIsVineSchemaRef(skelNameOf(schema)) {
			ret = append(ret, schema)
		}
	}
	return ret
}

func memoryBuildDomainSchemaViews(
	domainVersions []core.DomainSchemaVersion,
) []core.DomainSchemaView {
	return memoryBuildDomainSchemaViewsWithFilter(domainVersions, func(skelName string) bool {
		return !memoryIsVineSchemaRef(skelName)
	})
}

func memoryBuildVineHubDomainSchemaViews(
	domainVersions []core.DomainSchemaVersion,
) []core.DomainSchemaView {
	views := memoryBuildDomainSchemaViewsWithFilter(domainVersions, memoryIsVineHubSchemaRef)
	ret := make([]core.DomainSchemaView, 0, len(views))
	for _, view := range views {
		if view.DomainVersion.Schema.Domain == "vine.hub" {
			ret = append(ret, view)
		}
	}
	return ret
}

func memoryBuildDomainSchemaViewsWithFilter(
	domainVersions []core.DomainSchemaVersion,
	include func(skelName string) bool,
) []core.DomainSchemaView {
	actorStates := memorySchemaVersionStates(domainVersions, memoryActorSchemaRefs, include)
	configStates := memorySchemaVersionStates(domainVersions, memoryConfigSchemaRefs, include)
	dataStates := memorySchemaVersionStates(domainVersions, memoryDataSchemaRefs, include)
	enumStates := memorySchemaVersionStates(domainVersions, memoryEnumSchemaRefs, include)
	eventStates := memorySchemaVersionStates(domainVersions, memoryEventSchemaRefs, include)
	resourceStates := memorySchemaVersionStates(domainVersions, memoryResourceSchemaRefs, include)
	serviceStates := memorySchemaVersionStates(domainVersions, memoryServiceSchemaRefs, include)
	taskStates := memorySchemaVersionStates(domainVersions, memoryTaskSchemaRefs, include)
	webStates := memorySchemaVersionStates(domainVersions, memoryWebSchemaRefs, include)
	views := make([]core.DomainSchemaView, 0, len(domainVersions))
	for _, domainVersion := range domainVersions {
		views = append(views, core.DomainSchemaView{
			DomainVersion: domainVersion,
			Actors:        memoryBuildDomainSchemaItemVersions(domainVersion, actorStates, memoryActorSchemaRefs, include),
			Configs:       memoryBuildDomainSchemaItemVersions(domainVersion, configStates, memoryConfigSchemaRefs, include),
			Data:          memoryBuildDomainSchemaItemVersions(domainVersion, dataStates, memoryDataSchemaRefs, include),
			Enums:         memoryBuildDomainSchemaItemVersions(domainVersion, enumStates, memoryEnumSchemaRefs, include),
			Events:        memoryBuildDomainSchemaItemVersions(domainVersion, eventStates, memoryEventSchemaRefs, include),
			Resources:     memoryBuildDomainSchemaItemVersions(domainVersion, resourceStates, memoryResourceSchemaRefs, include),
			Services:      memoryBuildDomainSchemaItemVersions(domainVersion, serviceStates, memoryServiceSchemaRefs, include),
			Tasks:         memoryBuildDomainSchemaItemVersions(domainVersion, taskStates, memoryTaskSchemaRefs, include),
			Webs:          memoryBuildDomainSchemaItemVersions(domainVersion, webStates, memoryWebSchemaRefs, include),
		})
	}
	return views
}
