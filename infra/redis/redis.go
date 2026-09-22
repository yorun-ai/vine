package redis

import (
	"context"
	"reflect"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/util/vpre"
)

// Option configures a Redis component.
type Option struct {
	// Endpoint accepts a Redis URL or host:port.
	// redis+memory://name[/dbIndex] selects a process-shared, ephemeral instance.
	// The memory database index must be between 0 and 15 and defaults to zero.
	Endpoint string
}

// TypeAdder adds a Redis capability type to a specification.
type TypeAdder func(lockerType reflect.Type)

func defaultOption() *Option {
	return new(Option)
}

// RedisSpec describes a named Redis connection and its capabilities.
type RedisSpec interface {
	InitOption(option *Option)
	InitLockers(add TypeAdder)
	InitCaches(add TypeAdder)

	mustBeRedis()
}

type _RedisAccessor interface {
	embeddedRedis() *Redis
	setCmdable(cmdable goredis.Cmdable)
}

// Redis wraps a Redis client for Vine-managed execution contexts.
type Redis struct {
	app.BaseManagedComponent[*RedisManager]
	goredis.Cmdable
}

func (*Redis) InitOption(option *Option) {}

func (*Redis) InitLockers(add TypeAdder) {}

func (*Redis) InitCaches(add TypeAdder) {}

func (*Redis) mustBeRedis() {}

func (r *Redis) setCmdable(cmdable goredis.Cmdable) {
	r.Cmdable = cmdable
}

func (r *Redis) embeddedRedis() *Redis {
	return r
}

// RedisManager owns the Redis client and cache and lock dependency bindings.
type RedisManager struct {
	app.BaseComponentManager

	component   app.ManagedComponent
	option      *Option
	client      *goredis.Client
	release     func()
	lockerTypes []reflect.Type
	cacheTypes  []reflect.Type
}

func (m *RedisManager) InitComponent(component app.ManagedComponent) {
	m.component = component
	m.option = defaultOption()
	m.lockerTypes = []reflect.Type{}
	m.cacheTypes = []reflect.Type{}

	spec := component.(RedisSpec)
	spec.InitOption(m.option)
	vpre.Check(m.option.Endpoint != "", "redis endpoint is empty")

	spec.InitLockers(func(lockerType reflect.Type) {
		m.lockerTypes = append(m.lockerTypes, lockerType)
	})
	spec.InitCaches(func(cacheType reflect.Type) {
		m.cacheTypes = append(m.cacheTypes, cacheType)
	})

	if strings.HasPrefix(strings.ToLower(m.option.Endpoint), "redis+memory:") {
		m.client, m.release = acquireMemoryClient(m.option.Endpoint)
	} else {
		m.client = newRedisClient(m.option)
		m.release = func() { _ = m.client.Close() }
	}
	component.(_RedisAccessor).setCmdable(m.client)
}

func (m *RedisManager) Component() app.ManagedComponent {
	return m.component
}

func (m *RedisManager) Bind(b *di.Binder) {
	for _, lockerType := range m.lockerTypes {
		kind := lockerType
		b.Bind(kind).ToFactory(func(ctx context.Context) any {
			return m.instantiateLocker(kind, ctx)
		})
	}
	for _, cacheType := range m.cacheTypes {
		kind := cacheType
		b.Bind(kind).ToFactory(func(ctx context.Context) any {
			return m.instantiateCache(kind, ctx)
		})
	}
}

func (m *RedisManager) AfterAppStop() {
	m.release()
}

var newRedisClient = func(opt *Option) *goredis.Client {
	return goredis.NewClient(endpointOptions(opt.Endpoint))
}

func endpointOptions(endpoint string) *goredis.Options {
	options, err := goredis.ParseURL(endpoint)
	if err != nil {
		options = &goredis.Options{Addr: endpoint}
	}
	vpre.Check(options.Addr != "", "redis endpoint host is empty")
	options.Protocol = 2
	options.DisableIdentity = true
	return options
}
