package seedconfig

import (
	"fmt"
	"sort"

	"go.yorun.ai/vine/internal/core/logger"
	"gopkg.in/yaml.v3"
)

var ruleLogger = logger.New("vine.hub.seed")

// DecodePortalRule accepts legacy YAML fields only at the import boundary.
// A rule must use either legacy or new fields, never a mixture of both.
// TODO: Remove legacy field aliases, compatibility warnings, and mixed-field
// checks once old YAML support is retired; accept only match* / route* fields.
func DecodePortalRule(node *yaml.Node, target any) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("portal rule must be a YAML mapping")
	}
	aliases := map[string]string{
		"scheme": "matchScheme", "host": "matchHost", "port": "matchPort",
		"pathPrefix": "matchPathPrefix", "targetType": "routeType",
		"siteName": "routeSiteName", "targetPath": "routePathPrefix",
		"redirectionPattern": "routeRedirectionPattern",
	}
	// Decode the mapping first so YAML aliases and merge keys are resolved and
	// duplicate keys rejected before checking which field vocabulary is used.
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return err
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	name := fields["name"].Value
	legacy, current := "", ""
	for _, key := range keys {
		if _, ok := aliases[key]; ok {
			legacy = key
		}
		for _, replacement := range aliases {
			if key == replacement {
				current = key
			}
		}
	}
	if legacy != "" && current != "" {
		return fmt.Errorf("portal rule %q: legacy YAML field %q cannot be mixed with new field %q", name, legacy, current)
	}
	normalized := *node
	normalized.Content = make([]*yaml.Node, 0, len(fields)*2)
	for _, field := range keys {
		key := field
		if replacement, ok := aliases[field]; ok {
			ruleLogger.Warn("deprecated portal rule YAML field; use the new field instead", "rule", name, "field", field, "replacement", replacement)
			key = replacement
		}
		value := fields[field]
		normalized.Content = append(normalized.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &value)
	}
	return normalized.Decode(target)
}
