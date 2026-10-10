package seeder

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// validateSeedRuleEntries requires each rule to reference an entry declared by the seed.
func validateSeedRuleEntries(payload *_SettingsYAMLPayload) error {
	names := make(map[string]bool, len(payload.PortalEntries))
	for _, entry := range payload.PortalEntries {
		names[entry.Name] = true
	}
	for _, rule := range payload.PortalRules {
		if rule.EntryName == "" {
			return fmt.Errorf("portal rule %q requires entryName", rule.Name)
		}
		if !names[rule.EntryName] {
			return fmt.Errorf("portal rule %q references portal entry %q that the seed does not declare", rule.Name, rule.EntryName)
		}
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
	return node.Decode(target)
}
