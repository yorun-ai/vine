package seedconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/logger"
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
	return DecodePortalRule(node, (*plain)(r))
}

func TestPortalRuleYAMLCompatibility(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "warnings.log")
	previous := ruleLogger
	ruleLogger = logger.New("vine.hub.seed", logger.WithOption{OutputPath: logPath, Level: logger.LevelWarn})
	t.Cleanup(func() { ruleLogger = previous })
	legacy := "name: example\nscheme: https\nhost: example.com\nport: 443\npathPrefix: /api\ntargetType: SITE\nsiteName: web\ntargetPath: /internal\nredirectionPattern: ''\n"
	var oldRule testRule
	require.NoError(t, yaml.Unmarshal([]byte(legacy), &oldRule))
	expected := testRule{Name: "example", MatchScheme: "https", MatchHost: "example.com", MatchPort: 443, MatchPathPrefix: "/api", RouteType: "SITE", RouteSiteName: "web", RoutePathPrefix: "/internal"}
	require.Equal(t, expected, oldRule)
	logged, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Equal(t, 8, strings.Count(string(logged), "level=WARN"))
	require.Contains(t, string(logged), "replacement=routePathPrefix")
	current, err := yaml.Marshal(oldRule)
	require.NoError(t, err)
	require.NotContains(t, string(current), "targetPath:")
	require.Contains(t, string(current), "routePathPrefix:")
	var newRule testRule
	require.NoError(t, yaml.Unmarshal(current, &newRule))
	require.Equal(t, expected, newRule)
	after, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Equal(t, string(logged), string(after), "new fields must not warn")
}

func TestPortalRuleYAMLRejectsMixedFields(t *testing.T) {
	for _, input := range []string{
		"host: old\nmatchHost: ''", "port: 80\nmatchPort: 0",
		"targetPath: /old\nroutePathPrefix: ''", "scheme: http\nrouteType: SITE",
		"matchScheme: http\ntargetType: SITE", "host: ''\nmatchHost: ''",
	} {
		t.Run(input, func(t *testing.T) {
			var rule testRule
			err := yaml.Unmarshal([]byte("name: mixed\n"+input), &rule)
			require.ErrorContains(t, err, "cannot be mixed")
			require.ErrorContains(t, err, "mixed")
		})
	}
}

func TestPortalRuleYAMLMergeAndDuplicateKeys(t *testing.T) {
	var payload struct {
		Rules []testRule `yaml:"rules"`
	}
	require.NoError(t, yaml.Unmarshal([]byte("defaults: &defaults\n  scheme: http\n  targetType: SITE\nrules:\n  - <<: *defaults\n    name: merged\n"), &payload))
	require.Equal(t, "http", payload.Rules[0].MatchScheme)
	require.Equal(t, "SITE", payload.Rules[0].RouteType)
	require.ErrorContains(t, yaml.Unmarshal([]byte("defaults: &defaults\n  scheme: http\nrules:\n  - <<: *defaults\n    matchScheme: https\n"), &payload), "cannot be mixed")
	var rule testRule
	require.Error(t, yaml.Unmarshal([]byte("host: a\nhost: b"), &rule))
}
