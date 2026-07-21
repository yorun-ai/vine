package redised

import (
	"fmt"

	"go.yorun.ai/vine/internal/core/skel"
)

const (
	schemaActorPrefix    = "schema:actor"
	schemaActorKeyFormat = schemaActorPrefix + ":%s"

	schemaServicePrefix    = "schema:service"
	schemaServiceKeyFormat = schemaServicePrefix + ":%s"

	schemaResourcePrefix    = "schema:resource"
	schemaResourceKeyFormat = schemaResourcePrefix + ":%s"
)

type SchemaActor = skel.ActorSchema
type SchemaService = skel.ServiceSchema
type SchemaResource = skel.ResourceSchema

func FormatSchemaActorKey(actorSkelName string) string {
	return fmt.Sprintf(schemaActorKeyFormat, actorSkelName)
}

func FormatSchemaActorPrefix() string {
	return schemaActorPrefix
}

func FormatSchemaServiceKey(serviceSkelName string) string {
	return fmt.Sprintf(schemaServiceKeyFormat, serviceSkelName)
}

func FormatSchemaServicePrefix() string {
	return schemaServicePrefix
}

func FormatSchemaResourceKey(resourceSkelName string) string {
	return fmt.Sprintf(schemaResourceKeyFormat, resourceSkelName)
}

func FormatSchemaResourcePrefix() string {
	return schemaResourcePrefix
}
