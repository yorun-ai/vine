package seeder

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func validateWebTimeoutYAML(node *yaml.Node) error {
	fields, err := seedMappingFields(node, "webTimeout")
	if err != nil {
		return err
	}
	for name, value := range fields {
		switch name {
		case "default":
			if err := validateWebTimeoutPolicyYAML(&value, false); err != nil {
				return err
			}
		case "paths":
			if value.Kind != yaml.SequenceNode {
				return fmt.Errorf("webTimeout.paths must be a list")
			}
			for _, path := range value.Content {
				if err := validateWebTimeoutPolicyYAML(path, true); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unknown webTimeout field %q", name)
		}
	}
	return nil
}

func validateWebTimeoutPolicyYAML(node *yaml.Node, path bool) error {
	fields, err := seedMappingFields(node, "webTimeout policy")
	if err != nil {
		return err
	}
	for name, value := range fields {
		if name != "total" && !(path && name == "path") {
			return fmt.Errorf("unknown webTimeout policy field %q", name)
		}
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			return fmt.Errorf("webTimeout.%s must be a string", name)
		}
	}
	if _, ok := fields["total"]; !ok {
		return fmt.Errorf("webTimeout policy requires total")
	}
	if path {
		if _, ok := fields["path"]; !ok {
			return fmt.Errorf("webTimeout path policy requires path")
		}
	}
	return nil
}
