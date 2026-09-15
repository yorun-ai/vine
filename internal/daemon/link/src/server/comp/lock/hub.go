package lock

import (
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
)

type _HubLocker struct {
	client hubskeled.LockServiceClient
}

func (l *_HubLocker) Acquire(ctx meta.Context, key, token string, ttlMillis int) bool {
	return l.client.Acquire(key, token, ttlMillis, client.WithContext(ctx))
}

func (l *_HubLocker) Renew(ctx meta.Context, key, token string, ttlMillis int) bool {
	return l.client.Renew(key, token, ttlMillis, client.WithContext(ctx))
}

func (l *_HubLocker) Release(ctx meta.Context, key, token string) bool {
	return l.client.Release(key, token, client.WithContext(ctx))
}

func (l *_HubLocker) Close() {
	// The Hub client is owned by the application lifecycle.
}
