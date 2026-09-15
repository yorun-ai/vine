package schema

import (
	"cmp"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

func actorSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.ActorSchema] {
	return schemaRefs(schema.Actors, func(item *skel.ActorSchema) string { return item.SkelName }, func(item *skel.ActorSchema) string { return item.Hash })
}

func configSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.ConfigSchema] {
	return schemaRefs(schema.Configs, func(item *skel.ConfigSchema) string { return item.SkelName }, func(item *skel.ConfigSchema) string { return item.Hash })
}

func dataSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.DataSchema] {
	refs := schemaRefs(schema.Data, func(item *skel.DataSchema) string { return item.SkelName }, func(item *skel.DataSchema) string { return item.Hash })
	for _, actor := range schema.Actors {
		if actor.AuthCredential != nil {
			refs = append(refs, _SchemaRef[*skel.DataSchema]{SkelName: actor.AuthCredential.SkelName, Hash: actor.AuthCredential.Hash, Schema: actor.AuthCredential})
		}
		if actor.AuthInfo != nil {
			refs = append(refs, _SchemaRef[*skel.DataSchema]{SkelName: actor.AuthInfo.SkelName, Hash: actor.AuthInfo.Hash, Schema: actor.AuthInfo})
		}
	}
	return refs
}

func enumSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.EnumSchema] {
	return schemaRefs(schema.Enums, func(item *skel.EnumSchema) string { return item.SkelName }, func(item *skel.EnumSchema) string { return item.Hash })
}

func eventSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.EventSchema] {
	return schemaRefs(schema.Events, func(item *skel.EventSchema) string { return item.SkelName }, func(item *skel.EventSchema) string { return item.Hash })
}

func resourceSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.ResourceSchema] {
	return schemaRefs(schema.Resources, func(item *skel.ResourceSchema) string { return item.SkelName }, func(item *skel.ResourceSchema) string { return item.Hash })
}

func serviceSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.ServiceSchema] {
	refs := schemaRefs(schema.Services, func(item *skel.ServiceSchema) string { return item.SkelName }, func(item *skel.ServiceSchema) string { return item.Hash })
	for _, actor := range schema.Actors {
		if actor.AuthService != nil {
			refs = append(refs, _SchemaRef[*skel.ServiceSchema]{SkelName: actor.AuthService.SkelName, Hash: actor.AuthService.Hash, Schema: actor.AuthService})
		}
		if actor.PermService != nil {
			refs = append(refs, _SchemaRef[*skel.ServiceSchema]{SkelName: actor.PermService.SkelName, Hash: actor.PermService.Hash, Schema: actor.PermService})
		}
	}
	for _, resource := range schema.Resources {
		if resource.CheckService != nil {
			refs = append(refs, _SchemaRef[*skel.ServiceSchema]{SkelName: resource.CheckService.SkelName, Hash: resource.CheckService.Hash, Schema: resource.CheckService})
		}
	}
	return refs
}

func taskSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.TaskSchema] {
	return schemaRefs(schema.Tasks, func(item *skel.TaskSchema) string { return item.SkelName }, func(item *skel.TaskSchema) string { return item.Hash })
}

func webSchemaRefs(schema *skel.DomainSchema) []_SchemaRef[*skel.WebSchema] {
	return schemaRefs(schema.Webs, func(item *skel.WebSchema) string { return item.SkelName }, func(item *skel.WebSchema) string { return item.Hash })
}

func schemaRefs[T any](schemas []T, skelNameOf func(T) string, hashOf func(T) string) []_SchemaRef[T] {
	refs := make([]_SchemaRef[T], 0, len(schemas))
	for _, schema := range schemas {
		refs = append(refs, _SchemaRef[T]{SkelName: skelNameOf(schema), Hash: hashOf(schema), Schema: schema})
	}
	return refs
}

func sortedSchemasBySkelName[T any](schemas []T, skelNameOf func(T) string) []T {
	return vslice.SortBy(schemas, func(a T, b T) bool {
		return cmp.Compare(skelNameOf(a), skelNameOf(b)) < 0
	})
}
