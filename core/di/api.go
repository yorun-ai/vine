package di

import (
	"reflect"

	internaldi "go.yorun.ai/vine/internal/core/di"
)

type (
	// Scope identifies a dependency lifetime.
	Scope = internaldi.Scope
	// SingletonScoped marks a type that is shared by an injector.
	SingletonScoped = internaldi.SingletonScoped
	// ExecutionScoped marks a type that is shared within one execution.
	ExecutionScoped = internaldi.ExecutionScoped
	// TransientScoped marks a type that is constructed for each resolution.
	TransientScoped = internaldi.TransientScoped
	// InitDefinition is implemented by dependencies requiring initialization.
	InitDefinition = internaldi.InitDefinition
	// DisposeDefinition is implemented by dependencies requiring disposal.
	DisposeDefinition = internaldi.DisposeDefinition
	// BindApplier adds bindings to a Binder.
	BindApplier = internaldi.BindApplier
	// Binder declares type, instance, and factory bindings.
	Binder = internaldi.Binder
	// Binding configures the target and scope of one binding.
	Binding = internaldi.Binding
	// Injector resolves dependencies by type.
	Injector = internaldi.Injector
	// PlainInjector is a root injector outside an execution scope.
	PlainInjector = internaldi.PlainInjector
	// ExecutionInjector resolves dependencies scoped to one execution.
	ExecutionInjector = internaldi.ExecutionInjector
	// ResolveContext tracks dependency resolution state.
	ResolveContext = internaldi.ResolveContext
	// SeedApplier adds seed values to a Seeder.
	SeedApplier = internaldi.SeedApplier
	// Seeder constructs values from an existing injector and explicit seeds.
	Seeder = internaldi.Seeder
)

const (
	// SingletonScope reuses one value for the lifetime of its injector.
	SingletonScope Scope = internaldi.SingletonScope
	// ExecutionScope reuses one value for the lifetime of an execution injector.
	ExecutionScope Scope = internaldi.ExecutionScope
	// TransientScope creates a value for every resolution.
	TransientScope Scope = internaldi.TransientScope
)

// T returns the reflection type for T without requiring a value of T.
func T[T any]() reflect.Type {
	return internaldi.T[T]()
}

// NewInjector creates a root injector and applies the supplied bindings in order.
func NewInjector(bindAppliers ...BindApplier) PlainInjector {
	return internaldi.NewInjector(bindAppliers...)
}
