package seeder

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"gopkg.in/yaml.v3"
)

// Generated skeled packages register these schemas during Go initialization,
// before standalone starts Hub. The YAML tree remains authoritative for presence.
type _VarsSchema struct {
	data    map[string]*skel.DataSchema
	configs map[string]*skel.ConfigSchema
	enums   map[string]*skel.EnumSchema
}

func newVarsSchema(domains []*skel.DomainSchema) *_VarsSchema {
	result := new(_VarsSchema{data: map[string]*skel.DataSchema{}, configs: map[string]*skel.ConfigSchema{}, enums: map[string]*skel.EnumSchema{}})
	for _, domain := range domains {
		for _, item := range domain.Data {
			result.data[item.SkelName] = item
		}
		for _, item := range domain.Configs {
			result.configs[item.SkelName] = item
		}
		for _, item := range domain.Enums {
			result.enums[item.SkelName] = item
		}
	}
	return result
}

func (s *_VarsSchema) members(kind *skel.TypeSchema) ([]*skel.MemberSchema, error) {
	switch kind.Kind {
	case skel.TypeKindConfig:
		if item := s.configs[kind.SkelName]; item != nil {
			return item.Members, nil
		}
	case skel.TypeKindData:
		if kind.SkelName == "seed.PortalCors" {
			return []*skel.MemberSchema{
				{Name: "mode", Type: seedScalar(skel.ScalarString)},
				{Name: "allowedOrigins", Type: new(skel.TypeSchema{Kind: skel.TypeKindList, Element: seedScalar(skel.ScalarString)})},
			}, nil
		}
		if item := s.data[kind.SkelName]; item != nil {
			return item.Members, nil
		}
	}
	return nil, fmt.Errorf("missing seed type schema %s", kind.SkelName)
}

func (s *_VarsSchema) childType(kind *skel.TypeSchema, key string) (*skel.TypeSchema, error) {
	if kind == nil {
		return nil, nil
	}
	switch kind.Kind {
	case skel.TypeKindMap:
		return kind.Value, nil
	case skel.TypeKindList:
		return kind.Element, nil
	case skel.TypeKindData, skel.TypeKindConfig:
		members, err := s.members(kind)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			if member.Name == key {
				return member.Type, nil
			}
		}
		return nil, fmt.Errorf("field %s not defined in %s", key, kind.SkelName)
	default:
		return nil, fmt.Errorf("cannot access field %s on a scalar seed variable", key)
	}
}

func (s *_VarsSchema) variableType(path string) (*skel.TypeSchema, error) {
	if s.data["app.Vars"] == nil {
		return nil, nil
	}
	kind := new(skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "app.Vars"})
	var err error
	for _, segment := range strings.Split(path, ".") {
		kind, err = s.childType(kind, segment)
		if err != nil {
			return nil, err
		}
	}
	return kind, nil
}

func lookupSeedVariable(dictionary *yaml.Node, path string) (*yaml.Node, bool, error) {
	current := dictionary
	for _, segment := range strings.Split(path, ".") {
		if current == nil {
			return nil, false, nil
		}
		if current.Kind != yaml.MappingNode {
			return nil, false, fmt.Errorf("cannot access %s in seed variable %s: parent is not an object", segment, path)
		}
		current = seedMappingValue(current, segment)
	}
	return current, current != nil, nil
}

func seedScalar(scalar skel.Scalar) *skel.TypeSchema {
	return new(skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: scalar})
}

// Application-point types come from registered config schemas or the Hub seed
// contract. Standalone gets application schemas from the imported skeled package.
func (s *_VarsSchema) targetType(root *yaml.Node, path string) (*skel.TypeSchema, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		return nil, nil
	}
	section, field := parts[0], parts[2]
	if section == "portalRules" {
		if canonical, ok := portalRuleAliases[field]; ok {
			field = canonical
		}
	}
	var kind *skel.TypeSchema
	switch {
	case section == "appConfigs" && field == "value":
		items := seedMappingValue(root, section)
		index, err := strconv.Atoi(parts[1])
		if err != nil || items == nil || index >= len(items.Content) {
			return nil, fmt.Errorf("invalid seed application point %s", path)
		}
		name := seedMappingValue(items.Content[index], "name")
		if name == nil || s.configs[name.Value] == nil {
			return nil, nil
		}
		kind = new(skel.TypeSchema{Kind: skel.TypeKindConfig, SkelName: name.Value})
	case section == "portalRules" && field == "matchPort":
		kind = seedScalar(skel.ScalarInt)
	case section == "portalSites" && field == "cors":
		kind = new(skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "seed.PortalCors"})
	case section == "portalCerts" && field == "domains":
		kind = new(skel.TypeSchema{Kind: skel.TypeKindList, Element: seedScalar(skel.ScalarString)})
	case section == "portalCerts" && (field == "validFrom" || field == "validTo"):
		kind = seedScalar(skel.ScalarTimestamp)
	default:
		if seedStringFields[section][field] {
			kind = seedScalar(skel.ScalarString)
		}
	}
	for _, key := range parts[3:] {
		key = strings.ReplaceAll(strings.ReplaceAll(key, "~1", "/"), "~0", "~")
		var err error
		kind, err = s.childType(kind, key)
		if err != nil {
			return nil, err
		}
	}
	return kind, nil
}

