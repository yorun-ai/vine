package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestWebTimeoutSeed(t *testing.T) {
	entities, err := parseSeedEntities(`portalSites:
- name: files
  type: WEBGW
  actorSkelName: demo.Actor
  webName: demo.Web
  webTimeout:
    default: {total: 30s}
    paths:
    - {path: /go/**, total: 15m}
`)
	require.NoError(t, err)
	site := new(core.PortalSiteCore).Validate(*entities.PortalSites[0])
	require.Equal(t, "30s", site.WebTimeout.Default.Total)
	require.Equal(t, "/go/**", site.WebTimeout.Paths[0].Path)
	require.Equal(t, "15m", site.WebTimeout.Paths[0].Total)
	for _, value := range []string{"null", "[]", "{defaults: {total: 1s}}", "{default: null}", "{default: {total: 30}}", "{paths: {}}", "{paths: [{path: /go/**, total: 1s, extra: true}]}", "{paths: [{total: 1s}]}"} {
		_, err := parseSeedEntities("portalSites: [{name: files, type: WEBGW, actorSkelName: demo.Actor, webName: demo.Web, webTimeout: " + value + "}]")
		require.Error(t, err, value)
	}
}

func TestWebTimeoutSeedVariables(t *testing.T) {
	node, _, err := resolveSeedInputWithDescriptors([]byte(`portalSites:
- name: files
  type: WEBGW
  actorSkelName: demo.Actor
  webName: demo.Web
  webTimeout:
    default: {total: '${text:30s}'}
    paths: [{path: /go/**, total: '${text:15m}'}]
`), nil, nil, testVarsDescriptors())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.Equal(t, "30s", payload.PortalSites[0].WebTimeout.Default.Total)
	require.Equal(t, "15m", payload.PortalSites[0].WebTimeout.Paths[0].Total)
}
