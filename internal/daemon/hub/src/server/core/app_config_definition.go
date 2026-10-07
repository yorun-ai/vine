package core

import (
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
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
// no layer above has to join descriptors again.
func NewAppConfigDefinition(descriptor *skeldesc.Config, enumDescriptors []*skeldesc.Enum, dataDescriptors []*skeldesc.Data) *AppConfigDefinition {
	if descriptor == nil {
		return nil
	}
	return &AppConfigDefinition{
		SkelName:         descriptor.SkelName,
		Name:             descriptor.Name,
		Description:      descriptor.Description,
		Deprecated:       descriptor.Deprecated,
		DeprecatedReason: descriptor.DeprecatedReason,
		Lifecycle:        string(descriptor.Lifecycle),
		Sensitive:        descriptor.Sensitive,
		DataTypes:        appConfigDataTypes(descriptor.Members, enumDescriptors, dataDescriptors),
		Fields:           newAppConfigFields(descriptor.Members, enumDescriptors),
	}
}

func newAppConfigFields(members []*skeldesc.Member, enumDescriptors []*skeldesc.Enum) []AppConfigField {
	fields := make([]AppConfigField, 0, len(members))
	for _, member := range members {
		field := AppConfigField{
			Name:             member.Name,
			ValueType:        newAppConfigType(member.Type, enumDescriptors),
			Sensitive:        member.Sensitive,
			Example:          member.Example,
			Type:             formatAppConfigFieldType(member.Type),
			Description:      member.Description,
			Deprecated:       member.Deprecated,
			DeprecatedReason: member.DeprecatedReason,
			EnumItems:        newAppConfigEnumItems(findEnumDescriptor(member.Type, enumDescriptors)),
		}
		field.MapKeyEnumItems = []AppConfigEnumItem{}
		field.MapValueEnumItems = []AppConfigEnumItem{}
		if member.Type != nil && member.Type.Kind == skeldesc.TypeKindMap {
			field.MapKeyEnumItems = newAppConfigEnumItems(findEnumDescriptor(member.Type.Key, enumDescriptors))
			field.MapValueEnumItems = newAppConfigEnumItems(findEnumDescriptor(member.Type.Value, enumDescriptors))
		}
		fields = append(fields, field)
	}
	return fields
}

func formatAppConfigFieldType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor == nil {
		return ""
	}
	var ret string
	switch typeDescriptor.Kind {
	case skeldesc.TypeKindScalar:
		ret = string(typeDescriptor.Scalar)
	case skeldesc.TypeKindEnum, skeldesc.TypeKindData, skeldesc.TypeKindConfig, skeldesc.TypeKindEvent, skeldesc.TypeKindTypeParameter:
		ret = formatAppConfigNamedType(typeDescriptor)
	case skeldesc.TypeKindList:
		ret = "list<" + formatAppConfigFieldType(typeDescriptor.Element) + ">"
	case skeldesc.TypeKindMap:
		ret = "map<" + formatAppConfigFieldType(typeDescriptor.Key) + ", " + formatAppConfigFieldType(typeDescriptor.Value) + ">"
	default:
		ret = string(typeDescriptor.Kind)
	}
	if typeDescriptor.Nullable {
		ret += "?"
	}
	return ret
}

func formatAppConfigNamedType(typeDescriptor *skeldesc.Type) string {
	if typeDescriptor.SkelName != "" {
		return appConfigTypeArguments(typeDescriptor.SkelName, typeDescriptor.TypeArguments)
	}
	return appConfigTypeArguments(typeDescriptor.Name, typeDescriptor.TypeArguments)
}

