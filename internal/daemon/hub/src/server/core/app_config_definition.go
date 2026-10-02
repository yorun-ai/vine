package core

import (
	"strings"

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
	Sensitive        bool
	DataTypes        []AppConfigData
}

// AppConfigField describes one config member and its resolved enum options.
type AppConfigField struct {
	Name              string
	ValueType         *AppConfigType
	Sensitive         bool
	Example           string
	Type              string
	Description       string
	Deprecated        bool
	DeprecatedReason  string
	EnumItems         []AppConfigEnumItem
	MapKeyEnumItems   []AppConfigEnumItem
	MapValueEnumItems []AppConfigEnumItem
}

// AppConfigType preserves the shape of a value type without expanding recursive data.
type AppConfigType struct {
	Kind          string
	Nullable      bool
	Name          string
	TypeArguments []*AppConfigType
	Element       *AppConfigType
	Key           *AppConfigType
	Value         *AppConfigType
	EnumItems     []AppConfigEnumItem
}

// AppConfigData describes a reusable data declaration reachable from a config.
type AppConfigData struct {
	Name             string
	SkelName         string
	Description      string
	Deprecated       bool
	DeprecatedReason string
	Sensitive        bool
	TypeParameters   []string
	Fields           []AppConfigField
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
func NewAppConfigDefinition(schema *skel.ConfigSchema, enumSchemas []*skel.EnumSchema, dataSchemas []*skel.DataSchema) *AppConfigDefinition {
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
		Sensitive:        schema.Sensitive,
		DataTypes:        appConfigDataTypes(schema.Members, enumSchemas, dataSchemas),
		Fields:           newAppConfigFields(schema.Members, enumSchemas),
	}
}

func newAppConfigFields(members []*skel.MemberSchema, enumSchemas []*skel.EnumSchema) []AppConfigField {
	fields := make([]AppConfigField, 0, len(members))
	for _, member := range members {
		field := AppConfigField{
			Name:             member.Name,
			ValueType:        newAppConfigType(member.Type, enumSchemas),
			Sensitive:        member.Sensitive,
			Example:          member.Example,
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
		return appConfigTypeArguments(typeSchema.SkelName, typeSchema.TypeArguments)
	}
	return appConfigTypeArguments(typeSchema.Name, typeSchema.TypeArguments)
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

func appConfigTypeArguments(name string, arguments []*skel.TypeSchema) string {
	if len(arguments) == 0 {
		return name
	}
	args := make([]string, len(arguments))
	for i, arg := range arguments {
		args[i] = formatAppConfigFieldType(arg)
	}
	return name + "<" + strings.Join(args, ", ") + ">"
}

func newAppConfigType(kind *skel.TypeSchema, enums []*skel.EnumSchema) *AppConfigType {
	if kind == nil {
		return nil
	}
	name := kind.SkelName
	if name == "" {
		name = kind.Name
	}
	if kind.Kind == skel.TypeKindScalar {
		name = string(kind.Scalar)
	}
	result := &AppConfigType{Kind: string(kind.Kind), Name: name, Nullable: kind.Nullable,
		Element: newAppConfigType(kind.Element, enums), Key: newAppConfigType(kind.Key, enums), Value: newAppConfigType(kind.Value, enums),
		TypeArguments: []*AppConfigType{}, EnumItems: []AppConfigEnumItem{}}
	if kind.Kind == skel.TypeKindEnum {
		result.EnumItems = newAppConfigEnumItems(findEnumSchema(kind, enums))
	}
	for _, arg := range kind.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, newAppConfigType(arg, enums))
	}
	return result
}

func appConfigDataTypes(members []*skel.MemberSchema, enums []*skel.EnumSchema, schemas []*skel.DataSchema) []AppConfigData {
	index := map[string]*skel.DataSchema{}
	for _, data := range schemas {
		index[data.SkelName] = data
	}
	result := []AppConfigData{}
	seen := map[string]bool{}
	var visit func(*skel.TypeSchema)
	visit = func(kind *skel.TypeSchema) {
		if kind == nil {
			return
		}
		for _, arg := range kind.TypeArguments {
			visit(arg)
		}
		visit(kind.Element)
		visit(kind.Key)
		visit(kind.Value)
		if kind.Kind != skel.TypeKindData || seen[kind.SkelName] {
			return
		}
		seen[kind.SkelName] = true
		data := index[kind.SkelName]
		if data == nil {
			return
		}
		result = append(result, AppConfigData{Name: data.Name, SkelName: data.SkelName, Description: data.Description,
			Deprecated: data.Deprecated, DeprecatedReason: data.DeprecatedReason, Sensitive: data.Sensitive,
			TypeParameters: data.TypeParameters, Fields: newAppConfigFields(data.Members, enums)})
		for _, member := range data.Members {
			visit(member.Type)
		}
	}
	for _, member := range members {
		visit(member.Type)
	}
	return result
}
