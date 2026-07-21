package di

import (
	"reflect"
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

type seederOtherExecutionResource struct {
	ExecutionScoped
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
	assert.Equal(t, "seeded", resolved.value)
}

func TestSeederRequiresExecutionInjector(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederExecutionResource]())
	})

	execution := injector.StartExecution()
	assert.IsType(t, &_ExecutionInjector{}, execution)
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

func TestSeederRejectsIncompatibleInstanceWithFriendlyError(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederExecutionResource]())
	})

	assert.PanicsWithError(t,
		"instance type *di.seederOtherExecutionResource is not compatible with *di.seederExecutionResource",
		func() {
			injector.StartExecution(func(seeder *Seeder) {
				seeder.Seed(T[*seederExecutionResource](), &seederOtherExecutionResource{})
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

func TestSeederSeedAcceptsExplicitTargetType(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*seederExecutionResource]())
	})

	seeded := reflect.ValueOf(&seederExecutionResource{value: "typed"})
	execution := injector.StartExecution(func(seeder *Seeder) {
		seeder.seed(T[*seederExecutionResource](), seeded)
	})

	var resolved *seederExecutionResource
	execution.Resolve(&resolved)

	assert.Equal(t, "typed", resolved.value)
}
