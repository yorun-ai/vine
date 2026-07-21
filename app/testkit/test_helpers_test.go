package testkit

import (
	"reflect"
	"sync"

	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/core/conf"
	"go.yorun.ai/vine/core/rpc"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

type _TestAppSpec struct {
	app.Application
}

func (*_TestAppSpec) Name() string {
	return "testkit.test"
}

type _TestConfig struct {
	conf.ConfigModel
	DSN string `json:"dsn"`
}

type _TestServiceClient interface {
	test()
}

type _TestServiceClientER interface {
	testER()
}

type _TestServiceClientImpl struct{}

func (*_TestServiceClientImpl) test() {}

type _TestServiceClientERImpl struct {
	client *rpc.Client
}

func (*_TestServiceClientERImpl) testER() {}

var registerTestFixturesOnce sync.Once

func registerTestFixtures() {
	registerTestFixturesOnce.Do(func() {
		conf.Register(conf.ConfigSpec{
			Name:      "TestConfig",
			SkelName:  "testkit.TestConfig",
			Hash:      "test",
			Lifecycle: conf.LifecycleEternal,
			Type:      reflect.TypeFor[*_TestConfig](),
		})
		rpc.Register(&rpc.ServiceSpec{
			Type:         rpc.ServiceSpecTypeClient,
			Name:         "TestService",
			SkelName:     "testkit.TestService",
			Hash:         "test",
			ClientType:   reflect.TypeFor[_TestServiceClient](),
			ClientCtor:   func(_ _TestServiceClientER) _TestServiceClient { return &_TestServiceClientImpl{} },
			ERClientType: reflect.TypeFor[_TestServiceClientER](),
			ERClientCtor: func(client *rpc.Client) _TestServiceClientER { return &_TestServiceClientERImpl{client: client} },
			Methods:      []*rpcspec.MethodSpec{},
		})
	})
}
