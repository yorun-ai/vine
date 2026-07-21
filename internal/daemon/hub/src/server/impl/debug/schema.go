package debug

import (
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/util/vslice"
)

func (s *ServiceDebugServiceServerImpl) hasServiceSchema(serviceSkelName string, schemaHash string) bool {
	for _, version := range s.SchemaRepo.ListServiceSchemaVersions() {
		if version.Schema.SkelName == serviceSkelName && (schemaHash == "" || version.SchemaHash == schemaHash) {
			return true
		}
	}
	return false
}

func (s *ServiceDebugServiceServerImpl) findServiceSchema(serviceSkelName string, schemaHash string) *skel.ServiceSchema {
	for _, version := range s.SchemaRepo.ListServiceSchemaVersions() {
		if version.Schema.SkelName == serviceSkelName && (schemaHash == "" || version.SchemaHash == schemaHash) {
			return version.Schema
		}
	}
	ex.PanicNew(ex.NotFound, "service schema not found")
	panic("unreachable")
}

func (s *ServiceDebugServiceServerImpl) findMethodSchema(serviceSchema *skel.ServiceSchema, methodSkelName string) *skel.MethodSchema {
	method, ok := serviceSchema.MethodByName(methodSkelName)
	ex.PanicNewIfNot(ok, ex.NotFound, "method schema not found")
	return method
}

func (s *ServiceDebugServiceServerImpl) findActorSchema(actorSkelName string) *skel.ActorSchema {
	for _, schema := range s.SchemaRepo.ListActorSchemas() {
		if schema.SkelName == actorSkelName {
			return schema
		}
	}
	ex.PanicNew(ex.NotFound, "actor schema not found")
	panic("unreachable")
}

func (s *ServiceDebugServiceServerImpl) serviceDebugActors(serviceSchema *skel.ServiceSchema) []skeled.ServiceDebugActorItem {
	ret := make([]skeled.ServiceDebugActorItem, 0, len(serviceSchema.Audiences))
	for _, audience := range serviceSchema.Audiences {
		actor := s.findActorSchema(audience.SkelName)
		if actor.AuthInfo == nil {
			continue
		}
		ret = append(ret, skeled.ServiceDebugActorItem{
			Name:          actor.Name,
			SkelName:      actor.SkelName,
			InfoSkelName:  actor.AuthInfo.SkelName,
			ActorInfoJson: s.defaultBuilder().defaultActorInfoJson(actor.SkelName),
		})
	}
	return vslice.SortBy(ret, func(a skeled.ServiceDebugActorItem, b skeled.ServiceDebugActorItem) bool {
		return strings.Compare(a.SkelName, b.SkelName) < 0
	})
}

func toServiceDebugMethodItem(method *skel.MethodSchema) skeled.ServiceDebugMethodItem {
	return skeled.ServiceDebugMethodItem{
		Name:              method.Name,
		SkelName:          method.SkelName,
		Description:       optionalString(method.Description),
		InputDescription:  optionalString(method.InputDescription),
		OutputDescription: optionalString(method.OutputDescription),
		Example:           optionalString(method.Example),
		OutputExample:     optionalString(method.OutputExample),
		Arguments:         toDebugSkeletonFields(method.Arguments),
		ResultType:        formatSkeletonType(method.ResultType),
	}
}

func toDebugSkeletonFields(schemas []*skel.MemberSchema) []skeled.SkeletonField {
	ret := make([]skeled.SkeletonField, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonField{
			Name:        schema.Name,
			Type:        formatSkeletonType(schema.Type),
			Description: optionalString(schema.Description),
			Example:     optionalString(schema.Example),
		})
	}
	return ret
}

func formatSkeletonType(typeSchema *skel.TypeSchema) string {
	if typeSchema == nil {
		return ""
	}
	var ret string
	switch typeSchema.Kind {
	case skel.TypeKindScalar:
		ret = string(typeSchema.Scalar)
	case skel.TypeKindEnum, skel.TypeKindData, skel.TypeKindConfig, skel.TypeKindEvent:
		ret = formatSkeletonNamedType(typeSchema)
		if len(typeSchema.TypeArguments) > 0 {
			args := make([]string, 0, len(typeSchema.TypeArguments))
			for _, arg := range typeSchema.TypeArguments {
				args = append(args, formatSkeletonType(arg))
			}
			ret += "<" + strings.Join(args, ", ") + ">"
		}
	case skel.TypeKindTypeParameter:
		ret = typeSchema.Name
		if ret == "" {
			ret = shortSkelName(typeSchema.SkelName)
		}
	case skel.TypeKindList:
		ret = "list<" + formatSkeletonType(typeSchema.Element) + ">"
	case skel.TypeKindMap:
		ret = "map<" + formatSkeletonType(typeSchema.Key) + ", " + formatSkeletonType(typeSchema.Value) + ">"
	default:
		ret = string(typeSchema.Kind)
	}
	if typeSchema.Nullable {
		ret += "?"
	}
	return ret
}

func formatSkeletonNamedType(typeSchema *skel.TypeSchema) string {
	if typeSchema.SkelName != "" {
		return typeSchema.SkelName
	}
	if typeSchema.Name != "" {
		return typeSchema.Name
	}
	return shortSkelName(typeSchema.SkelName)
}

func shortSkelName(skelName string) string {
	_, name, ok := strings.Cut(skelName, ".")
	for ok {
		skelName = name
		_, name, ok = strings.Cut(skelName, ".")
	}
	return skelName
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
