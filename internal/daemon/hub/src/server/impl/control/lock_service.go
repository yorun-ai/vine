package control

import (
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/lockserver"
)

type LockServiceServerImpl struct {
	skeled.DefaultLockServiceServer
	Server *lockserver.Server `inject:""`
}

func (s *LockServiceServerImpl) Acquire(key, token string, ttlMillis int) bool {
	return s.Server.Acquire(key, token, ttlMillis)
}

func (s *LockServiceServerImpl) Renew(key, token string, ttlMillis int) bool {
	return s.Server.Renew(key, token, ttlMillis)
}

func (s *LockServiceServerImpl) Release(key, token string) bool {
	return s.Server.Release(key, token)
}
