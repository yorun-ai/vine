package impl

import (
	"cmp"
	"encoding/json"
	"math"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

const (
	appConfigStatusNormal       = "NORMAL"
	appConfigStatusUnused       = "UNUSED"
	appConfigStatusUnconfigured = "UNCONFIGURED"
	appConfigStatusMismatch     = "MISMATCH"
)

type AppConfigServiceServerImpl struct {
	skeled.DefaultAppConfigServiceServer

	AppConfigCore *core.AppConfigCore `inject:""`
	SchemaRepo    core.SchemaRepo     `inject:""`
}

type _AppConfigListItem struct {
	ConfigItem skeled.AppConfigItem
	CreatedAt  time.Time
}

func (s *AppConfigServiceServerImpl) List() []skeled.AppConfigItem {
	items := s.AppConfigCore.List()
	schemas := s.SchemaRepo.ListAppConfigSchemas()
	enumSchemas := s.SchemaRepo.ListEnumSchemas()
	ret := make([]_AppConfigListItem, 0, len(items)+len(schemas))
	usedSchemas := make(map[*skel.ConfigSchema]struct{}, len(items))
	for _, item := range items {
		schema := findConfigSchema(item.Name, schemas)
		if schema != nil {
			usedSchemas[schema] = struct{}{}
		}
		ret = append(ret, _AppConfigListItem{
			ConfigItem: toServerAppConfigItem(item, schema, enumSchemas),
			CreatedAt:  item.CreatedAt,
		})
	}
	for _, schema := range schemas {
		if _, ok := usedSchemas[schema]; ok {
			continue
		}
		ret = append(ret, _AppConfigListItem{
			ConfigItem: toServerUnconfiguredAppConfigItem(schema, enumSchemas),
		})
	}
	return serverAppConfigItems(sortedServerAppConfigItems(ret))
}

func (s *AppConfigServiceServerImpl) Get(id int) skeled.AppConfigItem {
	return s.toServerAppConfigItem(s.AppConfigCore.Get(id))
}

func (s *AppConfigServiceServerImpl) Update(id int, update skeled.AppConfigUpdate) skeled.AppConfigItem {
	return s.toServerAppConfigItem(s.AppConfigCore.Update(id, core.AppConfigUpdate{
		Value: update.Value,
	}))
}

func (s *AppConfigServiceServerImpl) Create(creation skeled.AppConfigCreation) skeled.AppConfigItem {
	ex.PanicNewIfNot(isValidConfigSkelName(creation.SkelName), ex.OperationFailed, ex.F("invalid config skelName %q", creation.SkelName))
	return s.toServerAppConfigItem(s.AppConfigCore.Create(core.AppConfigCreation{
		Name:  creation.SkelName,
		Value: creation.Value,
	}))
}

func (s *AppConfigServiceServerImpl) Remove(id int) bool {
	item := s.AppConfigCore.Get(id)
	schema := findConfigSchema(item.Name, s.SchemaRepo.ListAppConfigSchemas())
	ex.PanicNewIfNot(schema == nil, ex.OperationFailed, ex.F("config %q is not unused", item.Name))
	return s.AppConfigCore.Remove(id)
}

func (s *AppConfigServiceServerImpl) toServerAppConfigItem(item *core.AppConfig) skeled.AppConfigItem {
	return toServerAppConfigItem(item, findConfigSchema(item.Name, s.SchemaRepo.ListAppConfigSchemas()), s.SchemaRepo.ListEnumSchemas())
}

func toServerAppConfigItem(item *core.AppConfig, schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema) skeled.AppConfigItem {
	return skeled.AppConfigItem{
		Id:        item.Id,
		Key:       item.Name,
		Status:    configItemStatus(schema, item.Value, enumSchemas),
		Lifecycle: configItemLifecycle(schema),
		Value:     item.Value,
		Schema:    toServerAppConfigSchema(schema, enumSchemas),
	}
}

func toServerUnconfiguredAppConfigItem(schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema) skeled.AppConfigItem {
	return skeled.AppConfigItem{
		Key:       schema.SkelName,
		Status:    appConfigStatusUnconfigured,
		Lifecycle: schema.Lifecycle,
		Value:     "",
		Schema:    toServerAppConfigSchema(schema, enumSchemas),
	}
}

func configItemStatus(schema *skel.ConfigSchema, value string, enumSchemas []*skel.EnumSchema) string {
	if schema == nil {
		return appConfigStatusUnused
	}
	if !appConfigValueMatchesSchema(value, schema, enumSchemas) {
		return appConfigStatusMismatch
	}
	return appConfigStatusNormal
}

func sortedServerAppConfigItems(items []_AppConfigListItem) []_AppConfigListItem {
	return vslice.SortBy(items, func(a, b _AppConfigListItem) bool {
		aStatusOrder := appConfigStatusOrder(a.ConfigItem.Status)
		bStatusOrder := appConfigStatusOrder(b.ConfigItem.Status)
		if aStatusOrder != bStatusOrder {
			return aStatusOrder < bStatusOrder
		}
		if a.ConfigItem.Status == appConfigStatusUnused && !a.CreatedAt.Equal(b.CreatedAt) {
			return b.CreatedAt.Compare(a.CreatedAt) < 0
		}
		return cmp.Compare(a.ConfigItem.Key, b.ConfigItem.Key) < 0
	})
}

func serverAppConfigItems(items []_AppConfigListItem) []skeled.AppConfigItem {
	ret := make([]skeled.AppConfigItem, 0, len(items))
	for _, item := range items {
		ret = append(ret, item.ConfigItem)
	}
	return ret
}

func appConfigStatusOrder(status string) int {
	switch status {
	case appConfigStatusMismatch:
		return 0
	case appConfigStatusUnconfigured:
		return 1
	case appConfigStatusUnused:
		return 2
	default:
		return 3
	}
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
			var decoded float64
			if json.Unmarshal([]byte(value), &decoded) != nil {
				return false
			}
			return math.Trunc(decoded) == decoded
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

func isValidConfigSkelName(skelName string) bool {
	parts := strings.Split(skelName, ".")
	if len(parts) < 2 {
		return false
	}

	configName := parts[len(parts)-1]
	if !strings.HasSuffix(configName, "Config") || !isValidSkelIdentifier(configName) {
		return false
	}

	for _, part := range parts[:len(parts)-1] {
		if !isValidSkelIdentifier(part) {
			return false
		}
	}
	return true
}

func isValidSkelIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if !isAsciiLetter(char) && char != '_' {
				return false
			}
			continue
		}
		if !isAsciiLetter(char) && !isAsciiDigit(char) && char != '_' {
			return false
		}
	}
	return true
}

