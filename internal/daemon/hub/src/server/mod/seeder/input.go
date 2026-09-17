package seeder

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var seedVariable = regexp.MustCompile(`\$\{([^{}]*)\}`)
var seedVariableSegment = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// readSeedInput reads the seed YAML the caller embedded, or the file it named.
func readSeedInput(inline string, path string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	return []byte(inline), nil
}

// parseSeedNode parses the one configuration mapping a seed input carries, and
// rejects the YAML features a seed must not use.
func parseSeedNode(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("seed input must contain exactly one YAML document")
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("seed YAML must contain a configuration mapping (use {} for empty configuration)")
	}
	if err := checkSeedYAMLSyntax(&doc); err != nil {
		return nil, err
	}
	// Decode once to reject duplicate mapping keys before interpolation.
	var checked any
	if err := doc.Decode(&checked); err != nil {
		return nil, err
	}
	return doc.Content[0], nil
}

// escapeSeedPointer escapes one path segment the way a JSON pointer spells it.
func escapeSeedPointer(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

// seedMappingValue returns the value a mapping node gives to key, and nil when
// the node is not a mapping or does not carry the key.
func seedMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// walkSeedValues visits every value of node with its JSON pointer path, and
// rejects a variable reference in a mapping key.
func walkSeedValues(node *yaml.Node, path string, visit func(*yaml.Node, string) error) error {
	if err := visit(node, path); err != nil {
		return err
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if strings.Contains(node.Content[i].Value, "${") {
				return fmt.Errorf("variable references in mapping keys are not supported at %s", path)
			}
			if err := walkSeedValues(node.Content[i+1], path+"/"+escapeSeedPointer(node.Content[i].Value), visit); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			if err := walkSeedValues(child, path+"/"+strconv.Itoa(i), visit); err != nil {
				return err
			}
		}
	}
	return nil
}
