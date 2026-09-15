package core

import (
	"go.yorun.ai/vine/internal/core/skel"
)

// AppConfigDefinition is the resolved declaration of one application config.
type AppConfigDefinition struct {
	SkelName         string
	Name             string
	Description      string
	Deprecated       bool
	DeprecatedReason string
	Lifecycle        string
	Fields           []AppConfigField
}

// AppConfigField describes one config member and its resolved enum options.
type AppConfigField struct {
	Name              string
	Type              string
	Description       string
	Deprecated        bool
	DeprecatedReason  string
	EnumItems         []AppConfigEnumItem
	MapKeyEnumItems   []AppConfigEnumItem
	MapValueEnumItems []AppConfigEnumItem
}

// AppConfigEnumItem is one resolved enumeration option.
type AppConfigEnumItem struct {
	Name             string
	Description      string
	Deprecated       bool
	DeprecatedReason string
}

// NewAppConfigDefinition resolves the declaration of an application into the
// definition the Hub serves: field types and enum options are resolved here so
// no layer above has to join schemas again.
func NewAppConfigDefinition(schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema) *AppConfigDefinition {
	if schema == nil {
		return nil
	}
	return &AppConfigDefinition{
		SkelName:         schema.SkelName,
		Name:             schema.Name,
		Description:      schema.Description,
		Deprecated:       schema.Deprecated,
		DeprecatedReason: schema.DeprecatedReason,
		Lifecycle:        schema.Lifecycle,
		Fields:           newAppConfigFields(schema.Members, enumSchemas),
	}
}

func newAppConfigFields(members []*skel.MemberSchema, enumSchemas []*skel.EnumSchema) []AppConfigField {
	fields := make([]AppConfigField, 0, len(members))
	for _, member := range members {
		field := AppConfigField{
			Name:             member.Name,
			Type:             formatAppConfigFieldType(member.Type),
			Description:      member.Description,
			Deprecated:       member.Deprecated,
			DeprecatedReason: member.DeprecatedReason,
			EnumItems:        newAppConfigEnumItems(findEnumSchema(member.Type, enumSchemas)),
		}
		field.MapKeyEnumItems = []AppConfigEnumItem{}
		field.MapValueEnumItems = []AppConfigEnumItem{}
		if member.Type != nil && member.Type.Kind == skel.TypeKindMap {
			field.MapKeyEnumItems = newAppConfigEnumItems(findEnumSchema(member.Type.Key, enumSchemas))
			field.MapValueEnumItems = newAppConfigEnumItems(findEnumSchema(member.Type.Value, enumSchemas))
		}
		fields = append(fields, field)
	}
	return fields
}

func formatAppConfigFieldType(typeSchema *skel.TypeSchema) string {
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
		ret = "list<" + formatAppConfigFieldType(typeSchema.Element) + ">"
	case skel.TypeKindMap:
		ret = "map<" + formatAppConfigFieldType(typeSchema.Key) + ", " + formatAppConfigFieldType(typeSchema.Value) + ">"
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

func newAppConfigEnumItems(enumSchema *skel.EnumSchema) []AppConfigEnumItem {
	if enumSchema == nil {
		return []AppConfigEnumItem{}
	}
	items := make([]AppConfigEnumItem, 0, len(enumSchema.Items))
	for _, item := range enumSchema.Items {
		items = append(items, AppConfigEnumItem{
			Name:             item.Name,
			Description:      item.Description,
			Deprecated:       item.Deprecated,
			DeprecatedReason: item.DeprecatedReason,
		})
	}
	return items
}
