package core

import (
	"encoding/json/v2"
	"math"
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
func AppConfigStatusFor(schema *skel.ConfigSchema, value string, enumSchemas []*skel.EnumSchema) AppConfigStatus {
	if schema == nil {
		return AppConfigStatusUnused
	}
	if !appConfigValueMatchesSchema(value, schema, enumSchemas) {
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

func appConfigValueMatchesSchema(value string, schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema) bool {
	var decoded any
	if json.Unmarshal([]byte(value), &decoded) != nil {
		return false
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		return false
	}

	expectedFields := make(map[string]struct{}, len(schema.Members))
	for _, member := range schema.Members {
		expectedFields[member.Name] = struct{}{}
		fieldValue, ok := object[member.Name]
		if !ok || !jsonValueMatchesType(fieldValue, member.Type, enumSchemas) {
			return false
		}
	}
	for name := range object {
		if _, ok := expectedFields[name]; !ok {
			return false
		}
	}
	return true
}

func jsonValueMatchesType(value any, typeSchema *skel.TypeSchema, enumSchemas []*skel.EnumSchema) bool {
	if value == nil {
		return typeSchema != nil && typeSchema.Nullable
	}
	if typeSchema == nil {
		return false
	}

	switch typeSchema.Kind {
	case skel.TypeKindScalar:
		return jsonValueMatchesScalar(value, typeSchema.Scalar)
	case skel.TypeKindEnum:
		text, ok := value.(string)
		return ok && enumValueExists(text, typeSchema, enumSchemas)
	case skel.TypeKindList:
		items, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if !jsonValueMatchesType(item, typeSchema.Element, enumSchemas) {
				return false
			}
		}
		return true
	case skel.TypeKindMap:
		items, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for key, item := range items {
			if !jsonMapKeyMatchesType(key, typeSchema.Key, enumSchemas) || !jsonValueMatchesType(item, typeSchema.Value, enumSchemas) {
				return false
			}
		}
		return true
	case skel.TypeKindData, skel.TypeKindConfig, skel.TypeKindEvent, skel.TypeKindTypeParameter:
		_, ok := value.(map[string]any)
		return ok
	default:
		return false
	}
}

func jsonValueMatchesScalar(value any, scalar skel.Scalar) bool {
	switch scalar {
	case skel.ScalarBool:
		_, ok := value.(bool)
		return ok
	case skel.ScalarInt, skel.ScalarLong:
		number, ok := value.(float64)
		return ok && math.Trunc(number) == number
	case skel.ScalarFloat, skel.ScalarDouble:
		_, ok := value.(float64)
		return ok
	case skel.ScalarDecimal:
		switch value.(type) {
		case float64, string:
			return true
		default:
			return false
		}
	case skel.ScalarJson:
		return true
	default:
		_, ok := value.(string)
		return ok
	}
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
			return true
		}
	default:
		return true
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
	return true
}
