package seeder

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// applySeedVariables overlays literal assignments onto the file dictionary.
// Assigning an object replaces that subtree; assigning a child retains siblings.
func applySeedVariables(dictionary *yaml.Node, assignments []string) (*yaml.Node, error) {
	if len(assignments) == 0 {
		return dictionary, nil
	}
	if dictionary == nil {
		dictionary = new(yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"})
	}
	for _, assignment := range assignments {
		path, literal, found := strings.Cut(assignment, "=")
		if !found {
			return nil, fmt.Errorf("seed variable assignment must use path=YAML")
		}
		segments := strings.Split(path, ".")
		for _, segment := range segments {
			if !seedVariableSegment.MatchString(segment) {
				return nil, fmt.Errorf("seed variable %q must use camelCase path segments", path)
			}
		}
		value, err := parseSeedValue(literal)
		if err != nil {
			return nil, fmt.Errorf("seed variable %s: %w", path, err)
		}
		// Reject duplicate mapping keys, including in unused assignments.
		var checked any
		if err := value.Decode(&checked); err != nil {
			return nil, fmt.Errorf("seed variable %s: invalid YAML value", path)
		}
		current := dictionary
		for index, segment := range segments {
			if current.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("cannot assign seed variable %s: parent is not an object", path)
			}
			position := -1
			for i := 0; i < len(current.Content); i += 2 {
				if current.Content[i].Value == segment {
					position = i + 1
					break
				}
			}
			child := value
			if index < len(segments)-1 {
				child = new(yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"})
			}
			if position < 0 {
				current.Content = append(current.Content, new(yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: segment}), child)
			} else if index == len(segments)-1 {
				current.Content[position] = child
			} else {
				child = current.Content[position]
			}
			current = child
		}
	}
	return dictionary, nil
}
