package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/webtimeout"
)

func TestToServerPortalSiteMapsCompleteEntity(t *testing.T) {
	entry := &core.PortalSite{
		Id:            1,
		Name:          "web",
		Type:          core.PortalSiteTypeWEBGW,
		ActorSkelName: "demo.Actor",
		McpEnabled:    true,
		WebName:       "demo.Web",
		WebMountPath:  "/demo",
		RpcgwServices: []string{"demo.Service"},
		FieldSources:  core.FieldSources{"/name": {Source: "hub", Override: "hub"}},
	}

	site := toServerPortalSite(entry, toServerFieldSources(entry.FieldSources))

	assert.Equal(t, "demo.Web", site.WebName)
	assert.True(t, site.McpEnabled)
	assert.Equal(t, "/demo", site.WebMountPath)
	assert.Equal(t, []string{"demo.Service"}, site.RpcgwServices)
	assert.Len(t, site.FieldSources, 1)
	assert.Equal(t, "/name", site.FieldSources[0].Path)
	// List responses carry the entity without its seed provenance.
	assert.Empty(t, toServerPortalSite(entry, nil).FieldSources)
}

func TestWebTimeoutAdminMapping(t *testing.T) {
	config := &webtimeout.Config{Default: &webtimeout.Policy{Total: "30s"}, Paths: []webtimeout.Path{{Path: "/go/**", Total: "15m"}}}
	entry := &core.PortalSite{WebTimeout: config}
	detail := toServerPortalSite(entry, nil)
	require.Equal(t, config, toCoreWebTimeout(detail.WebTimeout))
	require.Equal(t, detail.WebTimeout, toServerPortalSiteListItem(entry).WebTimeout)
	require.Nil(t, toCoreWebTimeout(nil))
	require.Equal(t, &webtimeout.Config{Paths: []webtimeout.Path{}}, toCoreWebTimeout(toServerWebTimeout(&webtimeout.Config{})))
}
