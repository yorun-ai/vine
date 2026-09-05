package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type diFacadeConfig struct {
	SingletonScoped

	Name string
}

type diFacadeRequest struct {
	ExecutionScoped

	Value string
}

type diFacadeService struct {
	SingletonScoped

	Config  *diFacadeConfig  `inject:""`
	Request *diFacadeRequest `inject:""`
	Reader  Injector         `inject:""`
}

func TestNewInjectorFacadeSupportsPlainAndExecutionInterfaces(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.BindInstance(&diFacadeConfig{Name: "vine"})
		b.Bind(T[*diFacadeRequest]())
		b.Bind(T[*diFacadeService]()).In(ExecutionScope)
	})

	var plain PlainInjector = injector
	assert.NotNil(t, plain)

	execution := plain.StartExecution(func(s *Seeder) {
		s.SeedInstance(&diFacadeRequest{Value: "hello"})
	})

	var service *diFacadeService
	execution.Resolve(&service)

	if assert.NotNil(t, service) {
		assert.Equal(t, "vine", service.Config.Name)
		assert.Equal(t, "hello", service.Request.Value)
		assert.NotNil(t, service.Reader)
	}
}

func TestBindingWithDependenciesFacade(t *testing.T) {
	calls := 0
	injector := NewInjector(func(b *Binder) {
		b.BindInstance(&diFacadeConfig{Name: "configured"})
		b.Bind(T[*diFacadeRequest]()).WithDependencies(
			[]reflect.Type{T[*diFacadeConfig]()},
			func(value reflect.Value, dependencies []reflect.Value) {
				value.Interface().(*diFacadeRequest).Value = dependencies[0].Interface().(*diFacadeConfig).Name
				calls++
			},
		).In(SingletonScope)
	})
	first := injector.Get(T[*diFacadeRequest]()).Interface().(*diFacadeRequest)
	second := injector.Get(T[*diFacadeRequest]()).Interface().(*diFacadeRequest)
	assert.Equal(t, "configured", first.Value)
	assert.Same(t, first, second)
	assert.Equal(t, 1, calls)
}
