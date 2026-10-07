package core

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
)

// AppConfigStatus describes how a configuration value relates to the descriptor an
// application declared for it.
type AppConfigStatus string

const (
	AppConfigStatusNormal       AppConfigStatus = "NORMAL"
	AppConfigStatusUnused       AppConfigStatus = "UNUSED"
	AppConfigStatusUnconfigured AppConfigStatus = "UNCONFIGURED"
	AppConfigStatusMismatch     AppConfigStatus = "MISMATCH"
)

// AppConfigStatusFor returns the status of a value against the declaration of
// the application that owns the configuration.
func AppConfigStatusFor(descriptor *skeldesc.Config, value string, enumDescriptors []*skeldesc.Enum, dataDescriptors []*skeldesc.Data) AppConfigStatus {
	if descriptor == nil {
		return AppConfigStatusUnused
	}
	if !appConfigValueMatchesDescriptor(value, descriptor, enumDescriptors, dataDescriptors) {
		return AppConfigStatusMismatch
	}
	return AppConfigStatusNormal
}

// AppConfigLifecycleFor returns the lifecycle the owning application declared.
func AppConfigLifecycleFor(descriptor *skeldesc.Config) string {
	if descriptor == nil {
		return ""
	}
	return string(descriptor.Lifecycle)
}

func appConfigValueMatchesDescriptor(value string, descriptor *skeldesc.Config, enumDescriptors []*skeldesc.Enum, dataDescriptors []*skeldesc.Data) bool {
	index := map[string]*skeldesc.Data{}
	for _, data := range dataDescriptors {
		index[data.SkelName] = data
	}
	return configObjectMatches(jsontext.Value(value), descriptor.Members, enumDescriptors, index, nil)
}

func configObjectMatches(value jsontext.Value, members []*skeldesc.Member, enums []*skeldesc.Enum, data map[string]*skeldesc.Data, bindings map[string]*skeldesc.Type) bool {
	var object map[string]jsontext.Value
	if value.Kind() != '{' || json.Unmarshal(value, &object) != nil || len(object) != len(members) {
		return false
	}
	for _, member := range members {
		item, ok := object[member.Name]
		if !ok || !configTypeMatches(item, member.Type, enums, data, bindings) {
			return false
		}
	}
	return true
}

func configBoundType(kind *skeldesc.Type, bindings map[string]*skeldesc.Type) *skeldesc.Type {
	if kind == nil {
		return nil
	}
	if kind.Kind == skeldesc.TypeKindTypeParameter {
		bound := bindings[kind.Name]
		if bound == nil {
			return nil
		}
		copy := *bound
		copy.Nullable = copy.Nullable || kind.Nullable
		return &copy
	}
	copy := *kind
	copy.Element = configBoundType(kind.Element, bindings)
	copy.Key = configBoundType(kind.Key, bindings)
	copy.Value = configBoundType(kind.Value, bindings)
	copy.TypeArguments = make([]*skeldesc.Type, len(kind.TypeArguments))
	for i, arg := range kind.TypeArguments {
		copy.TypeArguments[i] = configBoundType(arg, bindings)
	}
	return &copy
}

func configTypeMatches(value jsontext.Value, kind *skeldesc.Type, enums []*skeldesc.Enum, data map[string]*skeldesc.Data, bindings map[string]*skeldesc.Type) bool {
	kind = configBoundType(kind, bindings)
	if kind == nil {
		return false
	}
	if value.Kind() == 'n' {
		return kind.Nullable
	}
	switch kind.Kind {
	case skeldesc.TypeKindScalar:
		return configScalarMatches(value, kind.Scalar)
	case skeldesc.TypeKindEnum:
		var text string
		return json.Unmarshal(value, &text) == nil && enumValueExists(text, kind, enums)
	case skeldesc.TypeKindList:
		var items []jsontext.Value
		if value.Kind() != '[' || json.Unmarshal(value, &items) != nil {
			return false
		}
		for _, item := range items {
			if !configTypeMatches(item, kind.Element, enums, data, nil) {
				return false
			}
		}
		return true
	case skeldesc.TypeKindMap:
		var items map[string]jsontext.Value
		if value.Kind() != '{' || json.Unmarshal(value, &items) != nil {
			return false
		}
		for key, item := range items {
			if !jsonMapKeyMatchesType(key, kind.Key, enums) || !configTypeMatches(item, kind.Value, enums, data, nil) {
				return false
			}
		}
		return true
	case skeldesc.TypeKindData:
		declaration := data[kind.SkelName]
		if declaration == nil || len(declaration.TypeParameters) != len(kind.TypeArguments) {
			return false
		}
		args := map[string]*skeldesc.Type{}
		for i, name := range declaration.TypeParameters {
			args[name] = kind.TypeArguments[i]
		}
		return configObjectMatches(value, declaration.Members, enums, data, args)
	default:
		return false
	}
}

func configScalarMatches(value jsontext.Value, scalar skeldesc.Scalar) bool {
	var target any
	switch scalar {
	case skeldesc.ScalarString:
		target = new(string)
	case skeldesc.ScalarBoolean:
		target = new(bool)
	case skeldesc.ScalarInt:
		target = new(int64)
	case skeldesc.ScalarFloat:
		target = new(float64)
	case skeldesc.ScalarBinary:
		target = new(skeltype.Binary)
	case skeldesc.ScalarDecimal:
		target = new(skeltype.Decimal)
	case skeldesc.ScalarJSON:
		target = new(skeltype.JSON)
	case skeldesc.ScalarUUID:
		target = new(skeltype.UUID)
	case skeldesc.ScalarDuration:
		target = new(skeltype.Duration)
	case skeldesc.ScalarTimestamp:
		target = new(skeltype.Timestamp)
	case skeldesc.ScalarLocalDate:
		target = new(skeltype.LocalDate)
	case skeldesc.ScalarLocalTime:
		target = new(skeltype.LocalTime)
	case skeldesc.ScalarLocalDateTime:
		target = new(skeltype.LocalDateTime)
	default:
		return false
	}
	return json.Unmarshal(value, target) == nil
}

func jsonMapKeyMatchesType(value string, typeDescriptor *skeldesc.Type, enumDescriptors []*skeldesc.Enum) bool {
	if typeDescriptor == nil {
		return false
	}
	switch typeDescriptor.Kind {
	case skeldesc.TypeKindEnum:
		return enumValueExists(value, typeDescriptor, enumDescriptors)
	case skeldesc.TypeKindScalar:
		switch typeDescriptor.Scalar {
		case skeldesc.ScalarBoolean:
			return value == "true" || value == "false"
		case skeldesc.ScalarInt:
			decoded, err := strconv.ParseInt(value, 10, 64)
			return err == nil && (strconv.FormatInt(decoded, 10) == value || value == "-0")
		default:
			raw, err := json.Marshal(value)
			return err == nil && configScalarMatches(raw, typeDescriptor.Scalar)
		}
	default:
		return false
	}
}

func enumValueExists(value string, typeDescriptor *skeldesc.Type, enumDescriptors []*skeldesc.Enum) bool {
	for _, enumDescriptor := range enumDescriptors {
		if enumDescriptor.SkelName != typeDescriptor.SkelName {
			continue
		}
		for _, item := range enumDescriptor.Items {
			if item.Name == value {
				return true
			}
		}
		return false
	}
	return false
}
