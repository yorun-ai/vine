package lock

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
)

func TestDisabledAndMissingModeRejectOperations(t *testing.T) {
	for _, mode := range []string{"disable", ""} {
		t.Run(mode, func(t *testing.T) {
			locker, hub := newTestLocker(t, mode, "", false)
			ctx := lockContext(context.Background())
			assert.Panics(t, func() { locker.Acquire(ctx, "key", "token", 1000) })
			assert.Panics(t, func() { locker.Renew(ctx, "key", "token", 1000) })
			assert.Panics(t, func() { locker.Release(ctx, "key", "token") })
			assert.Empty(t, hub.calls)
			assert.Nil(t, locker.impl)
		})
	}
}

func TestLockerKeepsRedisClientWhenHubLockEndpointIsUnchanged(t *testing.T) {
	locker, _, info := newTestLockerWithHubInfo(t, hublock.ModeRedis, "redis://127.0.0.1:6379/0")
	previous := locker.impl

	info.Refresh()

	assert.Same(t, previous, locker.impl)
}

func TestLockerReplacesRedisClientAndDropsLeasesWhenHubLockEndpointChanged(t *testing.T) {
	locker, hubInfoClient, info := newTestLockerWithHubInfo(t, hublock.ModeRedis, "redis://127.0.0.1:6379/0")
	previous := locker.impl.(*_RedisLocker)

	hubInfoClient.info.LockRedisEndpoint = "redis://127.0.0.1:6380/0"
	info.Refresh()

	next, ok := locker.impl.(*_RedisLocker)
	if !ok {
		t.Fatalf("expected a redis locker, got %T", locker.impl)
	}
	assert.NotSame(t, previous, next)
	assert.Equal(t, "127.0.0.1:6380", next.client.Options().Addr)
	// The abandoned connection is closed, so leases held through it are dropped
	// instead of being carried over to the new endpoint.
	_, err := previous.client.Ping(context.Background()).Result()
	assert.ErrorIs(t, err, redis.ErrClosed)
}

func TestLockerRebuildsWhenHubLockModeChanges(t *testing.T) {
	locker, hubInfoClient, info := newTestLockerWithHubInfo(t, hublock.ModeEmbedded, "")

	hubInfoClient.info.LockMode = hublock.ModeRedis
	hubInfoClient.info.LockRedisEndpoint = "redis://127.0.0.1:6379/0"
	info.Refresh()

	assert.IsType(t, new(_RedisLocker), locker.impl)
}
