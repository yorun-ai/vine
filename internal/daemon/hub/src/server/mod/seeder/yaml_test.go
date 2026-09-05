package seeder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleSeedTargetPath(t *testing.T) {
	seed := _PortalRule{Name: "mapped", RouteType: "SITE", RoutePathPrefix: "/internal/"}
	rule := seed.ToCorePortalRule(&core.PortalRule{Id: 17})
	assert.Equal(t, 17, rule.Id)
	assert.Equal(t, "/internal", rule.RoutePathPrefix)
	seed.RoutePathPrefix = ""
	assert.Empty(t, seed.ToCorePortalRule(rule).RoutePathPrefix)
	seed.RouteType = "PERMANENT_REDIRECT"
	seed.RoutePathPrefix = "/internal"
	assert.Panics(t, func() { seed.ToCorePortalRule(nil) })
}

func TestSeedPortalRuleFieldNames(t *testing.T) {
	for _, content := range []string{
		"portalRules:\n  - name: example\n    matchScheme: http\n    routeType: SITE\n    routePathPrefix: /internal",
		"portalRules:\n  - name: example\n    scheme: http\n    targetType: SITE\n    targetPath: /internal",
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
		require.NoError(t, err)
		rule := payload.PortalRules[0].ToCorePortalRule(nil)
		assert.Equal(t, "http", rule.MatchScheme)
		assert.Equal(t, "/internal", rule.RoutePathPrefix)
	}
	_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalRules:\n  - scheme: http\n    routeType: SITE")
	require.ErrorContains(t, err, "cannot be mixed")
}
