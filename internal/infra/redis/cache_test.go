package redis

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/di"
)

type _TestCacheRedis struct {
	Redis
}

func (*_TestCacheRedis) InitOption(option *Option) {
	option.Endpoint = "redis://127.0.0.1:6379"
}

func (*_TestCacheRedis) InitCaches(add TypeAdder) {
	add(reflect.TypeFor[*_TestUserCache]())
}

type _TestUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type _TestUserCache struct {
	Cache[*_TestUser]
}

func (*_TestUserCache) KeyPrefix() string {
	return "user"
}

type _TestCacheConsumer struct {
	Cache *_TestUserCache `inject:""`
}

type _TestDefaultCache struct {
	Cache[*_TestUser]
}

type _TestEmptyPrefixCache struct {
	Cache[*_TestUser]
}

func (*_TestEmptyPrefixCache) KeyPrefix() string {
	return ""
}

type _TestCacheCmdable struct {
	goredis.Cmdable

	mutex    sync.Mutex
	getCalls []_TestGetCall
	setCalls []_TestSetCall
	delCalls []_TestDelCall
	getFunc  func(ctx context.Context, key string) *goredis.StringCmd
	setFunc  func(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd
	delFunc  func(ctx context.Context, keys ...string) *goredis.IntCmd
}

type _TestGetCall struct {
	ctx context.Context
	key string
}

type _TestSetCall struct {
	ctx        context.Context
	key        string
	value      interface{}
	expiration time.Duration
}

type _TestDelCall struct {
	ctx  context.Context
	keys []string
}

func newTestCacheCmdable() *_TestCacheCmdable {
	return &_TestCacheCmdable{
		getFunc: func(ctx context.Context, key string) *goredis.StringCmd {
			return goredis.NewStringResult("", goredis.Nil)
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd {
			return goredis.NewStatusResult("OK", nil)
		},
		delFunc: func(ctx context.Context, keys ...string) *goredis.IntCmd {
			return goredis.NewIntResult(1, nil)
		},
	}
}

func (c *_TestCacheCmdable) Get(ctx context.Context, key string) *goredis.StringCmd {
	c.mutex.Lock()
	c.getCalls = append(c.getCalls, _TestGetCall{ctx: ctx, key: key})
	c.mutex.Unlock()
	return c.getFunc(ctx, key)
}

func (c *_TestCacheCmdable) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd {
	c.mutex.Lock()
	c.setCalls = append(c.setCalls, _TestSetCall{ctx: ctx, key: key, value: value, expiration: expiration})
	c.mutex.Unlock()
	return c.setFunc(ctx, key, value, expiration)
}

func (c *_TestCacheCmdable) Del(ctx context.Context, keys ...string) *goredis.IntCmd {
	c.mutex.Lock()
	c.delCalls = append(c.delCalls, _TestDelCall{ctx: ctx, keys: append([]string(nil), keys...)})
	c.mutex.Unlock()
	return c.delFunc(ctx, keys...)
}

func TestNewCacheReturnsHandle(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}
	ctx := context.Background()

	cache := NewCache[*_TestUser](redis, ctx, "user")

	require.NotNil(t, cache)
	assert.Equal(t, ctx, cache.ctx)
	assert.Equal(t, "user", cache.keyPrefix)
	assert.Equal(t, cmdable, cache.cmdable)
}

func TestRedisNewCacheByTypeReturnsTypedHandle(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}
	ctx := context.Background()

	cache := redis.NewCacheByType(reflect.TypeFor[*_TestUserCache](), ctx).(*_TestUserCache)

	require.NotNil(t, cache)
	assert.Equal(t, ctx, cache.ctx)
	assert.Equal(t, "user", cache.keyPrefix)
	assert.Equal(t, cmdable, cache.cmdable)
}

func TestRedisNewCacheByTypeUsesDefaultTypePrefixWhenNotOverridden(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}

	cache := redis.NewCacheByType(reflect.TypeFor[*_TestDefaultCache](), context.Background()).(*_TestDefaultCache)

	assert.Equal(t, "go.yorun.ai_vine_internal_infra_redis._TestDefaultCache", cache.keyPrefix)
}

func TestRedisMinderBindProvidesCache(t *testing.T) {
	original := newRedisClient
	t.Cleanup(func() {
		newRedisClient = original
	})

	newRedisClient = func(opt *Option) *goredis.Client {
		return goredis.NewClient(endpointOptions(opt.Endpoint))
	}

	component := new(_TestCacheRedis)
	minder := new(RedisMinder)
	minder.InitComponent(component)
	t.Cleanup(minder.AfterAppStop)

	injector := di.NewInjector(func(b *di.Binder) {
		b.Bind(reflect.TypeFor[context.Context]()).ToInstance(context.Background())
		minder.Bind(b)
		b.Bind(reflect.TypeFor[*_TestCacheConsumer]()).In(di.TransientScope)
	})

	consumer := injector.Get(reflect.TypeFor[*_TestCacheConsumer]()).Interface().(*_TestCacheConsumer)
	require.NotNil(t, consumer.Cache)
	assert.Equal(t, "user", consumer.Cache.keyPrefix)
	require.NotNil(t, consumer.Cache.ctx)
	require.NotNil(t, consumer.Cache.cmdable)
}