func findEnumDescriptor(typeDescriptor *skeldesc.Type, enumDescriptors []*skeldesc.Enum) *skeldesc.Enum {
	if typeDescriptor == nil {
		return nil
	}
	if typeDescriptor.Kind == skeldesc.TypeKindList {
		return findEnumDescriptor(typeDescriptor.Element, enumDescriptors)
	}
	if typeDescriptor.Kind == skeldesc.TypeKindMap {
		return findEnumDescriptor(typeDescriptor.Key, enumDescriptors)
	}
	if typeDescriptor.Kind != skeldesc.TypeKindEnum {
		return nil
	}
	for _, enumDescriptor := range enumDescriptors {
		if enumDescriptor.SkelName == typeDescriptor.SkelName {
			return enumDescriptor
		}
	}
	return nil
}

func newAppConfigEnumItems(enumDescriptor *skeldesc.Enum) []AppConfigEnumItem {
	if enumDescriptor == nil {
		return []AppConfigEnumItem{}
	}
	items := make([]AppConfigEnumItem, 0, len(enumDescriptor.Items))
	for _, item := range enumDescriptor.Items {
		items = append(items, AppConfigEnumItem{
			Name:             item.Name,
			Description:      item.Description,
			Deprecated:       item.Deprecated,
			DeprecatedReason: item.DeprecatedReason,
		})
	}
	return items
}

func appConfigTypeArguments(name string, arguments []*skeldesc.Type) string {
	if len(arguments) == 0 {
		return name
	}
	args := make([]string, len(arguments))
	for i, arg := range arguments {
		args[i] = formatAppConfigFieldType(arg)
	}
	return name + "<" + strings.Join(args, ", ") + ">"
}

func newAppConfigType(kind *skeldesc.Type, enums []*skeldesc.Enum) *AppConfigType {
	if kind == nil {
		return nil
	}
	name := kind.SkelName
	if name == "" {
		name = kind.Name
	}
	if kind.Kind == skeldesc.TypeKindScalar {
		name = string(kind.Scalar)
	}
	result := &AppConfigType{
		Kind:          string(kind.Kind),
		Name:          name,
		Nullable:      kind.Nullable,
		Element:       newAppConfigType(kind.Element, enums),
		Key:           newAppConfigType(kind.Key, enums),
		Value:         newAppConfigType(kind.Value, enums),
		TypeArguments: []*AppConfigType{},
		EnumItems:     []AppConfigEnumItem{},
	}
	if kind.Kind == skeldesc.TypeKindEnum {
		result.EnumItems = newAppConfigEnumItems(findEnumDescriptor(kind, enums))
	}
	for _, arg := range kind.TypeArguments {
		result.TypeArguments = append(result.TypeArguments, newAppConfigType(arg, enums))
	}
	return result
}

func appConfigDataTypes(members []*skeldesc.Member, enums []*skeldesc.Enum, descriptors []*skeldesc.Data) []AppConfigData {
	index := map[string]*skeldesc.Data{}
	for _, data := range descriptors {
		index[data.SkelName] = data
	}
	result := []AppConfigData{}
	seen := map[string]bool{}
	var visit func(*skeldesc.Type)
	visit = func(kind *skeldesc.Type) {
		if kind == nil {
			return
		}
		for _, arg := range kind.TypeArguments {
			visit(arg)
		}
		visit(kind.Element)
		visit(kind.Key)
		visit(kind.Value)
		if kind.Kind != skeldesc.TypeKindData || seen[kind.SkelName] {
			return
		}
		seen[kind.SkelName] = true
		data := index[kind.SkelName]
		if data == nil {
			return
		}
		result = append(result, AppConfigData{
			Name:             data.Name,
			SkelName:         data.SkelName,
			Description:      data.Description,
			Deprecated:       data.Deprecated,
			DeprecatedReason: data.DeprecatedReason,
			Sensitive:        data.Sensitive,
			TypeParameters:   data.TypeParameters,
			Fields:           newAppConfigFields(data.Members, enums),
		})
		for _, member := range data.Members {
			visit(member.Type)
		}
	}
	for _, member := range members {
		visit(member.Type)
	}
	return result
}
