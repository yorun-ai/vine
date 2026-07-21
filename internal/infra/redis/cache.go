package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

const (
	cacheKeyPrefixGlobal = "vine:cache:"
	// cacheKeyPrefixSentinel marks caches that still use the base
	// Cache.KeyPrefix implementation, so we can derive a type-based prefix.
	cacheKeyPrefixSentinel = "\x00"
)

type _CacheSpec interface {
	KeyPrefix() string
	configure(ctx context.Context, cmdable goredis.Cmdable, keyPrefix string)
}

type Cache[T any] struct {
	ctx       context.Context
	cmdable   goredis.Cmdable
	keyPrefix string
}

func (c *Cache[T]) KeyPrefix() string {
	// By default, each cache type gets a unique key prefix derived from its full
	// type name. If multiple cache types need to share the same Redis namespace,
	// they must override KeyPrefix() to return the same value.
	return cacheKeyPrefixSentinel
}

func (r *Redis) NewCacheByType(cacheType reflect.Type, ctx context.Context) any {
	vpre.CheckNotNil(ctx, "redis cache context is nil")
	vpre.Check(cacheType.Kind() == reflect.Pointer, "redis cache type %s must be pointer", cacheType)

	cacheValue := reflect.New(cacheType.Elem())
	cache := cacheValue.Interface()
	cacheSpec, ok := cache.(_CacheSpec)
	vpre.Check(ok, "cache type %s must embed redis.Cache", cacheType)
	keyPrefix := cacheSpec.KeyPrefix()
	if keyPrefix == cacheKeyPrefixSentinel {
		keyPrefix = defaultCacheTypeKeyPrefix(cacheType)
	}
	vpre.CheckNotEmpty(keyPrefix, "redis cache key prefix is empty")
	cacheSpec.configure(ctx, r.Cmdable, keyPrefix)
	return cache
}

func NewCache[T any](redis *Redis, ctx context.Context, keyPrefix string) *Cache[T] {
	vpre.CheckNotNil(redis, "redis is nil")
	vpre.CheckNotNil(ctx, "redis cache context is nil")
	vpre.CheckNotEmpty(keyPrefix, "redis cache key prefix is empty")
	return &Cache[T]{
		ctx:       ctx,
		cmdable:   redis.Cmdable,
		keyPrefix: keyPrefix,
	}
}

func (m *RedisMinder) instantiateCache(cacheType reflect.Type, ctx context.Context) any {
	return m.component.(_RedisAccessor).embeddedRedis().NewCacheByType(cacheType, ctx)
}

func defaultCacheTypeKeyPrefix(cacheType reflect.Type) string {
	kind := cacheType.Elem()
	return strings.ReplaceAll(kind.PkgPath()+"."+kind.Name(), "/", "_")
}

func (c *Cache[T]) configure(ctx context.Context, cmdable goredis.Cmdable, keyPrefix string) {
	c.ctx = ctx
	c.cmdable = cmdable
	c.keyPrefix = keyPrefix
}

func (c *Cache[T]) Get(key string) (T, bool) {
	value, err := c.cmdable.Get(c.ctx, joinCacheKey(c.keyPrefix, key)).Result()
	if errors.Is(err, goredis.Nil) {
		return *new(T), false
	}
	vpre.CheckNilError(err, "get redis cache failed")

	return vcode.MustUnmarshalJsonS[T](value), true
}

func (c *Cache[T]) GetOrLoad(key string, ttl time.Duration, load func() T) T {
	value, ok := c.Get(key)
	if ok {
		return value
	}

	value = load()
	c.Set(key, value, ttl)
	return value
}

func (c *Cache[T]) Set(key string, value T, ttl time.Duration) {
	data, err := json.Marshal(value)
	vpre.CheckNilError(err, "marshal redis cache failed")
	err = c.cmdable.Set(c.ctx, joinCacheKey(c.keyPrefix, key), data, ttl).Err()
	vpre.CheckNilError(err, "set redis cache failed")
}

func (c *Cache[T]) Delete(key string) {
	err := c.cmdable.Del(c.ctx, joinCacheKey(c.keyPrefix, key)).Err()
	vpre.CheckNilError(err, "delete redis cache failed")
}

// joinCacheKey combines the global cache namespace with the cache prefix and
// resource key. For example:
//
//	joinCacheKey("user", "123") == "vine:cache:user:123"
func joinCacheKey(keyPrefix string, key string) string {
	return fmt.Sprintf("%s%s:%s", cacheKeyPrefixGlobal, keyPrefix, key)
}
