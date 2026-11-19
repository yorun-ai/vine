package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
)

func testSyncer(watchServer *watchserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{WatchServer: watchServer, Access: new(configaccess.Access)}
	target.DIInit()
	return target
}
