package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/webtimeout"
)

func TestWebTimeoutAdminMapping(t *testing.T) {
	config := &webtimeout.Config{Default: &webtimeout.Policy{Total: "30s"}, Paths: []webtimeout.Path{{Path: "/go/**", Total: "15m"}}}
	entry := &core.PortalSite{WebTimeout: config}
	detail := toServerPortalSite(entry, nil)
	require.Equal(t, config, toCoreWebTimeout(detail.WebTimeout))
	require.Equal(t, detail.WebTimeout, toServerPortalSiteListItem(entry).WebTimeout)
	require.Nil(t, toCoreWebTimeout(nil))
	require.Equal(t, &webtimeout.Config{Paths: []webtimeout.Path{}}, toCoreWebTimeout(toServerWebTimeout(&webtimeout.Config{})))
}
