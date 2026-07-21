package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
)

func testSyncer(redisServer *redisserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}