func TestInstantiateCacheUsesDefaultTypePrefixWhenNotOverridden(t *testing.T) {
	minder := &RedisMinder{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	minder.component = &Redis{Cmdable: minder.client}
	t.Cleanup(func() {
		_ = minder.client.Close()
	})

	cache := minder.instantiateCache(reflect.TypeFor[*_TestDefaultCache](), context.Background()).(*_TestDefaultCache)

	assert.Equal(t, "go.yorun.ai_vine_internal_infra_redis._TestDefaultCache", cache.keyPrefix)
}

func TestInstantiateCacheRequiresNonEmptyOverriddenPrefix(t *testing.T) {
	minder := &RedisMinder{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	minder.component = &Redis{Cmdable: minder.client}
	t.Cleanup(func() {
		_ = minder.client.Close()
	})

	assert.Panics(t, func() {
		minder.instantiateCache(reflect.TypeFor[*_TestEmptyPrefixCache](), context.Background())
	})
}

func TestCacheGetReturnsValue(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cmdable.getFunc = func(ctx context.Context, key string) *goredis.StringCmd {
		return goredis.NewStringResult(`{"id":"1","name":"demo"}`, nil)
	}
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	value, ok := cache.Get("1")

	require.True(t, ok)
	assert.Equal(t, &_TestUser{ID: "1", Name: "demo"}, value)
	require.Len(t, cmdable.getCalls, 1)
	assert.Equal(t, "vine:cache:user:1", cmdable.getCalls[0].key)
}

func TestCacheGetReturnsFalseWhenMissing(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	value, ok := cache.Get("1")

	require.False(t, ok)
	assert.Nil(t, value)
}

func TestCacheSetMarshalsValue(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	cache.Set("1", &_TestUser{ID: "1", Name: "demo"}, time.Minute)

	require.Len(t, cmdable.setCalls, 1)
	assert.Equal(t, "vine:cache:user:1", cmdable.setCalls[0].key)
	assert.Equal(t, time.Minute, cmdable.setCalls[0].expiration)
	assert.JSONEq(t, `{"id":"1","name":"demo"}`, string(cmdable.setCalls[0].value.([]byte)))
}

func TestCacheDeleteUsesNamespacedKey(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	cache.Delete("1")

	require.Len(t, cmdable.delCalls, 1)
	assert.Equal(t, []string{"vine:cache:user:1"}, cmdable.delCalls[0].keys)
}

func TestCacheGetOrLoadReturnsCachedValue(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cmdable.getFunc = func(ctx context.Context, key string) *goredis.StringCmd {
		return goredis.NewStringResult(`{"id":"1","name":"cached"}`, nil)
	}
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}
	loaded := false

	value := cache.GetOrLoad("1", time.Minute, func() *_TestUser {
		loaded = true
		return &_TestUser{ID: "1", Name: "loaded"}
	})

	assert.Equal(t, &_TestUser{ID: "1", Name: "cached"}, value)
	assert.False(t, loaded)
	assert.Empty(t, cmdable.setCalls)
}

func TestCacheGetOrLoadLoadsAndSetsWhenMissing(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}
	loaded := false

	value := cache.GetOrLoad("1", time.Minute, func() *_TestUser {
		loaded = true
		return &_TestUser{ID: "1", Name: "loaded"}
	})

	assert.Equal(t, &_TestUser{ID: "1", Name: "loaded"}, value)
	assert.True(t, loaded)
	require.Len(t, cmdable.setCalls, 1)
	assert.Equal(t, "vine:cache:user:1", cmdable.setCalls[0].key)
	assert.JSONEq(t, `{"id":"1","name":"loaded"}`, string(cmdable.setCalls[0].value.([]byte)))
}

func TestCacheGetPanicsOnInvalidJson(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cmdable.getFunc = func(ctx context.Context, key string) *goredis.StringCmd {
		return goredis.NewStringResult(`{`, nil)
	}
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	assert.Panics(t, func() {
		cache.Get("1")
	})
}

func TestCacheDeletePanicsOnRedisError(t *testing.T) {
	cmdable := newTestCacheCmdable()
	cmdable.delFunc = func(ctx context.Context, keys ...string) *goredis.IntCmd {
		return goredis.NewIntResult(0, errors.New("delete failed"))
	}
	cache := &Cache[*_TestUser]{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "user",
	}

	assert.Panics(t, func() {
		cache.Delete("1")
	})
}
