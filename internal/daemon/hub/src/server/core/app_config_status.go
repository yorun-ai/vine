package core

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"

	"go.yorun.ai/vine/internal/core/skel"
)

// AppConfigStatus describes how a configuration value relates to the schema an
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
func AppConfigStatusFor(schema *skel.ConfigSchema, value string, enumSchemas []*skel.EnumSchema, dataSchemas []*skel.DataSchema) AppConfigStatus {
	if schema == nil {
		return AppConfigStatusUnused
	}
	if !appConfigValueMatchesSchema(value, schema, enumSchemas, dataSchemas) {
		return AppConfigStatusMismatch
	}
	return AppConfigStatusNormal
}

// AppConfigLifecycleFor returns the lifecycle the owning application declared.
func AppConfigLifecycleFor(schema *skel.ConfigSchema) string {
	if schema == nil {
		return ""
	}
	return schema.Lifecycle
}

func appConfigValueMatchesSchema(value string, schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema, dataSchemas []*skel.DataSchema) bool {
	index := map[string]*skel.DataSchema{}
	for _, data := range dataSchemas {
		index[data.SkelName] = data
	}
	return configObjectMatches(jsontext.Value(value), schema.Members, enumSchemas, index, nil)
}

func configObjectMatches(value jsontext.Value, members []*skel.MemberSchema, enums []*skel.EnumSchema, data map[string]*skel.DataSchema, bindings map[string]*skel.TypeSchema) bool {
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

func configBoundType(kind *skel.TypeSchema, bindings map[string]*skel.TypeSchema) *skel.TypeSchema {
	if kind == nil {
		return nil
	}
	if kind.Kind == skel.TypeKindTypeParameter {
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
	copy.TypeArguments = make([]*skel.TypeSchema, len(kind.TypeArguments))
	for i, arg := range kind.TypeArguments {
		copy.TypeArguments[i] = configBoundType(arg, bindings)
	}
	return &copy
}

func configTypeMatches(value jsontext.Value, kind *skel.TypeSchema, enums []*skel.EnumSchema, data map[string]*skel.DataSchema, bindings map[string]*skel.TypeSchema) bool {
	kind = configBoundType(kind, bindings)
	if kind == nil {
		return false
	}
	if value.Kind() == 'n' {
		return kind.Nullable
	}
	switch kind.Kind {
	case skel.TypeKindScalar:
		return configScalarMatches(value, kind.Scalar)
	case skel.TypeKindEnum:
		var text string
		return json.Unmarshal(value, &text) == nil && enumValueExists(text, kind, enums)
	case skel.TypeKindList:
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
	case skel.TypeKindMap:
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
	case skel.TypeKindData:
		declaration := data[kind.SkelName]
		if declaration == nil || len(declaration.TypeParameters) != len(kind.TypeArguments) {
			return false
		}
		args := map[string]*skel.TypeSchema{}
		for i, name := range declaration.TypeParameters {
			args[name] = kind.TypeArguments[i]
		}
		return configObjectMatches(value, declaration.Members, enums, data, args)
	default:
		return false
	}
}

func configScalarMatches(value jsontext.Value, scalar skel.Scalar) bool {
	var target any
	switch scalar {
	case skel.ScalarString:
		target = new(string)
	case skel.ScalarBool:
		target = new(bool)
	case skel.ScalarInt, skel.ScalarLong:
		target = new(int64)
	case skel.ScalarFloat, skel.ScalarDouble:
		target = new(float64)
	case skel.ScalarBinary:
		target = new(skel.Binary)
	case skel.ScalarDecimal:
		target = new(skel.Decimal)
	case skel.ScalarJson:
		target = new(skel.JSON)
	case skel.ScalarUuid:
		target = new(skel.UUID)
	case skel.ScalarDuration:
		target = new(skel.Duration)
	case skel.ScalarTimestamp:
		target = new(skel.Timestamp)
	case skel.ScalarLocalDate:
		target = new(skel.LocalDate)
	case skel.ScalarLocalTime:
		target = new(skel.LocalTime)
	case skel.ScalarLocalDateTime:
		target = new(skel.LocalDateTime)
	default:
		return false
	}
	return json.Unmarshal(value, target) == nil
}

func jsonMapKeyMatchesType(value string, typeSchema *skel.TypeSchema, enumSchemas []*skel.EnumSchema) bool {
	if typeSchema == nil {
		return false
	}
	switch typeSchema.Kind {
	case skel.TypeKindEnum:
		return enumValueExists(value, typeSchema, enumSchemas)
	case skel.TypeKindScalar:
		switch typeSchema.Scalar {
		case skel.ScalarBool:
			return value == "true" || value == "false"
		case skel.ScalarInt, skel.ScalarLong:
			decoded, err := strconv.ParseInt(value, 10, 64)
			return err == nil && (strconv.FormatInt(decoded, 10) == value || value == "-0")
		default:
			raw, err := json.Marshal(value)
			return err == nil && configScalarMatches(raw, typeSchema.Scalar)
		}
	default:
		return false
	}
}

func enumValueExists(value string, typeSchema *skel.TypeSchema, enumSchemas []*skel.EnumSchema) bool {
	for _, enumSchema := range enumSchemas {
		if enumSchema.SkelName != typeSchema.SkelName {
			continue
		}
		for _, item := range enumSchema.Items {
			if item.Name == value {
				return true
			}
		}
		return false
	}
	return false
}
