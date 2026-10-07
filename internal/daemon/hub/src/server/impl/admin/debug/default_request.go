package debug

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

var debugDefaultTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type _DebugDefaultBuilder struct {
	DescriptorRepo core.DescriptorRepo
}

func (b _DebugDefaultBuilder) defaultActorInfoJson(actorSkelName string) skeltype.JSON {
	actor := b.findActorDescriptor(actorSkelName)
	if actor.Auth == nil {
		return skeltype.JSON("{}")
	}
	return skeltype.JSON(debugPrettyJson(b.defaultDataValue(actor.Auth.Info)))
}

func (b _DebugDefaultBuilder) defaultParamsJson(method *skeldesc.Method) skeltype.JSON {
	if strings.TrimSpace(method.Example) != "" {
		return skeltype.JSON(debugPrettyJson(debugParseJson(method.Example)))
	}
	params := map[string]any{}
	for _, argument := range method.Arguments {
		params[argument.Name] = b.defaultMemberValue(argument)
	}
	return skeltype.JSON(debugPrettyJson(params))
}

func (b _DebugDefaultBuilder) defaultArgumentsJson(trigger *skeldesc.TaskTrigger) skeltype.JSON {
	args := map[string]any{}
	for _, argument := range trigger.Arguments {
		args[argument.Name] = b.defaultMemberValue(argument)
	}
	return skeltype.JSON(debugPrettyJson(args))
}

func (b _DebugDefaultBuilder) defaultEventJson(event *skeldesc.Event) skeltype.JSON {
	return skeltype.JSON(debugPrettyJson(b.defaultMembersValue(event.Members)))
}

func (b _DebugDefaultBuilder) defaultMemberValue(member *skeldesc.Member) any {
	if strings.TrimSpace(member.Example) != "" {
		return debugParseJson(member.Example)
	}
	return b.defaultValue(member.Type)
}

func (b _DebugDefaultBuilder) defaultValue(typeDescriptor *skeldesc.Type) any {
	if typeDescriptor == nil {
		return nil
	}
	if typeDescriptor.Nullable {
		return nil
	}
	switch typeDescriptor.Kind {
	case skeldesc.TypeKindScalar:
		switch typeDescriptor.Scalar {
		case skeldesc.ScalarBoolean:
			return false
		case skeldesc.ScalarInt, skeldesc.ScalarFloat:
			return 0
		case skeldesc.ScalarDecimal:
			return "0"
		case skeldesc.ScalarJSON:
			return map[string]any{}
		case skeldesc.ScalarUUID:
			return uuid.Nil().String()
		case skeldesc.ScalarTimestamp:
			return debugScalarJsonString(skeltype.NewTimestamp(debugDefaultTime))
		case skeldesc.ScalarDuration:
			return debugScalarJsonString(skeltype.NewDuration(0))
		case skeldesc.ScalarLocalDate:
			return debugScalarJsonString(skeltype.NewLocalDate(civil.DateOf(debugDefaultTime)))
		case skeldesc.ScalarLocalTime:
			return debugScalarJsonString(skeltype.NewLocalTime(civil.TimeOf(debugDefaultTime)))
		case skeldesc.ScalarLocalDateTime:
			return debugScalarJsonString(skeltype.NewLocalDateTime(civil.DateTimeOf(debugDefaultTime)))
		case skeldesc.ScalarBinary:
			return debugScalarJsonString(skeltype.Binary{})
		default:
			return ""
		}
	case skeldesc.TypeKindList:
		return []any{}
	case skeldesc.TypeKindMap:
		return map[string]any{}
	case skeldesc.TypeKindData:
		if dataDescriptor, ok := b.findDataDescriptor(typeDescriptor.SkelName); ok {
			return b.defaultMembersValue(dataDescriptor.Members)
		}
		return map[string]any{}
	case skeldesc.TypeKindConfig:
		if configDescriptor, ok := b.findConfigDescriptor(typeDescriptor.SkelName); ok {
			return b.defaultMembersValue(configDescriptor.Members)
		}
		return map[string]any{}
	case skeldesc.TypeKindEvent:
		if eventDescriptor, ok := b.findEventDescriptor(typeDescriptor.SkelName); ok {
			return b.defaultMembersValue(eventDescriptor.Members)
		}
		return map[string]any{}
	case skeldesc.TypeKindEnum:
		return ""
	default:
		return nil
	}
}

func (b _DebugDefaultBuilder) defaultDataValue(dataDescriptor *skeldesc.Data) map[string]any {
	return b.defaultMembersValue(dataDescriptor.Members)
}

func (b _DebugDefaultBuilder) defaultMembersValue(members []*skeldesc.Member) map[string]any {
	ret := map[string]any{}
	for _, member := range members {
		ret[member.Name] = b.defaultMemberValue(member)
	}
	return ret
}

func (b _DebugDefaultBuilder) findActorDescriptor(actorSkelName string) *skeldesc.Actor {
	for _, descriptor := range b.DescriptorRepo.ListActorDescriptors() {
		if descriptor.SkelName == actorSkelName {
			return descriptor
		}
	}
	ex.PanicNew(ex.NotFound, "actor descriptor not found")
	panic("unreachable")
}

func (b _DebugDefaultBuilder) findDataDescriptor(dataSkelName string) (*skeldesc.Data, bool) {
	for _, version := range b.DescriptorRepo.ListDataDescriptorVersions() {
		if version.Descriptor.SkelName == dataSkelName {
			return version.Descriptor, true
		}
	}
	return nil, false
}

func (b _DebugDefaultBuilder) findConfigDescriptor(configSkelName string) (*skeldesc.Config, bool) {
	for _, version := range b.DescriptorRepo.ListConfigDescriptorVersions() {
		if version.Descriptor.SkelName == configSkelName {
			return version.Descriptor, true
		}
	}
	return nil, false
}

func (b _DebugDefaultBuilder) findEventDescriptor(eventSkelName string) (*skeldesc.Event, bool) {
	for _, version := range b.DescriptorRepo.ListEventDescriptorVersions() {
		if version.Descriptor.SkelName == eventSkelName {
			return version.Descriptor, true
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
	ret, err := vcode.MarshalJsonWithOptions(value, jsontext.WithIndent("  "))
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
