package redis_test

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/redis"
)

func TestRedisNewCacheGenericMethod(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() { _ = client.Close() })
	component := &redis.Redis{Cmdable: client}

	cache := component.NewCache[string](context.Background(), "test")

	require.NotNil(t, cache)
}
