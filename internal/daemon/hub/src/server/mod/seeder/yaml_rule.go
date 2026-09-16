package seeder

import (
	"fmt"
	"sort"

	"go.yorun.ai/vine/internal/core/logger"
	"gopkg.in/yaml.v3"
)

var ruleLogger = logger.New("vine.hub.seed")

var portalRuleAliases = map[string]string{
	"scheme": "matchScheme", "host": "matchHost", "port": "matchPort",
	"pathPrefix": "matchPathPrefix", "targetType": "routeType",
	"siteName": "routeSiteName", "targetPath": "routePathPrefix",
	"redirectionPattern": "routeRedirectionPattern",
}

// checkSeedRuleStyle rejects a document that mixes the two ways a rule joins a
// Portal entry. A rule either names its entry with entryName, or declares the
// access that the entry serves; one document uses one of the two, so a seed
// never describes the entries of an application in two ways at once. A document
// that declares portalEntries names them, so Hub never guesses whether a rule
// meant a declared entry or the access it serves.
func checkSeedRuleStyle(payload *_SettingsYAMLPayload) error {
	named, declared := "", ""
	for _, rule := range payload.PortalRules {
		if rule.EntryName != "" {
			if named == "" {
				named = rule.Name
			}
			continue
		}
		if declared == "" && (rule.MatchScheme != "" || rule.MatchHost != "" || rule.MatchPort != 0) {
			declared = rule.Name
		}
	}
	if named != "" && declared != "" {
		return fmt.Errorf("portal rule %q declares an access while portal rule %q names an entry; a seed declares one or the other, never both", declared, named)
	}
	if declared != "" && len(payload.PortalEntries) > 0 {
		return fmt.Errorf("portal rule %q declares an access while the seed declares portalEntries; name the entry with entryName instead", declared)
	}
	// A seed is self-contained: a rule references an entry the same document
	// declares, so Hub never completes the relationship from stored data.
	names := make(map[string]bool, len(payload.PortalEntries))
	for _, entry := range payload.PortalEntries {
		names[entry.Name] = true
	}
	for _, rule := range payload.PortalRules {
		if rule.EntryName == "" || names[rule.EntryName] {
			continue
		}
		return fmt.Errorf("portal rule %q references portal entry %q that the seed does not declare", rule.Name, rule.EntryName)
	}
	return nil
}

// decodePortalRule accepts legacy YAML fields only at the import boundary.
// A rule must use either legacy or new fields, never a mixture of both.
// TODO: Remove legacy field aliases, compatibility warnings, and mixed-field
// checks after explicitly retiring the old YAML format with migration guidance;
// then accept only match* / route* fields. Raising the minimum Vine version alone
// does not retire this format: Vine v0.15.7 still accepts these aliases.
func decodePortalRule(node *yaml.Node, target any) error {
	if err := checkSeedYAMLSyntax(node); err != nil {
		return err
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("portal rule must be a YAML mapping")
	}

	// Reject duplicate keys before checking which field vocabulary is used.
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
		if _, ok := portalRuleAliases[key]; ok {
			legacy = key
		}
		for _, replacement := range portalRuleAliases {
			if key == replacement {
				current = key
			}
		}
	}
	if legacy != "" && current != "" {
		return fmt.Errorf("portal rule %q: legacy YAML field %q cannot be mixed with new field %q", name, legacy, current)
	}
	// A rule joins an entry either by naming it or by declaring the access that
	// entry serves. The entry owns the access, so a rule never does both.
	if _, named := fields["entryName"]; named {
		for _, key := range []string{"matchScheme", "matchHost", "matchPort", "scheme", "host", "port"} {
			if _, ok := fields[key]; ok {
				return fmt.Errorf("portal rule %q: entryName cannot be mixed with %s", name, key)
			}
		}
	}
	normalized := *node
	normalized.Content = make([]*yaml.Node, 0, len(fields)*2)
	for _, field := range keys {
		key := field
		if replacement, ok := portalRuleAliases[field]; ok {
			ruleLogger.Warn("deprecated portal rule YAML field; use the new field instead", "rule", name, "field", field, "replacement", replacement)
			key = replacement
		}
		value := fields[field]
		normalized.Content = append(normalized.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &value)
	}
	return normalized.Decode(target)
}
