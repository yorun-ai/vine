package seeder

import (
	"encoding/json/v2"
	"fmt"

	"gopkg.in/yaml.v3"
)

// DecodeAppConfig accepts structured YAML values and legacy JSON strings.
func DecodeAppConfig(node *yaml.Node, target any) error {
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return err
	}
	if value, ok := fields["value"]; ok && value.ShortTag() != "!!str" {
		normalized, err := configJSONNode(&value, map[*yaml.Node]*yaml.Node{})
		if err != nil {
			return fmt.Errorf("app config %q value: %w", fields["name"].Value, err)
		}
		var decoded any
		if err := normalized.Decode(&decoded); err != nil {
			return fmt.Errorf("app config %q value: %w", fields["name"].Value, err)
		}
		data, err := json.Marshal(decoded, json.Deterministic(true))
		if err != nil {
			return fmt.Errorf("app config %q value: %w", fields["name"].Value, err)
		}
		fields["value"] = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: string(data)}
	}
	normalized := *node
	normalized.Content = nil
	for name, value := range fields {
		normalized.Content = append(normalized.Content,
			new(yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}), &value)
	}
	return normalized.Decode(target)
}

func configJSONNode(node *yaml.Node, copies map[*yaml.Node]*yaml.Node) (*yaml.Node, error) {
	if copy, ok := copies[node]; ok {
		if copy == nil {
			return nil, fmt.Errorf("cyclic YAML aliases are not supported")
		}
		return copy, nil
	}
	copies[node] = nil
	copy := *node
	copy.Content = nil
	if node.Kind == yaml.AliasNode {
		alias, err := configJSONNode(node.Alias, copies)
		if err != nil {
			return nil, err
		}
		copy.Alias = alias
	} else {
		switch node.ShortTag() {
		case "!!timestamp":
			copy.Tag = "!!str"
		case "!!str", "!!bool", "!!int", "!!float", "!!null", "!!map", "!!seq", "!!merge":
		default:
			return nil, fmt.Errorf("unsupported YAML tag %q", node.Tag)
		}
		for index, child := range node.Content {
			normalized, err := configJSONNode(child, copies)
			if err != nil {
				return nil, err
			}
			if node.Kind == yaml.MappingNode && index%2 == 0 && child.ShortTag() != "!!merge" {
				if child.Kind != yaml.ScalarNode || child.ShortTag() == "!!null" {
					return nil, fmt.Errorf("configuration map keys must be non-null scalars")
				}
				key := *normalized
				key.Tag = "!!str"
				normalized = &key
			}
			copy.Content = append(copy.Content, normalized)
		}
	}
	copies[node] = &copy
	return &copy, nil
}
