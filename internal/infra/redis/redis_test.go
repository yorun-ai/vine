package redis

import (
	"reflect"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
)

type testRedis struct {
	Redis
}

func (*testRedis) InitOption(option *Option) {
	option.Endpoint = "redis://demo-user:demo-pass@127.0.0.1:6379/2"
}

func (*testRedis) InitLockers(add TypeAdder) {}

func initTestRedis(component app.FrameworkComponent) *RedisMinder {
	minder := new(RedisMinder)
	minder.InitComponent(component)
	return minder
}

func TestRedisMinderInitComponentInitializesOptionAndClient(t *testing.T) {
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
	minder := initTestRedis(component)
	t.Cleanup(minder.AfterAppStop)

	require.NotNil(t, minder.option)
	require.NotNil(t, minder.client)
	require.NotNil(t, gotOption)
	require.NotNil(t, component.Cmdable)
	assert.Equal(t, "redis://demo-user:demo-pass@127.0.0.1:6379/2", minder.option.Endpoint)
	assert.Same(t, minder.option, gotOption)
	assert.Same(t, minder.client, component.Cmdable)
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

func TestRedisMinderBindProvidesRedis(t *testing.T) {
	component := new(testRedis)
	minder := initTestRedis(component)
	t.Cleanup(minder.AfterAppStop)

	injector := di.NewInjector(func(b *di.Binder) {
		b.BindInstance(component)
		minder.Bind(b)
		b.Bind(reflect.TypeFor[*testConsumer]()).In(di.TransientScope)
	})

	consumer := injector.Get(reflect.TypeFor[*testConsumer]()).Interface().(*testConsumer)
	require.NotNil(t, consumer.Redis)
	require.NotNil(t, consumer.Redis.Cmdable)
	assert.Same(t, component, consumer.Redis)
	assert.Same(t, minder.client, consumer.Redis.Cmdable)
}
