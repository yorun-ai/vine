package schema

import (
	"cmp"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

func memoryActorSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.ActorSchema] {
	return memorySchemaRefs(schema.Actors, func(item *skel.ActorSchema) string { return item.SkelName }, func(item *skel.ActorSchema) string { return item.Hash })
}

func memoryConfigSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.ConfigSchema] {
	return memorySchemaRefs(schema.Configs, func(item *skel.ConfigSchema) string { return item.SkelName }, func(item *skel.ConfigSchema) string { return item.Hash })
}

func memoryDataSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.DataSchema] {
	refs := memorySchemaRefs(schema.Data, func(item *skel.DataSchema) string { return item.SkelName }, func(item *skel.DataSchema) string { return item.Hash })
	for _, actor := range schema.Actors {
		if actor.AuthCredential != nil {
			refs = append(refs, _MemorySchemaRef[*skel.DataSchema]{SkelName: actor.AuthCredential.SkelName, Hash: actor.AuthCredential.Hash, Schema: actor.AuthCredential})
		}
		if actor.AuthInfo != nil {
			refs = append(refs, _MemorySchemaRef[*skel.DataSchema]{SkelName: actor.AuthInfo.SkelName, Hash: actor.AuthInfo.Hash, Schema: actor.AuthInfo})
		}
	}
	return refs
}

func memoryEnumSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.EnumSchema] {
	return memorySchemaRefs(schema.Enums, func(item *skel.EnumSchema) string { return item.SkelName }, func(item *skel.EnumSchema) string { return item.Hash })
}

func memoryEventSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.EventSchema] {
	return memorySchemaRefs(schema.Events, func(item *skel.EventSchema) string { return item.SkelName }, func(item *skel.EventSchema) string { return item.Hash })
}

func memoryResourceSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.ResourceSchema] {
	return memorySchemaRefs(schema.Resources, func(item *skel.ResourceSchema) string { return item.SkelName }, func(item *skel.ResourceSchema) string { return item.Hash })
}

func memoryServiceSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.ServiceSchema] {
	refs := memorySchemaRefs(schema.Services, func(item *skel.ServiceSchema) string { return item.SkelName }, func(item *skel.ServiceSchema) string { return item.Hash })
	for _, actor := range schema.Actors {
		if actor.AuthService != nil {
			refs = append(refs, _MemorySchemaRef[*skel.ServiceSchema]{SkelName: actor.AuthService.SkelName, Hash: actor.AuthService.Hash, Schema: actor.AuthService})
		}
		if actor.PermService != nil {
			refs = append(refs, _MemorySchemaRef[*skel.ServiceSchema]{SkelName: actor.PermService.SkelName, Hash: actor.PermService.Hash, Schema: actor.PermService})
		}
	}
	for _, resource := range schema.Resources {
		if resource.CheckService != nil {
			refs = append(refs, _MemorySchemaRef[*skel.ServiceSchema]{SkelName: resource.CheckService.SkelName, Hash: resource.CheckService.Hash, Schema: resource.CheckService})
		}
	}
	return refs
}

func memoryTaskSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.TaskSchema] {
	return memorySchemaRefs(schema.Tasks, func(item *skel.TaskSchema) string { return item.SkelName }, func(item *skel.TaskSchema) string { return item.Hash })
}

func memoryWebSchemaRefs(schema *skel.DomainSchema) []_MemorySchemaRef[*skel.WebSchema] {
	return memorySchemaRefs(schema.Webs, func(item *skel.WebSchema) string { return item.SkelName }, func(item *skel.WebSchema) string { return item.Hash })
}

func memorySchemaRefs[T any](schemas []T, skelNameOf func(T) string, hashOf func(T) string) []_MemorySchemaRef[T] {
	refs := make([]_MemorySchemaRef[T], 0, len(schemas))
	for _, schema := range schemas {
		refs = append(refs, _MemorySchemaRef[T]{SkelName: skelNameOf(schema), Hash: hashOf(schema), Schema: schema})
	}
	return refs
}

func sortedConfigSchemas(schemas []*skel.ConfigSchema) []*skel.ConfigSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.ConfigSchema) string { return item.SkelName })
}

func sortedActorSchemas(schemas []*skel.ActorSchema) []*skel.ActorSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.ActorSchema) string { return item.SkelName })
}

func sortedEnumSchemas(schemas []*skel.EnumSchema) []*skel.EnumSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.EnumSchema) string { return item.SkelName })
}

func sortedDataSchemas(schemas []*skel.DataSchema) []*skel.DataSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.DataSchema) string { return item.SkelName })
}

func sortedEventSchemas(schemas []*skel.EventSchema) []*skel.EventSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.EventSchema) string { return item.SkelName })
}

func sortedResourceSchemas(schemas []*skel.ResourceSchema) []*skel.ResourceSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.ResourceSchema) string { return item.SkelName })
}

func sortedServiceSchemas(schemas []*skel.ServiceSchema) []*skel.ServiceSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.ServiceSchema) string { return item.SkelName })
}

func sortedTaskSchemas(schemas []*skel.TaskSchema) []*skel.TaskSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.TaskSchema) string { return item.SkelName })
}

func sortedWebSchemas(schemas []*skel.WebSchema) []*skel.WebSchema {
	return sortedSchemasBySkelName(schemas, func(item *skel.WebSchema) string { return item.SkelName })
}

func sortedSchemasBySkelName[T any](schemas []T, skelNameOf func(T) string) []T {
	return vslice.SortBy(schemas, func(a T, b T) bool {
		return cmp.Compare(skelNameOf(a), skelNameOf(b)) < 0
	})
}
