package seeder

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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

// decodePortalRule validates the current seed rule vocabulary.
func decodePortalRule(node *yaml.Node, target any) error {
	if err := checkSeedYAMLSyntax(node); err != nil {
		return err
	}
	fields, err := seedMappingFields(node, "portal rule")
	if err != nil {
		return err
	}
	if err := checkSeedFields(fields, "portalRules"); err != nil {
		return err
	}
	if _, named := fields["entryName"]; named {
		for _, key := range []string{"matchScheme", "matchHost", "matchPort"} {
			if _, ok := fields[key]; ok {
				return fmt.Errorf("portal rule %q: entryName cannot be mixed with %s", fields["name"].Value, key)
			}
		}
	}
	return node.Decode(target)
}
