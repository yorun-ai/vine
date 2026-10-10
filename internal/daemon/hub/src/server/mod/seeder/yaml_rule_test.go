package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testRule struct {
	Name                    string `yaml:"name"`
	MatchScheme             string `yaml:"matchScheme"`
	MatchHost               string `yaml:"matchHost"`
	MatchPort               int    `yaml:"matchPort"`
	MatchPathPrefix         string `yaml:"matchPathPrefix"`
	RouteType               string `yaml:"routeType"`
	RouteSiteName           string `yaml:"routeSiteName"`
	RoutePathPrefix         string `yaml:"routePathPrefix"`
	RouteRedirectionPattern string `yaml:"routeRedirectionPattern"`
}

func (r *testRule) UnmarshalYAML(node *yaml.Node) error {
	type plain testRule
	return decodePortalRule(node, (*plain)(r))
}

func TestPortalRuleYAMLCurrentFieldsAndRejectsAliases(t *testing.T) {
	var rule testRule
	require.NoError(t, yaml.Unmarshal([]byte("name: example\nmatchScheme: https\nmatchHost: example.com\nmatchPort: 443\nmatchPathPrefix: /api\nrouteType: SITE\nrouteSiteName: web\nroutePathPrefix: /internal"), &rule))
	require.Equal(t, "https", rule.MatchScheme)
	for _, field := range []string{"scheme", "host", "port", "pathPrefix", "targetType", "siteName", "targetPath", "redirectionPattern"} {
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
			var rule testRule
			require.ErrorContains(t, yaml.Unmarshal([]byte(input), &rule), "not supported")
		})
	}
	var rule testRule
	require.Error(t, yaml.Unmarshal([]byte("host: a\nhost: b"), &rule))
}
