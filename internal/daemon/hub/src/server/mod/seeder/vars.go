package seeder

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"gopkg.in/yaml.v3"
)

// Generated skeled packages register these descriptors during Go initialization,
// before standalone starts Hub. The YAML tree remains authoritative for presence.
type _VarsDescriptor struct {
	data    map[string]*skeldesc.Data
	configs map[string]*skeldesc.Config
	enums   map[string]*skeldesc.Enum
}

func newVarsDescriptor(domains []*skeldesc.Domain) *_VarsDescriptor {
	result := new(_VarsDescriptor{
		data:    map[string]*skeldesc.Data{},
		configs: map[string]*skeldesc.Config{},
		enums:   map[string]*skeldesc.Enum{},
	})
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

func (s *_VarsDescriptor) members(kind *skeldesc.Type) ([]*skeldesc.Member, error) {
	switch kind.Kind {
	case skeldesc.TypeKindConfig:
		if item := s.configs[kind.SkelName]; item != nil {
			return item.Members, nil
		}
	case skeldesc.TypeKindData:
		if kind.SkelName == "seed.PortalCors" {
			return []*skeldesc.Member{
				{
					Name: "mode",
					Type: seedScalar(skeldesc.ScalarString),
				},
				{
					Name: "allowedOrigins",
					Type: new(skeldesc.Type{
						Kind:    skeldesc.TypeKindList,
						Element: seedScalar(skeldesc.ScalarString),
					}),
				},
			}, nil
		}
		if item := s.data[kind.SkelName]; item != nil {
			return item.Members, nil
		}
	}
	return nil, fmt.Errorf("missing seed type descriptor %s", kind.SkelName)
}

func (s *_VarsDescriptor) childType(kind *skeldesc.Type, key string) (*skeldesc.Type, error) {
	if kind == nil {
		return nil, nil
	}
	switch kind.Kind {
	case skeldesc.TypeKindMap:
		return kind.Value, nil
	case skeldesc.TypeKindList:
		return kind.Element, nil
	case skeldesc.TypeKindData, skeldesc.TypeKindConfig:
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

func (s *_VarsDescriptor) variableType(path string) (*skeldesc.Type, error) {
	if s.data["app.Vars"] == nil {
		return nil, nil
	}
	kind := new(skeldesc.Type{
		Kind:     skeldesc.TypeKindData,
		SkelName: "app.Vars",
	})
	var err error
	for segment := range strings.SplitSeq(path, ".") {
		kind, err = s.childType(kind, segment)
		if err != nil {
			return nil, err
		}
	}
	return kind, nil
}

func lookupSeedVariable(dictionary *yaml.Node, path string) (*yaml.Node, bool, error) {
	current := dictionary
	for segment := range strings.SplitSeq(path, ".") {
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

func seedScalar(scalar skeldesc.Scalar) *skeldesc.Type {
	return new(skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: scalar,
	})
}

// Application-point types come from registered config descriptors or the Hub seed
// contract. Standalone gets application descriptors from the imported skeled package.
func (s *_VarsDescriptor) targetType(root *yaml.Node, path string) (*skeldesc.Type, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		return nil, nil
	}
	section, field := parts[0], parts[2]

	var kind *skeldesc.Type
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
		kind = new(skeldesc.Type{
			Kind:     skeldesc.TypeKindConfig,
			SkelName: name.Value,
		})
	case section == "portalRules" && field == "matchPort":
		kind = seedScalar(skeldesc.ScalarInt)
	case section == "portalEntries" && field == "port":
		kind = seedScalar(skeldesc.ScalarInt)
	case field == "enabled" && (section == "portalSites" || section == "portalEntries" || section == "portalRules" || section == "portalCerts"):
		kind = seedScalar(skeldesc.ScalarBoolean)
	case section == "portalSites" && field == "cors":
		kind = new(skeldesc.Type{
			Kind:     skeldesc.TypeKindData,
			SkelName: "seed.PortalCors",
		})
	case section == "portalCerts" && field == "domains":
		kind = new(skeldesc.Type{
			Kind:    skeldesc.TypeKindList,
			Element: seedScalar(skeldesc.ScalarString),
		})
	case section == "portalCerts" && (field == "validFrom" || field == "validTo"):
		kind = seedScalar(skeldesc.ScalarTimestamp)
	default:
		if seedStringFields[section][field] {
			kind = seedScalar(skeldesc.ScalarString)
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

func parseSeedDefault(value string, kind *skeldesc.Type) (*yaml.Node, error) {
	// Text defaults (including empty strings, URLs, dates and enum names) retain
	// their literal spelling. Other defaults use YAML's native value syntax.
	if kind != nil && (kind.Kind == skeldesc.TypeKindEnum || (kind.Kind == skeldesc.TypeKindScalar && seedStringScalar(kind.Scalar))) {
		return new(yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: value,
		}), nil
	}
	return parseSeedValue(value)
}

// parseSeedValue preserves the native YAML type of one literal value.
func parseSeedValue(value string) (*yaml.Node, error) {
	if value == "" {
		return new(yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "",
		}), nil
	}
	var doc yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid seed value")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("seed value must contain one YAML value")
	}
	if len(doc.Content) != 1 {
		return nil, fmt.Errorf("invalid seed value")
	}
	if err := checkSeedYAMLSyntax(&doc); err != nil {
		return nil, fmt.Errorf("invalid seed value syntax")
	}
	return doc.Content[0], nil
}

func seedStringScalar(scalar skeldesc.Scalar) bool {
	switch scalar {
	case skeldesc.ScalarBoolean, skeldesc.ScalarInt, skeldesc.ScalarFloat:
		return false
	default:
		return true
	}
}

// validateValue validates only the value being applied. It never fills absent
// fields with zero values, and ignores extra object fields rather than copying
// them into the effective configuration.
func (s *_VarsDescriptor) validateValue(node *yaml.Node, kind *skeldesc.Type, path string, required bool) (*yaml.Node, error) {
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
	case skeldesc.TypeKindData, skeldesc.TypeKindConfig:
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
			result.Content = append(result.Content, new(yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: member.Name,
			}), checked)
		}
	case skeldesc.TypeKindList:
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
	case skeldesc.TypeKindMap:
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
	case skeldesc.TypeKindEnum:
		enum := s.enums[kind.SkelName]
		if enum == nil {
			return nil, fmt.Errorf("%s: enum descriptor %s not found", path, kind.SkelName)
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
	case skeldesc.TypeKindScalar:
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

func validateSeedScalar(node *yaml.Node, scalar skeldesc.Scalar) error {
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
	case skeldesc.ScalarString:
		target = new(string)
	case skeldesc.ScalarBoolean:
		target = new(bool)
	case skeldesc.ScalarInt:
		target = new(int)
	case skeldesc.ScalarFloat:
		target = new(float64)
	case skeldesc.ScalarDecimal:
		target = new(skeltype.Decimal)
	case skeldesc.ScalarJSON:
		target = new(skeltype.JSON)
	case skeldesc.ScalarUUID:
		target = new(skeltype.UUID)
	case skeldesc.ScalarTimestamp:
		target = new(skeltype.Timestamp)
	case skeldesc.ScalarDuration:
		target = new(skeltype.Duration)
	case skeldesc.ScalarLocalDate:
		target = new(skeltype.LocalDate)
	case skeldesc.ScalarLocalTime:
		target = new(skeltype.LocalTime)
	case skeldesc.ScalarLocalDateTime:
		target = new(skeltype.LocalDateTime)
	case skeldesc.ScalarBinary:
		target = new(skeltype.Binary)
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
