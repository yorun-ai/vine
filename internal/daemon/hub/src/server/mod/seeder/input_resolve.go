package seeder

import (
	"encoding/json/jsontext"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

// resolveSeedInput preserves YAML types for whole-field references. Substituted
// variable values are literal data, not recursively evaluated templates.
func resolveSeedInput(template []byte, variables []byte, source []byte) (*yaml.Node, core.FieldSources, error) {
	return resolveSeedInputWithSchemas(template, variables, source, skel.RegisteredDomainSchemas())
}

func resolveSeedInputWithSchemas(template []byte, variables []byte, source []byte, domains []*skel.DomainSchema) (*yaml.Node, core.FieldSources, error) {
	resolver, err := newSeedResolver(template, variables, source, domains)
	if err != nil {
		return nil, nil, err
	}
	if err := resolver.walk(resolver.root, ""); err != nil {
		return nil, nil, err
	}
	return resolver.root, resolver.sources, nil
}

// _SeedResolver resolves the variable references of one seed template and records
// the field sources it derives. root keeps the paths that seed locations and
// application points are looked up with, and originals keeps the unresolved
// template a field source reports.
type _SeedResolver struct {
	root       *yaml.Node
	originals  map[string]*yaml.Node
	sources    core.FieldSources
	schema     *_VarsSchema
	dictionary *yaml.Node
}

func newSeedResolver(template []byte, variables []byte, source []byte, domains []*skel.DomainSchema) (*_SeedResolver, error) {
	root, err := parseSeedNode(template)
	if err != nil {
		return nil, err
	}
	originals := map[string]*yaml.Node{}
	if err := walkSeedValues(cloneSeedNode(root), "", func(node *yaml.Node, path string) error {
		originals[path] = node
		return nil
	}); err != nil {
		return nil, err
	}
	sources, err := parseSeedSource(source, template, root)
	if err != nil {
		return nil, err
	}
	var dictionary *yaml.Node
	if len(variables) > 0 {
		dictionary, err = parseSeedNode(variables)
		if err != nil {
			return nil, fmt.Errorf("seed variables: %w", err)
		}
	}
	return &_SeedResolver{
		root:       root,
		originals:  originals,
		sources:    sources,
		schema:     newVarsSchema(domains),
		dictionary: dictionary,
	}, nil
}

// walk resolves the references of node and of the values below it. Only original
// template nodes are resolved, so inserted values remain literal.
func (r *_SeedResolver) walk(node *yaml.Node, path string) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		return r.resolveString(node, path)
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if err := r.walk(node.Content[i+1], path+"/"+escapeSeedPointer(node.Content[i].Value)); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			if err := r.walk(child, path+"/"+strconv.Itoa(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveString substitutes the references of one string. A string that is only a
// reference takes the variable with its YAML type, a reference inside a longer
// string interpolates text, and a config value accepts an interpolated JSON
// document.
func (r *_SeedResolver) resolveString(node *yaml.Node, path string) error {
	location := seedLocation(r.root, path)
	matches := seedVariable.FindAllStringSubmatch(node.Value, -1)
	if strings.Contains(seedVariable.ReplaceAllString(node.Value, ""), "${") {
		return fmt.Errorf("malformed seed variable at %s", location)
	}
	if len(matches) == 0 {
		return nil
	}
	target, err := r.schema.targetType(r.root, path)
	if err != nil {
		return fmt.Errorf("seed application point %s: %w", location, err)
	}
	whole := len(matches) == 1 && matches[0][0] == node.Value
	substitution := &_SeedSubstitution{
		whole:        whole,
		jsonConfig:   !whole && target != nil && target.Kind == skel.TypeKindConfig,
		location:     location,
		target:       target,
		replacements: map[string]*yaml.Node{},
		dependencies: map[string]bool{},
	}
	if !substitution.whole && !substitution.jsonConfig && target != nil && (target.Kind != skel.TypeKindScalar || target.Scalar != skel.ScalarString) {
		return fmt.Errorf("string interpolation requires a string target at %s", location)
	}
	for _, match := range matches {
		if err := r.substitute(substitution, match); err != nil {
			return err
		}
	}
	if err := r.recordSources(path, substitution); err != nil {
		return err
	}
	if substitution.whole {
		*node = *cloneSeedNode(substitution.replacements[matches[0][0]])
		return nil
	}
	node.Value = seedVariable.ReplaceAllStringFunc(node.Value, func(ref string) string {
		return substitution.replacements[ref].Value
	})
	if substitution.jsonConfig {
		return r.resolveConfigJSON(node, substitution)
	}
	return nil
}

// _SeedSubstitution is what one string resolves to: the value every reference is
// replaced with, and what the field source records about them.
type _SeedSubstitution struct {
	// whole reports a string that is nothing but one reference, which takes the
	// value of the variable instead of interpolating it as text.
	whole bool
	// jsonConfig reports a config value that interpolates into a JSON document.
	jsonConfig bool

	location     string
	target       *skel.TypeSchema
	replacements map[string]*yaml.Node
	bindings     []core.FieldSourceBinding
	dependencies map[string]bool
}

// substitute resolves one reference and records the variable it names.
func (r *_SeedResolver) substitute(substitution *_SeedSubstitution, match []string) error {
	location := substitution.location
	name, fallback, hasDefault := strings.Cut(match[1], ":")
	for _, segment := range strings.Split(name, ".") {
		if !seedVariableSegment.MatchString(segment) {
			return fmt.Errorf("seed variable %s must use camelCase path segments at %s", name, location)
		}
	}
	kind, err := r.schema.variableType(name)
	if err != nil {
		return fmt.Errorf("seed variable %s at %s: %w", name, location, err)
	}
	value, found, err := lookupSeedVariable(r.dictionary, name)
	if err != nil {
		return fmt.Errorf("seed variable at %s: %w", location, err)
	}
	if !found {
		if !hasDefault {
			return fmt.Errorf("variable %q is missing and has no default", name)
		}
		defaultType := kind
		if !substitution.whole {
			defaultType = seedScalar(skel.ScalarString)
		} else if defaultType == nil {
			defaultType = substitution.target
		}
		value, err = parseSeedDefault(fallback, defaultType)
		if err != nil {
			return fmt.Errorf("seed variable %s at %s: %w", name, location, err)
		}
	}
	// Interpolation defaults are already text; supplied variables still obey their
	// declared type. Null is preserved until use.
	if substitution.whole || found {
		value, err = r.schema.validateValue(value, kind, name, false)
		if err != nil {
			return fmt.Errorf("seed variable at %s: %w", location, err)
		}
	}
	if substitution.whole {
		value, err = r.schema.validateValue(value, substitution.target, location, true)
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
	substitution.bindings = append(substitution.bindings, core.FieldSourceBinding{
		Variable:    name,
		Reference:   match[0],
		Value:       encoded,
		DefaultUsed: !found,
	})
	substitution.dependencies[name] = true
	substitution.replacements[match[0]] = value
	return nil
}

// recordSources records the template and the bindings a substituted field
// declares. A config value keeps one source for the whole config, so a nested key
// records its bindings on the path of that config.
func (r *_SeedResolver) recordSources(path string, substitution *_SeedSubstitution) error {
	sourcePath := seedSourcePath(path)
	origin := r.sources[sourcePath]
	if origin.Template == nil {
		original, err := seedNodeJSON(r.originals[sourcePath])
		if err != nil {
			return fmt.Errorf("cannot record template at %s: %w", substitution.location, err)
		}
		origin.Template = &original
	}
	for _, binding := range substitution.bindings {
		binding.Path = strings.TrimPrefix(path, sourcePath)
		origin.Bindings = append(origin.Bindings, binding)
	}
	for _, name := range origin.Variables {
		substitution.dependencies[name] = true
	}
	origin.Variables = nil
	for name := range substitution.dependencies {
		origin.Variables = append(origin.Variables, name)
	}
	sort.Strings(origin.Variables)
	r.sources[sourcePath] = origin
	return nil
}

// resolveConfigJSON validates the JSON an interpolated config value produced and
// replaces the string with the value it decodes to.
func (r *_SeedResolver) resolveConfigJSON(node *yaml.Node, substitution *_SeedSubstitution) error {
	location := substitution.location
	if !jsontext.Value(node.Value).IsValid() {
		return fmt.Errorf("invalid interpolated config JSON at %s", location)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(node.Value), &doc); err != nil {
		return fmt.Errorf("invalid interpolated config JSON at %s", location)
	}
	checked, err := r.schema.validateValue(doc.Content[0], substitution.target, location, true)
	if err != nil {
		return err
	}
	*node = *checked
	return nil
}
