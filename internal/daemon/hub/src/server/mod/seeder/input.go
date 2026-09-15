package seeder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

type _SeedHubSourceFile struct {
	Version    int               `yaml:"version"`
	SeedSHA256 string            `yaml:"seedSha256"`
	Fields     core.FieldSources `yaml:"fields"`
}

var seedVariable = regexp.MustCompile(`\$\{([^{}]*)\}`)
var seedVariableSegment = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func readSeedInput(inline string, path string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	return []byte(inline), nil
}
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
func escapeSeedPointer(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}
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

// resolveSeedInput preserves YAML types for whole-field references. Substituted
// variable values are literal data, not recursively evaluated templates.
func resolveSeedInput(template []byte, variables []byte, source []byte) (*yaml.Node, core.FieldSources, error) {
	return resolveSeedInputWithSchemas(template, variables, source, skel.RegisteredDomainSchemas())
}

func resolveSeedInputWithSchemas(template []byte, variables []byte, source []byte, domains []*skel.DomainSchema) (*yaml.Node, core.FieldSources, error) {
	schema := newVarsSchema(domains)
	node, err := parseSeedNode(template)
	if err != nil {
		return nil, nil, err
	}
	originals := map[string]*yaml.Node{}
	if err := walkSeedValues(cloneSeedNode(node), "", func(n *yaml.Node, path string) error { originals[path] = n; return nil }); err != nil {
		return nil, nil, err
	}
	sources := core.FieldSources{}
	paths := map[string]bool{}
	if err := walkSeedValues(node, "", func(_ *yaml.Node, path string) error { paths[path] = true; return nil }); err != nil {
		return nil, nil, err
	}
	if len(source) > 0 {
		if _, err := parseSeedNode(source); err != nil {
			return nil, nil, fmt.Errorf("seed source: %w", err)
		}
		var file _SeedHubSourceFile
		decoder := yaml.NewDecoder(bytes.NewReader(source))
		decoder.KnownFields(true)
		if err := decoder.Decode(&file); err != nil {
			return nil, nil, fmt.Errorf("seed source: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, nil, fmt.Errorf("seed source must contain one document")
		}
		if file.Version != 1 {
			return nil, nil, fmt.Errorf("unsupported seed source version %d", file.Version)
		}
		if file.SeedSHA256 != fmt.Sprintf("%x", sha256.Sum256(template)) {
			return nil, nil, fmt.Errorf("seed source digest does not match seed template")
		}
		for path, origin := range file.Fields {
			parts := strings.Split(path, "/")
			if !paths[path] || len(parts) < 4 || origin.Define == "" || origin.Source == "" || (parts[1] == "appConfigs" && parts[3] == "value" && len(parts) > 5) {
				return nil, nil, fmt.Errorf("invalid seed source field %s", path)
			}
			if len(origin.Variables) > 0 || origin.Template != nil || len(origin.Bindings) > 0 {
				return nil, nil, fmt.Errorf("substitution metadata must be derived from the seed template: %s", path)
			}
			sources[path] = origin
		}
	}
	var dictionary *yaml.Node
	if len(variables) > 0 {
		dictionary, err = parseSeedNode(variables)
		if err != nil {
			return nil, nil, fmt.Errorf("seed variables: %w", err)
		}
	}
	// Resolve only original template nodes, so inserted values remain literal.
	var resolve func(*yaml.Node, string) error
	resolve = func(n *yaml.Node, path string) error {
		location := seedLocation(node, path)
		if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
			matches := seedVariable.FindAllStringSubmatch(n.Value, -1)
			if strings.Contains(seedVariable.ReplaceAllString(n.Value, ""), "${") {
				return fmt.Errorf("malformed seed variable at %s", location)
			}
			if len(matches) == 0 {
				return nil
			}
			whole := len(matches) == 1 && matches[0][0] == n.Value
			target, err := schema.targetType(node, path)
			if err != nil {
				return fmt.Errorf("seed application point %s: %w", location, err)
			}
			jsonStringConfig := !whole && target != nil && target.Kind == skel.TypeKindConfig
			if !whole && !jsonStringConfig && target != nil && (target.Kind != skel.TypeKindScalar || target.Scalar != skel.ScalarString) {
				return fmt.Errorf("string interpolation requires a string target at %s", location)
			}
			dependencies := map[string]bool{}
			replacements := map[string]*yaml.Node{}
			bindings := []core.FieldSourceBinding{}
			for _, m := range matches {
				name, fallback, hasDefault := strings.Cut(m[1], ":")
				for _, segment := range strings.Split(name, ".") {
					if !seedVariableSegment.MatchString(segment) {
						return fmt.Errorf("seed variable %s must use camelCase path segments at %s", name, location)
					}
				}
				kind, err := schema.variableType(name)
				if err != nil {
					return fmt.Errorf("seed variable %s at %s: %w", name, location, err)
				}
				value, found, err := lookupSeedVariable(dictionary, name)
				if err != nil {
					return fmt.Errorf("seed variable at %s: %w", location, err)
				}
				if !found {
					if !hasDefault {
						return fmt.Errorf("variable %q is missing and has no default", name)
					}
					defaultType := kind
					if !whole {
						defaultType = seedScalar(skel.ScalarString)
					} else if defaultType == nil {
						defaultType = target
					}
					value, err = parseSeedDefault(fallback, defaultType)
					if err != nil {
						return fmt.Errorf("seed variable %s at %s: %w", name, location, err)
					}
				}
				// Interpolation defaults are already text; supplied variables
				// still obey their declared type. Null is preserved until use.
				if whole || found {
					value, err = schema.validateValue(value, kind, name, false)
					if err != nil {
						return fmt.Errorf("seed variable at %s: %w", location, err)
					}
				}
				if whole {
					value, err = schema.validateValue(value, target, location, true)
					if err != nil {
						return err
					}
				} else if value.Kind != yaml.ScalarNode || value.ShortTag() == "!!null" {
					return fmt.Errorf("seed variable %s must be a non-null scalar for interpolation at %s", name, location)
				}
				encoded, err := seedNodeJSON(value)
				if err != nil {
					return fmt.Errorf("cannot record variable %q: %w", name, err)
				}
				bindings = append(bindings, core.FieldSourceBinding{Variable: name, Reference: m[0], Value: encoded, DefaultUsed: !found})
				dependencies[name] = true
				replacements[m[0]] = value
			}
			sourcePath := path
			parts := strings.Split(path, "/")
			if len(parts) > 5 && parts[1] == "appConfigs" && parts[3] == "value" {
				sourcePath = strings.Join(parts[:5], "/")
			}
			origin := sources[sourcePath]
			if origin.Template == nil {
				original, err := seedNodeJSON(originals[sourcePath])
				if err != nil {
					return fmt.Errorf("cannot record template at %s: %w", location, err)
				}
				origin.Template = &original
			}
			for _, binding := range bindings {
				binding.Path = strings.TrimPrefix(path, sourcePath)
				origin.Bindings = append(origin.Bindings, binding)
			}
			for _, name := range origin.Variables {
				dependencies[name] = true
			}
			origin.Variables = nil
			for name := range dependencies {
				origin.Variables = append(origin.Variables, name)
			}
			sort.Strings(origin.Variables)
			sources[sourcePath] = origin
			if whole {
				*n = *cloneSeedNode(replacements[matches[0][0]])
				return nil
			}
			n.Value = seedVariable.ReplaceAllStringFunc(n.Value, func(ref string) string { return replacements[ref].Value })
			if jsonStringConfig {
				if !jsontext.Value(n.Value).IsValid() {
					return fmt.Errorf("invalid interpolated config JSON at %s", location)
				}
				var doc yaml.Node
				if err := yaml.Unmarshal([]byte(n.Value), &doc); err != nil {
					return fmt.Errorf("invalid interpolated config JSON at %s", location)
				}
				checked, err := schema.validateValue(doc.Content[0], target, location, true)
				if err != nil {
					return err
				}
				*n = *checked
			}
			return nil
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i < len(n.Content); i += 2 {
				if err := resolve(n.Content[i+1], path+"/"+escapeSeedPointer(n.Content[i].Value)); err != nil {
					return err
				}
			}
		}
		if n.Kind == yaml.SequenceNode {
			for i, c := range n.Content {
				if err := resolve(c, path+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := resolve(node, ""); err != nil {
		return nil, nil, err
	}
	return node, sources, nil
}

// seedLocation presents entity names and fields instead of internal JSON pointers.
func seedLocation(root *yaml.Node, path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	kinds := map[string]string{"appConfigs": "appConfig", "portalRules": "portalRule", "portalSites": "portalSite", "portalCerts": "portalCert"}
	if len(parts) < 2 || kinds[parts[0]] == "" {
		return "seed"
	}
	label := kinds[parts[0]]
	items := seedMappingValue(root, parts[0])
	index, err := strconv.Atoi(parts[1])
	if err == nil && items != nil && items.Kind == yaml.SequenceNode && index >= 0 && index < len(items.Content) {
		name := seedMappingValue(items.Content[index], "name")
		if name != nil && name.Value != "" && !strings.Contains(name.Value, "${") {
			label += fmt.Sprintf(" %q", name.Value)
		} else {
			label += " (unnamed)"
		}
	}
	fields := parts[2:]
	if parts[0] == "appConfigs" && len(fields) > 1 && fields[0] == "value" {
		fields = fields[1:]
	}
	for i := range fields {
		fields[i] = strings.ReplaceAll(strings.ReplaceAll(fields[i], "~1", "/"), "~0", "~")
	}
	if len(fields) > 0 {
		label += fmt.Sprintf(" field %q", strings.Join(fields, "."))
	}
	return label
}

func entityFieldSources(all core.FieldSources, kind string, index int) core.FieldSources {
	result := core.FieldSources{}
	prefix := "/" + kind + "/" + strconv.Itoa(index) + "/"
	for path, origin := range all {
		if strings.HasPrefix(path, prefix) {
			field := strings.TrimPrefix(path, prefix)
			if kind == "portalRules" {
				if canonical, ok := portalRuleAliases[field]; ok {
					field = canonical
				}
			}
			result["/"+field] = origin
		}
	}
	return result
}

func seedNodeJSON(node *yaml.Node) (skel.JSON, error) {
	normalized, err := configJSONNode(node)
	if err != nil {
		return "", err
	}
	var value any
	if err := normalized.Decode(&value); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(value, json.Deterministic(true))
	return skel.JSON(encoded), err
}
