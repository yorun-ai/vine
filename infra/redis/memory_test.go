package redis

import (
	"net"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMemoryNamedInstances(t *testing.T) {
	ctx := t.Context()
	a, releaseA := acquireMemoryClient("redis+memory://cache")
	defer releaseA()
	shared, releaseShared := acquireMemoryClient("redis+memory://cache")
	defer releaseShared()
	isolated, releaseIsolated := acquireMemoryClient("redis+memory://session")
	defer releaseIsolated()
	require.Same(t, a, shared)
	require.NotSame(t, a, isolated)
	require.NoError(t, a.Set(ctx, "key", "value", 0).Err())
	require.Equal(t, "value", shared.Get(ctx, "key").Val())
	require.ErrorIs(t, isolated.Get(ctx, "key").Err(), goredis.Nil)
	releaseA()
	releaseA()
	require.Equal(t, "value", shared.Get(ctx, "key").Val())
	dial := shared.Options().Dialer
	releaseShared()
	_, err := dial(ctx, "tcp", "cache")
	require.ErrorIs(t, err, net.ErrClosed)
	fresh, releaseFresh := acquireMemoryClient("redis+memory://cache")
	defer releaseFresh()
	require.NotSame(t, shared, fresh)
	require.ErrorIs(t, fresh.Get(ctx, "key").Err(), goredis.Nil)
}

func TestMemoryConcurrentAcquisition(t *testing.T) {
	first, release := acquireMemoryClient("redis+memory://concurrent")
	defer release()
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			client, done := acquireMemoryClient("redis+memory://concurrent")
			defer done()
			if client != first {
				t.Error("different client for same URL")
			}
			if err := client.Incr(t.Context(), "count").Err(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	require.Equal(t, "20", first.Get(t.Context(), "count").Val())
}

func TestMemoryEndpointValidation(t *testing.T) {
	for _, endpoint := range []string{"redis+memory://", "redis+memory://cache/", "redis+memory://cache:6379", "redis+memory://user@cache", "redis+memory://cache?", "redis+memory://cache?a=1", "redis+memory://cache#", "redis+memory://cache#part", "redis://cache", "redis+memory://%zz", "redis+memory://cache/-1", "redis+memory://cache/16", "redis+memory://cache/100", "redis+memory://cache/1/2", "redis+memory://cache/abc", "redis+memory://cache/999999999999999999999999"} {
		t.Run(endpoint, func(t *testing.T) {
			require.Panics(t, func() { _, release := acquireMemoryClient(endpoint); release() })
		})
	}
}

func TestMemoryDatabaseUpperBoundary(t *testing.T) {
	client, release := acquireMemoryClient("redis+memory://boundary/15")
	defer release()
	require.Equal(t, 15, client.Options().DB)
	require.NoError(t, client.Set(t.Context(), "key", "value", 0).Err())
	require.Equal(t, "value", client.Get(t.Context(), "key").Val())
}

func TestMemoryClockTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, release := acquireMemoryClient("redis+memory://ttl")
		defer release()
		ctx := t.Context()
		require.NoError(t, client.Set(ctx, "ttl", "value", time.Second).Err())
		time.Sleep(500 * time.Millisecond)
		require.Equal(t, "value", client.Get(ctx, "ttl").Val())
		time.Sleep(501 * time.Millisecond)
		require.ErrorIs(t, client.Get(ctx, "ttl").Err(), goredis.Nil)
	})
}

func TestMemoryDatabases(t *testing.T) {
	ctx := t.Context()
	zero, releaseZero := acquireMemoryClient("redis+memory://databases")
	defer releaseZero()
	explicitZero, releaseExplicit := acquireMemoryClient("redis+memory://databases/0")
	defer releaseExplicit()
	require.Same(t, zero, explicitZero)
	one, releaseOne := acquireMemoryClient("redis+memory://databases/1")
	defer releaseOne()
	alias, releaseAlias := acquireMemoryClient("redis+memory://databases/01")
	defer releaseAlias()
	require.Same(t, one, alias)
	require.NotSame(t, zero, one)
	require.NoError(t, zero.Set(ctx, "key", "zero", 0).Err())
	require.ErrorIs(t, one.Get(ctx, "key").Err(), goredis.Nil)
	require.NoError(t, one.Set(ctx, "key", "one", 0).Err())
	require.Equal(t, "zero", zero.Get(ctx, "key").Val())
	require.Equal(t, "one", one.Get(ctx, "key").Val())
	require.True(t, zero.Move(ctx, "key", 2).Val(), "databases must belong to the same server")
	two, releaseTwo := acquireMemoryClient("redis+memory://databases/2")
	defer releaseTwo()
	require.Equal(t, "zero", two.Get(ctx, "key").Val())
	releaseOne()
	releaseAlias()
	reopened, releaseReopened := acquireMemoryClient("redis+memory://databases/1")
	defer releaseReopened()
	require.NotSame(t, one, reopened)
	require.Equal(t, "one", reopened.Get(ctx, "key").Val(), "other DB references must keep the server alive")
	releaseZero()
	releaseExplicit()
	releaseTwo()
	releaseReopened()
	fresh, releaseFresh := acquireMemoryClient("redis+memory://databases/1")
	defer releaseFresh()
	require.ErrorIs(t, fresh.Get(ctx, "key").Err(), goredis.Nil)
}

func TestMemoryPeriodicCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, release := acquireMemoryClient("redis+memory://cleanup")
		defer release()
		instance := memoryRedises["redis+memory://cleanup"]
		// Hold RedisDB pointers: their accessors do not trigger ClockTTL cleanup.
		// Reading through the client or server.DB below would mask a broken loop.
		zero, one := instance.server.DB(0), instance.server.DB(1)
		require.NoError(t, zero.Set("expired", "zero"))
		zero.SetTTL("expired", time.Second)
		require.NoError(t, one.Set("expired", "one"))
		one.SetTTL("expired", time.Second)
		require.NoError(t, one.Set("live", "value"))
		one.SetTTL("live", 90*time.Second)
		require.NoError(t, one.Set("persistent", "value"))
		synctest.Wait()
		time.Sleep(59 * time.Second)
		require.True(t, zero.Exists("expired"), "no access or cleanup tick yet")
		time.Sleep(time.Second)
		synctest.Wait()
		require.False(t, zero.Exists("expired"))
		require.False(t, one.Exists("expired"), "cleanup must cover nonzero databases")
		require.True(t, one.Exists("live"), "TTL must not be advanced twice")
		time.Sleep(60 * time.Second)
		synctest.Wait()
		require.False(t, one.Exists("live"))
		require.True(t, one.Exists("persistent"))
		release()
		select {
		case <-instance.cleanupDone:
		default:
			t.Fatal("release returned before cleanup stopped")
		}
		_, err := client.Options().Dialer(t.Context(), "tcp", "")
		require.ErrorIs(t, err, net.ErrClosed)
	})
}
