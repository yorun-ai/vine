package redis_test

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/redis"
)

type cacheCmdableStub struct {
	goredis.Cmdable

	lastKey string
}

func (c *cacheCmdableStub) Set(_ context.Context, key string, _ any, _ time.Duration) *goredis.StatusCmd {
	c.lastKey = key
	return goredis.NewStatusResult("OK", nil)
}

func TestRedisNewCacheGenericMethod(t *testing.T) {
	cmdable := new(cacheCmdableStub)
	component := &redis.Redis{Cmdable: cmdable}

	cache := component.NewCache[string](context.Background(), "test")
	cache.Set("1", "value", time.Minute)

	require.Equal(t, "vine:cache:test:1", cmdable.lastKey)
}
