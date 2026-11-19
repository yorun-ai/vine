package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type seederExecutionResource struct {
	ExecutionScoped
	value string
}

type seederSingletonResource struct {
	SingletonScoped
}

func TestSeederSeedsExecutionScopedInstance(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederExecutionResource]())
	})

	seeded := &seederExecutionResource{value: "seeded"}
	execution := injector.StartExecution(func(seeder *Seeder) {
		seeder.SeedInstance(seeded)
	})

	var resolved *seederExecutionResource
	execution.Resolve(&resolved)

	assert.Same(t, seeded, resolved)
}

func TestSeederRejectsNonExecutionScopedType(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederSingletonResource]())
	})

	assert.PanicsWithError(t,
		"type *di.seederSingletonResource is not execution scoped, cannot be seeded",
		func() {
			injector.StartExecution(func(seeder *Seeder) {
				seeder.SeedInstance(&seederSingletonResource{})
			})
		},
	)
}

func TestSeederRejectsSeedingAfterStartExecutionReturns(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederExecutionResource]())
	})

	var seeder *Seeder
	injector.StartExecution(func(current *Seeder) {
		seeder = current
	})

	assert.PanicsWithError(t,
		"execution injector is no longer accepting seeds",
		func() {
			seeder.SeedInstance(&seederExecutionResource{})
		},
	)
}
