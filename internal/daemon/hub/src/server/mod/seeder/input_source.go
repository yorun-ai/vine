package seeder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

// _SeedHubSourceFile is the seed source a deployment publishes beside the seed it
// was derived from.
type _SeedHubSourceFile struct {
	Version    int               `yaml:"version"`
	SeedSHA256 string            `yaml:"seedSha256"`
	Fields     core.FieldSources `yaml:"fields"`
}

// parseSeedSource reads the seed source, checks it against the template it names,
// and returns the field sources it declares.
func parseSeedSource(source []byte, template []byte, root *yaml.Node) (core.FieldSources, error) {
	if len(source) == 0 {
		return core.FieldSources{}, nil
	}
	if _, err := parseSeedNode(source); err != nil {
		return nil, fmt.Errorf("seed source: %w", err)
	}
	var file _SeedHubSourceFile
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("seed source: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("seed source must contain one document")
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported seed source version %d", file.Version)
	}
	if file.SeedSHA256 != fmt.Sprintf("%x", sha256.Sum256(template)) {
		return nil, fmt.Errorf("seed source digest does not match seed template")
	}
	paths := map[string]bool{}
	if err := walkSeedValues(root, "", func(_ *yaml.Node, path string) error { paths[path] = true; return nil }); err != nil {
		return nil, err
	}
	sources := core.FieldSources{}
	for path, origin := range file.Fields {
		parts := strings.Split(path, "/")
		if !paths[path] || len(parts) < 4 || origin.Define == "" || origin.Source == "" || seedSourcePath(path) != path {
			return nil, fmt.Errorf("invalid seed source field %s", path)
		}
		if len(origin.Variables) > 0 || origin.Template != nil || len(origin.Bindings) > 0 {
			return nil, fmt.Errorf("substitution metadata must be derived from the seed template: %s", path)
		}
		sources[path] = origin
	}
	return sources, nil
}

// seedSourcePath names the field source a path records: the value of a config
// carries one source for the whole config rather than one per nested key.
func seedSourcePath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 5 && parts[1] == "appConfigs" && parts[3] == "value" {
		return strings.Join(parts[:5], "/")
	}
	return path
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

// entityFieldSources narrows the sources of the whole seed to the fields of one
// entity, and canonicalizes the field names a Portal rule aliases.
func entityFieldSources(all core.FieldSources, kind string, index int) core.FieldSources {
	result := core.FieldSources{}
	prefix := "/" + kind + "/" + strconv.Itoa(index) + "/"
	for path, origin := range all {
		if field, ok := strings.CutPrefix(path, prefix); ok {
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

// seedNodeJSON encodes a node as the deterministic JSON a field source records.
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
