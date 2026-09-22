package redis

import (
	"reflect"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/core/di"
)

type testRedis struct {
	Redis
}

func (*testRedis) InitOption(option *Option) {
	option.Endpoint = "redis://demo-user:demo-pass@127.0.0.1:6379/2"
}

func (*testRedis) InitLockers(add TypeAdder) {}

func initTestRedis(component app.ManagedComponent) *RedisManager {
	manager := new(RedisManager)
	manager.InitComponent(component)
	return manager
}

func TestRedisManagerInitComponentInitializesOptionAndClient(t *testing.T) {
	original := newRedisClient
	t.Cleanup(func() {
		newRedisClient = original
	})

	var gotOption *Option
	newRedisClient = func(opt *Option) *goredis.Client {
		gotOption = opt
		return goredis.NewClient(endpointOptions(opt.Endpoint))
	}

	component := new(testRedis)
	manager := initTestRedis(component)
	t.Cleanup(manager.AfterAppStop)

	require.NotNil(t, manager.option)
	require.NotNil(t, manager.client)
	require.NotNil(t, gotOption)
	require.NotNil(t, component.Cmdable)
	assert.Equal(t, "redis://demo-user:demo-pass@127.0.0.1:6379/2", manager.option.Endpoint)
	assert.Same(t, manager.option, gotOption)
	assert.Same(t, manager.client, component.Cmdable)
}

func TestEndpointOptions(t *testing.T) {
	options := endpointOptions("127.0.0.1:6379")
	assert.Equal(t, "127.0.0.1:6379", options.Addr)
	assert.Equal(t, 2, options.Protocol)
	assert.True(t, options.DisableIdentity)

	options = endpointOptions("redis://demo-user:demo-pass@127.0.0.1:6379/2")
	assert.Equal(t, "127.0.0.1:6379", options.Addr)
	assert.Equal(t, "demo-user", options.Username)
	assert.Equal(t, "demo-pass", options.Password)
	assert.Equal(t, 2, options.DB)
	assert.Equal(t, 2, options.Protocol)
	assert.True(t, options.DisableIdentity)
}

type testConsumer struct {
	Redis *testRedis `inject:""`
}

func TestRedisManagerBindProvidesRedis(t *testing.T) {
	component := new(testRedis)
	manager := initTestRedis(component)
	t.Cleanup(manager.AfterAppStop)

	injector := di.NewInjector(func(b *di.Binder) {
		b.BindInstance(component)
		manager.Bind(b)
		b.Bind(reflect.TypeFor[*testConsumer]()).In(di.TransientScope)
	})

	consumer := injector.Get(reflect.TypeFor[*testConsumer]()).Interface().(*testConsumer)
	require.NotNil(t, consumer.Redis)
	require.NotNil(t, consumer.Redis.Cmdable)
	assert.Same(t, component, consumer.Redis)
	assert.Same(t, manager.client, consumer.Redis.Cmdable)
}

type memoryRedis struct{ Redis }

func (*memoryRedis) InitOption(option *Option) { option.Endpoint = "redis+memory://cache" }

func TestMemoryRedisComponentsShareClient(t *testing.T) {
	first, second := new(memoryRedis), new(memoryRedis)
	a, b := new(RedisManager), new(RedisManager)
	a.InitComponent(first)
	defer a.AfterAppStop()
	b.InitComponent(second)
	defer b.AfterAppStop()
	require.Same(t, a.client, b.client)
	require.NoError(t, first.Set(t.Context(), "shared", "value", 0).Err())
	a.AfterAppStop()
	require.Equal(t, "value", second.Get(t.Context(), "shared").Val())
	b.AfterAppStop()
	c := new(RedisManager)
	c.InitComponent(new(memoryRedis))
	defer c.AfterAppStop()
	require.NotSame(t, b.client, c.client)
	require.EqualValues(t, 0, c.client.Exists(t.Context(), "shared").Val())
}

func TestExternalRedisComponentsShareClient(t *testing.T) {
	first := initTestRedis(new(testRedis))
	defer first.AfterAppStop()
	second := initTestRedis(new(testRedis))
	defer second.AfterAppStop()
	require.Same(t, first.client, second.client)
	first.AfterAppStop()
	first.AfterAppStop()
	third := initTestRedis(new(testRedis))
	defer third.AfterAppStop()
	require.Same(t, second.client, third.client)
	second.AfterAppStop()
	third.AfterAppStop()
	fresh := initTestRedis(new(testRedis))
	defer fresh.AfterAppStop()
	require.NotSame(t, first.client, fresh.client)
}
