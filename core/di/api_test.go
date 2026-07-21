package di

import (
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
