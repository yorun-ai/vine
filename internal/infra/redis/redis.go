package redis

import (
	"context"
	"reflect"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	Endpoint string
}

type TypeAdder func(lockerType reflect.Type)

func defaultOption() *Option {
	return new(Option)
}

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

type Redis struct {
	app.BaseFrameworkComponent[*RedisMinder]
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

type RedisMinder struct {
	app.BaseFrameworkComponentMinder

	component   app.FrameworkComponent
	option      *Option
	client      *goredis.Client
	lockerTypes []reflect.Type
	cacheTypes  []reflect.Type
}

func (m *RedisMinder) InitComponent(component app.FrameworkComponent) {
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

	m.client = newRedisClient(m.option)
	component.(_RedisAccessor).setCmdable(m.client)
}

func (m *RedisMinder) Component() app.FrameworkComponent {
	return m.component
}

func (m *RedisMinder) Bind(b *di.Binder) {
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

func (m *RedisMinder) AfterAppStop() {
	_ = m.client.Close()
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