func parseSeedDefault(value string, kind *skel.TypeSchema) (*yaml.Node, error) {
	// Text defaults (including empty strings, URLs, dates and enum names) retain
	// their literal spelling. Other defaults use YAML's native value syntax.
	if kind != nil && (kind.Kind == skel.TypeKindEnum || (kind.Kind == skel.TypeKindScalar && seedStringScalar(kind.Scalar))) {
		return new(yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}), nil
	}
	if value == "" {
		return new(yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ""}), nil
	}
	var doc yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid seed default")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("seed default must contain one YAML value")
	}
	if len(doc.Content) != 1 {
		return nil, fmt.Errorf("invalid seed default")
	}
	if err := checkSeedYAMLSyntax(&doc); err != nil {
		return nil, fmt.Errorf("invalid seed default syntax")
	}
	return doc.Content[0], nil
}

func seedStringScalar(scalar skel.Scalar) bool {
	switch scalar {
	case skel.ScalarBool, skel.ScalarInt, skel.ScalarLong, skel.ScalarFloat, skel.ScalarDouble:
		return false
	default:
		return true
	}
}

// validateValue validates only the value being applied. It never fills absent
// fields with zero values, and ignores extra object fields rather than copying
// them into the effective configuration.
func (s *_VarsSchema) validateValue(node *yaml.Node, kind *skel.TypeSchema, path string, required bool) (*yaml.Node, error) {
	if kind == nil {
		return cloneSeedNode(node), nil
	}
	if node.ShortTag() == "!!null" {
		if required && !kind.Nullable {
			return nil, fmt.Errorf("%s: null is not allowed", path)
		}
		return cloneSeedNode(node), nil
	}
	result := *node
	result.Content = nil
	switch kind.Kind {
	case skel.TypeKindData, skel.TypeKindConfig:
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s: expected object", path)
		}
		members, err := s.members(kind)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			value := seedMappingValue(node, member.Name)
			if value == nil {
				if !required || member.Type.Nullable {
					continue
				}
				return nil, fmt.Errorf("%s.%s: required field is missing", path, member.Name)
			}
			checked, err := s.validateValue(value, member.Type, path+"."+member.Name, required)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, new(yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: member.Name}), checked)
		}
	case skel.TypeKindList:
		if node.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("%s: expected list", path)
		}
		for i, value := range node.Content {
			checked, err := s.validateValue(value, kind.Element, fmt.Sprintf("%s[%d]", path, i), required)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, checked)
		}
	case skel.TypeKindMap:
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s: expected map", path)
		}
		for i := 0; i < len(node.Content); i += 2 {
			key, err := s.validateValue(node.Content[i], kind.Key, path+" map key", required)
			if err != nil {
				return nil, err
			}
			value, err := s.validateValue(node.Content[i+1], kind.Value, path+"."+node.Content[i].Value, required)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, key, value)
		}
	case skel.TypeKindEnum:
		enum := s.enums[kind.SkelName]
		if enum == nil {
			return nil, fmt.Errorf("%s: enum schema %s not found", path, kind.SkelName)
		}
		valid := false
		for _, item := range enum.Items {
			if item.Name == node.Value && node.ShortTag() == "!!str" {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("%s: invalid %s enum value", path, kind.SkelName)
		}
	case skel.TypeKindScalar:
		if err := validateSeedScalar(node, kind.Scalar); err != nil {
			return nil, fmt.Errorf("%s: expected %s", path, kind.Scalar)
		}
		if node.ShortTag() == "!!timestamp" {
			result.Tag = "!!str"
		}
		result.Content = node.Content
	default:
		return nil, fmt.Errorf("%s: unsupported seed type %s", path, kind.Kind)
	}
	return &result, nil
}

func validateSeedScalar(node *yaml.Node, scalar skel.Scalar) error {
	normalized, err := configJSONNode(node)
	if err != nil {
		return err
	}
	var value any
	if err := normalized.Decode(&value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var target any
	switch scalar {
	case skel.ScalarString:
		target = new(string)
	case skel.ScalarBool:
		target = new(bool)
	case skel.ScalarInt:
		target = new(int)
	case skel.ScalarLong:
		target = new(int64)
	case skel.ScalarFloat:
		target = new(float32)
	case skel.ScalarDouble:
		target = new(float64)
	case skel.ScalarDecimal:
		target = new(skel.Decimal)
	case skel.ScalarJson:
		target = new(skel.JSON)
	case skel.ScalarUuid:
		target = new(skel.UUID)
	case skel.ScalarTimestamp:
		target = new(skel.Timestamp)
	case skel.ScalarDuration:
		target = new(skel.Duration)
	case skel.ScalarLocalDate:
		target = new(skel.LocalDate)
	case skel.ScalarLocalTime:
		target = new(skel.LocalTime)
	case skel.ScalarLocalDateTime:
		target = new(skel.LocalDateTime)
	case skel.ScalarBinary:
		target = new(skel.Binary)
	default:
		return fmt.Errorf("unsupported scalar %s", scalar)
	}
	return json.Unmarshal(encoded, target)
}

func cloneSeedNode(node *yaml.Node) *yaml.Node {
	result := *node
	result.Content = nil
	for _, child := range node.Content {
		result.Content = append(result.Content, cloneSeedNode(child))
	}
	return &result
}
