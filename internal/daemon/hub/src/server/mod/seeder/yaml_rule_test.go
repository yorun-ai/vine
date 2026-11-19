package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPortalRuleYAMLCurrentFieldsAndRejectsAliases(t *testing.T) {
	var rule _PortalRule
	require.NoError(t, yaml.Unmarshal([]byte("name: example\nentryName: web\nmatchPathPrefix: /api\nrouteType: SITE\nrouteSiteName: web\nroutePathPrefix: /internal"), &rule))
	require.Equal(t, "web", rule.EntryName)
	for _, field := range []string{"matchScheme", "matchHost", "matchPort", "scheme", "host", "port", "pathPrefix", "targetType", "siteName", "targetPath", "redirectionPattern"} {
		require.ErrorContains(t, yaml.Unmarshal([]byte("name: example\n"+field+": old"), &rule), "unknown field")
	}
}

func TestPortalRuleYAMLRejectsReferencesAndDuplicateKeys(t *testing.T) {
	for _, input := range []string{
		"name: rule\n<<: {scheme: http}",
		"name: rule\nhost: &host example.com",
		"name: &name rule\nhost: *name",
	} {
		t.Run(input, func(t *testing.T) {
			var rule _PortalRule
			require.ErrorContains(t, yaml.Unmarshal([]byte(input), &rule), "not supported")
		})
	}
	var rule _PortalRule
	require.ErrorContains(t, yaml.Unmarshal([]byte("name: a\nname: b"), &rule), "already defined")
}

func TestSeedRulesRequireDeclaredEntries(t *testing.T) {
	_, err := parseSeedEntities("portalRules: [{name: web, routeType: SITE, routeSiteName: app.Web}]")
	require.ErrorContains(t, err, "requires entryName")
	_, err = parseSeedEntities("portalRules: [{name: web, entryName: missing, routeType: SITE, routeSiteName: app.Web}]")
	require.ErrorContains(t, err, "seed does not declare")
}
