//go:build goroutineleak

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/testutil/goroutineleak"
)

func TestGoroutineLeakRedisLockRefreshLifecycle(t *testing.T) {
	runRedisLockRefreshLifecycle(t)
	goroutineleak.RequireNone(t)
}

func runRedisLockRefreshLifecycle(t *testing.T) {
	locker := &Locker{
		ctx:       context.Background(),
		cmdable:   newTestLockCmdable(),
		keyPrefix: "goroutineleak",
	}
	lock, ok := locker.Lock("refresh")
	require.True(t, ok)
	lock.Unlock()
}

func TestGoroutineLeakRedisLockTimeoutLifecycle(t *testing.T) {
	runRedisLockTimeoutLifecycle(t)
	goroutineleak.RequireNone(t)
}

func runRedisLockTimeoutLifecycle(t *testing.T) {
	locker := &Locker{
		ctx:       context.Background(),
		cmdable:   newTestLockCmdable(),
		keyPrefix: "goroutineleak",
	}
	lock, ok := locker.Lock("timeout", WithTimeout(time.Millisecond))
	require.True(t, ok)

	select {
	case <-lock.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("redis lock timeout goroutine did not stop")
	}
}
