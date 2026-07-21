package debug

import (
	"encoding/json"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

var debugDefaultTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type _DebugDefaultBuilder struct {
	SchemaRepo core.SchemaRepo
}

func (b _DebugDefaultBuilder) defaultActorInfoJson(actorSkelName string) skel.JSON {
	actor := b.findActorSchema(actorSkelName)
	if actor.AuthInfo == nil {
		return skel.JSON("{}")
	}
	return skel.JSON(debugPrettyJson(b.defaultDataValue(actor.AuthInfo)))
}

func (b _DebugDefaultBuilder) defaultParamsJson(method *skel.MethodSchema) skel.JSON {
	if strings.TrimSpace(method.Example) != "" {
		return skel.JSON(debugPrettyJson(debugParseJson(method.Example)))
	}
	params := map[string]any{}
	for _, argument := range method.Arguments {
		params[argument.Name] = b.defaultMemberValue(argument)
	}
	return skel.JSON(debugPrettyJson(params))
}

func (b _DebugDefaultBuilder) defaultArgumentsJson(trigger *skel.TriggerSchema) skel.JSON {
	if strings.TrimSpace(trigger.Example) != "" {
		return skel.JSON(debugPrettyJson(debugParseJson(trigger.Example)))
	}
	args := map[string]any{}
	for _, argument := range trigger.Arguments {
		args[argument.Name] = b.defaultMemberValue(argument)
	}
	return skel.JSON(debugPrettyJson(args))
}

func (b _DebugDefaultBuilder) defaultEventJson(event *skel.EventSchema) skel.JSON {
	return skel.JSON(debugPrettyJson(b.defaultMembersValue(event.Members)))
}

func (b _DebugDefaultBuilder) defaultMemberValue(member *skel.MemberSchema) any {
	if strings.TrimSpace(member.Example) != "" {
		return debugParseJson(member.Example)
	}
	return b.defaultValue(member.Type)
}

func (b _DebugDefaultBuilder) defaultValue(typeSchema *skel.TypeSchema) any {
	if typeSchema == nil {
		return nil
	}
	if typeSchema.Nullable {
		return nil
	}
	switch typeSchema.Kind {
	case skel.TypeKindScalar:
		switch typeSchema.Scalar {
		case skel.ScalarBool:
			return false
		case skel.ScalarInt, skel.ScalarLong, skel.ScalarFloat, skel.ScalarDouble:
			return 0
		case skel.ScalarDecimal:
			return "0"
		case skel.ScalarJson:
			return map[string]any{}
		case skel.ScalarUuid:
			return uuid.Nil.String()
		case skel.ScalarTimestamp:
			return debugScalarJsonString(skel.NewTimestamp(debugDefaultTime))
		case skel.ScalarDuration:
			return debugScalarJsonString(skel.NewDuration(0))
		case skel.ScalarLocalDate:
			return debugScalarJsonString(skel.NewLocalDate(civil.DateOf(debugDefaultTime)))
		case skel.ScalarLocalTime:
			return debugScalarJsonString(skel.NewLocalTime(civil.TimeOf(debugDefaultTime)))
		case skel.ScalarLocalDateTime:
			return debugScalarJsonString(skel.NewLocalDateTime(civil.DateTimeOf(debugDefaultTime)))
		case skel.ScalarBinary:
			return debugScalarJsonString(skel.Binary{})
		default:
			return ""
		}
	case skel.TypeKindList:
		return []any{}
	case skel.TypeKindMap:
		return map[string]any{}
	case skel.TypeKindData:
		if dataSchema, ok := b.findDataSchema(typeSchema.SkelName); ok {
			return b.defaultMembersValue(dataSchema.Members)
		}
		return map[string]any{}
	case skel.TypeKindConfig:
		if configSchema, ok := b.findConfigSchema(typeSchema.SkelName); ok {
			return b.defaultMembersValue(configSchema.Members)
		}
		return map[string]any{}
	case skel.TypeKindEvent:
		if eventSchema, ok := b.findEventSchema(typeSchema.SkelName); ok {
			return b.defaultMembersValue(eventSchema.Members)
		}
		return map[string]any{}
	case skel.TypeKindEnum:
		return ""
	default:
		return nil
	}
}

func (b _DebugDefaultBuilder) defaultDataValue(dataSchema *skel.DataSchema) map[string]any {
	return b.defaultMembersValue(dataSchema.Members)
}

func (b _DebugDefaultBuilder) defaultMembersValue(members []*skel.MemberSchema) map[string]any {
	ret := map[string]any{}
	for _, member := range members {
		ret[member.Name] = b.defaultMemberValue(member)
	}
	return ret
}

func (b _DebugDefaultBuilder) findActorSchema(actorSkelName string) *skel.ActorSchema {
	for _, schema := range b.SchemaRepo.ListActorSchemas() {
		if schema.SkelName == actorSkelName {
			return schema
		}
	}
	ex.PanicNew(ex.NotFound, "actor schema not found")
	panic("unreachable")
}

func (b _DebugDefaultBuilder) findDataSchema(dataSkelName string) (*skel.DataSchema, bool) {
	for _, version := range b.SchemaRepo.ListDataSchemaVersions() {
		if version.Schema.SkelName == dataSkelName {
			return version.Schema, true
		}
	}
	return nil, false
}

func (b _DebugDefaultBuilder) findConfigSchema(configSkelName string) (*skel.ConfigSchema, bool) {
	for _, version := range b.SchemaRepo.ListConfigSchemaVersions() {
		if version.Schema.SkelName == configSkelName {
			return version.Schema, true
		}
	}
	return nil, false
}

func (b _DebugDefaultBuilder) findEventSchema(eventSkelName string) (*skel.EventSchema, bool) {
	for _, version := range b.SchemaRepo.ListEventSchemaVersions() {
		if version.Schema.SkelName == eventSkelName {
			return version.Schema, true
		}
	}
	return nil, false
}

func debugParseJson(value string) any {
	var ret any
	err := json.Unmarshal([]byte(value), &ret)
	ex.PanicNewIfError(err, ex.InvalidRequest)
	return ret
}

func debugPrettyJson(value any) string {
	ret, err := json.MarshalIndent(value, "", "  ")
	ex.PanicNewIfError(err, ex.InvalidRequest)
	return string(ret)
}

func debugScalarJsonString(value any) string {
	data, err := json.Marshal(value)
	ex.PanicNewIfError(err, ex.InvalidRequest)

	var ret string
	err = json.Unmarshal(data, &ret)
	ex.PanicNewIfError(err, ex.InvalidRequest)
	return ret
}
