package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleTargetPathWatchRoundTrip(t *testing.T) {
	rule := &core.PortalRule{Name: "mapped", RouteType: "SITE", MatchPathPrefix: "/api", RoutePathPrefix: "/internal"}
	wire := vcode.MustMarshalJsonS(ToWatchedPortalRule(rule))
	decoded := vcode.MustUnmarshalJsonS[*watched.PortalRule](wire)
	assert.Equal(t, "/internal", decoded.RoutePathPrefix)
	assert.Contains(t, wire, `"routePathPrefix":"/internal"`)
	assert.Contains(t, wire, `"matchPathPrefix":"/api"`)
	assert.NotContains(t, wire, `"targetPath"`)
	assert.NotContains(t, wire, `"targetType"`)
}

func TestPortalSitePublishesWebMountPath(t *testing.T) {
	for _, mountPath := range []string{"", "/", "/app"} {
		site := &core.PortalSite{Name: "web", Type: core.PortalSiteTypeWEBGW, WebName: "demo.Web", WebMountPath: mountPath}
		wire := vcode.MustMarshalJsonS(toWatchedPortalSite(site))
		decoded := vcode.MustUnmarshalJsonS[watched.PortalSite](wire)
		assert.Equal(t, mountPath, decoded.WebgwConfig.MountPath)
		site.BuiltIn = true
		assert.Empty(t, toWatchedPortalSite(site).WebgwConfig.MountPath, "built-in Dashboard access remains configurable")
	}
}
