package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestToServerPortalSiteKeepsDerivedWebMountPath(t *testing.T) {
	site := toServerPortalSite(core.PortalSite{
		Id:            1,
		Name:          "web",
		Type:          core.PortalSiteTypeWEBGW,
		ActorSkelName: "demo.Actor",
		ActorVia:      "client",
		WebName:       "demo.Web",
		WebMountPath:  "/demo",
	}, []string{"demo.Service"})

	assert.Equal(t, "demo.Web", site.WebName)
	assert.Equal(t, "/demo", site.WebMountPath)
	assert.Equal(t, []string{"demo.Service"}, site.RpcgwServices)
}
