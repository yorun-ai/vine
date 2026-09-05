package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleTargetPathRedisRoundTrip(t *testing.T) {
	rule := &core.PortalRule{Name: "mapped", RouteType: "SITE", MatchPathPrefix: "/api", RoutePathPrefix: "/internal"}
	wire := vcode.MustMarshalJsonS(ToRedisedPortalRule(rule))
	decoded := vcode.MustUnmarshalJsonS[*redised.PortalRule](wire)
	assert.Equal(t, "/internal", decoded.RoutePathPrefix)
	assert.Contains(t, wire, `"routePathPrefix":"/internal"`)
	assert.Contains(t, wire, `"matchPathPrefix":"/api"`)
	assert.NotContains(t, wire, `"targetPath"`)
	assert.NotContains(t, wire, `"targetType"`)
}