func isAsciiLetter(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isAsciiDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

func configItemLifecycle(schema *skel.ConfigSchema) string {
	if schema == nil {
		return ""
	}
	return schema.Lifecycle
}

func toServerAppConfigSchema(schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema) *skeled.AppConfigSchema {
	if schema == nil {
		return nil
	}
	return &skeled.AppConfigSchema{
		SkelName:    schema.SkelName,
		Name:        schema.Name,
		Description: optionalString(schema.Description),
		Lifecycle:   schema.Lifecycle,
		Fields:      toServerAppConfigSchemaFields(schema.Members, enumSchemas),
	}
}

func findConfigSchema(name string, schemas []*skel.ConfigSchema) *skel.ConfigSchema {
	for _, schema := range schemas {
		if schema.SkelName == name {
			return schema
		}
	}
	return nil
}

func toServerAppConfigSchemaFields(members []*skel.MemberSchema, enumSchemas []*skel.EnumSchema) []skeled.AppConfigSchemaField {
	fields := make([]skeled.AppConfigSchemaField, 0, len(members))
	for _, member := range members {
		fields = append(fields, skeled.AppConfigSchemaField{
			Name:        member.Name,
			Type:        formatAppConfigSchemaFieldType(member.Type),
			Description: optionalString(member.Description),
			EnumItems:   toServerAppConfigSchemaEnumItems(findEnumSchema(member.Type, enumSchemas)),
		})
	}
	return fields
}

func formatAppConfigSchemaFieldType(typeSchema *skel.TypeSchema) string {
	if typeSchema == nil {
		return ""
	}
	var ret string
	switch typeSchema.Kind {
	case skel.TypeKindScalar:
		ret = string(typeSchema.Scalar)
	case skel.TypeKindEnum, skel.TypeKindData, skel.TypeKindConfig, skel.TypeKindEvent, skel.TypeKindTypeParameter:
		ret = formatAppConfigNamedType(typeSchema)
	case skel.TypeKindList:
		ret = "list<" + formatAppConfigSchemaFieldType(typeSchema.Element) + ">"
	case skel.TypeKindMap:
		ret = "map<" + formatAppConfigSchemaFieldType(typeSchema.Key) + ", " + formatAppConfigSchemaFieldType(typeSchema.Value) + ">"
	default:
		ret = string(typeSchema.Kind)
	}
	if typeSchema.Nullable {
		ret += "?"
	}
	return ret
}

func formatAppConfigNamedType(typeSchema *skel.TypeSchema) string {
	if typeSchema.SkelName != "" {
		return typeSchema.SkelName
	}
	return typeSchema.Name
}

func findEnumSchema(typeSchema *skel.TypeSchema, enumSchemas []*skel.EnumSchema) *skel.EnumSchema {
	if typeSchema == nil {
		return nil
	}
	if typeSchema.Kind == skel.TypeKindList {
		return findEnumSchema(typeSchema.Element, enumSchemas)
	}
	if typeSchema.Kind == skel.TypeKindMap {
		return findEnumSchema(typeSchema.Key, enumSchemas)
	}
	if typeSchema.Kind != skel.TypeKindEnum {
		return nil
	}
	for _, enumSchema := range enumSchemas {
		if enumSchema.SkelName == typeSchema.SkelName {
			return enumSchema
		}
	}
	return nil
}

func toServerAppConfigSchemaEnumItems(enumSchema *skel.EnumSchema) []skeled.AppConfigSchemaEnumItem {
	if enumSchema == nil {
		return []skeled.AppConfigSchemaEnumItem{}
	}
	items := make([]skeled.AppConfigSchemaEnumItem, 0, len(enumSchema.Items))
	for _, item := range enumSchema.Items {
		items = append(items, skeled.AppConfigSchemaEnumItem{
			Name:        item.Name,
			Description: optionalString(item.Description),
		})
	}
	return items
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
