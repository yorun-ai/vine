package seeder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleSeedMapsWithoutDomainValidation(t *testing.T) {
	seed := _PortalRule{Name: "mapped", RouteType: "SITE", RoutePathPrefix: "/internal/"}
	rule := seed.ToCorePortalRule()
	assert.Zero(t, rule.Id)
	assert.False(t, rule.BuiltIn)
	assert.Equal(t, "/internal/", rule.RoutePathPrefix)
	seed.RoutePathPrefix = ""
	assert.Empty(t, seed.ToCorePortalRule().RoutePathPrefix)
	seed.RouteType = "PERMANENT_REDIRECT"
	seed.RoutePathPrefix = "/internal"
	assert.NotPanics(t, func() { seed.ToCorePortalRule() })
}

func TestSeedPortalRuleFieldNames(t *testing.T) {
	for _, content := range []string{
		"portalRules:\n  - name: example\n    matchScheme: http\n    routeType: SITE\n    routePathPrefix: /internal",
		"portalRules:\n  - name: example\n    scheme: http\n    targetType: SITE\n    targetPath: /internal",
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
		require.NoError(t, err)
		rule := payload.PortalRules[0].ToCorePortalRule()
		assert.Equal(t, "http", rule.MatchScheme)
		assert.Equal(t, "/internal", rule.RoutePathPrefix)
	}
	_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalRules:\n  - scheme: http\n    routeType: SITE")
	require.ErrorContains(t, err, "cannot be mixed")
}
