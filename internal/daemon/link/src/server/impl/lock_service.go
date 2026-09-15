package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"

	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/lock"
)

// LockServiceServerImpl forwards lock operations using the active Rpc context.
type LockServiceServerImpl struct {
	skeled.DefaultLockServiceServer
	Context spec.Context `inject:""`
	Locker  *lock.Locker `inject:""`
}

func (s *LockServiceServerImpl) Acquire(key, token string, ttlMillis int) bool {
	return s.Locker.Acquire(s.Context, key, token, ttlMillis)
}

func (s *LockServiceServerImpl) Renew(key, token string, ttlMillis int) bool {
	return s.Locker.Renew(s.Context, key, token, ttlMillis)
}

func (s *LockServiceServerImpl) Release(key, token string) bool {
	return s.Locker.Release(s.Context, key, token)
}
