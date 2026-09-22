package redis

import (
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestExternalConnectionReferences(t *testing.T) {
	server := miniredis.RunT(t)
	option := &Option{Endpoint: "redis://" + server.Addr() + "/0"}
	first, releaseFirst := acquireRedisClient(option)
	defer releaseFirst()
	second, releaseSecond := acquireRedisClient(option)
	defer releaseSecond()
	require.Same(t, first, second)
	require.NoError(t, first.Set(t.Context(), "key", "value", 0).Err())
	releaseFirst()
	releaseFirst()
	require.Equal(t, "value", second.Get(t.Context(), "key").Val())
	different, releaseDifferent := acquireRedisClient(&Option{Endpoint: "redis://" + server.Addr() + "/1"})
	defer releaseDifferent()
	require.NotSame(t, first, different)
	require.ErrorIs(t, different.Get(t.Context(), "key").Err(), goredis.Nil)
	releaseSecond()
	require.Error(t, second.Ping(t.Context()).Err())
	fresh, releaseFresh := acquireRedisClient(option)
	defer releaseFresh()
	require.NotSame(t, first, fresh)
	require.Equal(t, "value", fresh.Get(t.Context(), "key").Val(), "closing a pool must preserve external data")
}

func TestExternalConnectionConcurrentAcquisition(t *testing.T) {
	server := miniredis.RunT(t)
	option := &Option{Endpoint: server.Addr()}
	first, release := acquireRedisClient(option)
	defer release()
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			client, done := acquireRedisClient(option)
			defer done()
			if client != first {
				t.Error("same endpoint returned different clients")
			}
			if err := client.Incr(t.Context(), "count").Err(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	require.Equal(t, "20", first.Get(t.Context(), "count").Val())
}
