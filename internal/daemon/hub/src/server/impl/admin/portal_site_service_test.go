package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestToServerPortalSiteMapsCompleteEntity(t *testing.T) {
	entry := &core.PortalSite{
		Id:            1,
		Name:          "web",
		Type:          core.PortalSiteTypeWEBGW,
		ActorSkelName: "demo.Actor",
		ActorVia:      "client",
		WebName:       "demo.Web",
		WebMountPath:  "/demo",
		RpcgwServices: []string{"demo.Service"},
		FieldSources:  core.FieldSources{"/name": {Source: "hub", Override: "hub"}},
	}

	site := toServerPortalSite(entry, toServerFieldSources(entry.FieldSources))

	assert.Equal(t, "demo.Web", site.WebName)
	assert.Equal(t, "/demo", site.WebMountPath)
	assert.Equal(t, []string{"demo.Service"}, site.RpcgwServices)
	assert.Len(t, site.FieldSources, 1)
	assert.Equal(t, "/name", site.FieldSources[0].Path)
	// List responses carry the entity without its seed provenance.
	assert.Empty(t, toServerPortalSite(entry, nil).FieldSources)
}
