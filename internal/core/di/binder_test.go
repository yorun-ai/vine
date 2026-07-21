package di

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type binderTestService interface {
	Run() string
}

type binderTestImpl struct {
	SingletonScoped
}

func (s *binderTestImpl) Run() string {
	return "ok"
}

type binderTestResource struct {
	SingletonScoped
}

func TestBinderBindFactoryUsesFirstReturnType(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.BindFactory(func() *binderTestResource {
			return &binderTestResource{}
		})
	}).(*_PlainInjector)

	binding := injector.bindings[T[*binderTestResource]()]
	if assert.NotNil(t, binding) {
		assert.Equal(t, T[*binderTestResource](), binding.targetType)
		assert.NotNil(t, binding.factory)
	}
}

func TestBinderBindFactoryPanicsForNonFunc(t *testing.T) {
	binder := newBinder(&_PlainInjector{bindings: map[reflect.Type]*Binding{}})

	assert.PanicsWithError(t,
		"factory must be function",
		func() {
			binder.BindFactory(123)
		},
	)
}

func TestBinderBindFactoryPanicsWithoutReturnValue(t *testing.T) {
	binder := newBinder(&_PlainInjector{bindings: map[reflect.Type]*Binding{}})

	assert.PanicsWithError(t,
		"factory must return at least one value",
		func() {
			binder.BindFactory(func() {})
		},
	)
}

func TestBindingToInstanceSetsSingletonFactory(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]()).ToInstance(&binderTestResource{})

	assert.Equal(t, SingletonScope, binding.explicitScope)
	assert.NotNil(t, binding.factory)
	assert.Nil(t, binding.factoryDependenciesOverridden)
}

func TestBindingToImplementationOverridesFactoryDeps(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[binderTestService]()).ToImplementation(T[*binderTestImpl]())

	assert.Equal(t, []reflect.Type{T[*binderTestImpl]()}, binding.factoryDependenciesOverridden)
	assert.NotNil(t, binding.factory)
}

func TestBindingToImplementationPanicsForNonInterfaceTarget(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]())

	assert.PanicsWithError(t,
		"only interface target type can bind to implementation",
		func() {
			binding.ToImplementation(T[*binderTestImpl]())
		},
	)
}

func TestBindingAsNullableMarksBinding(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]()).AsNullable()

	assert.True(t, binding.isNullable)
}

func TestBindingAsNullablePanicsForNonNilableType(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinding(injector, T[int](), false)

	assert.PanicsWithError(t,
		"type int cannot be nullable",
		func() {
			binding.AsNullable()
		},
	)
}

func TestBindingWithDisposerAcceptsMatchingSignature(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]())

	assert.NotPanics(t, func() {
		binding.WithDisposer(func(*binderTestResource) error {
			return nil
		})
	})

	assert.NotNil(t, binding.disposer)
}

func TestBindingWithDisposerPanicsWhenDisposeDefinitionAlreadyImplemented(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*boundTestDisposable]())

	assert.PanicsWithError(t,
		"type *di.boundTestDisposable already implements di.DisposeDefinition, manual disposer is not allowed",
		func() {
			binding.WithDisposer(func(*boundTestDisposable) {})
		},
	)
}

func TestBindingWithDisposerPanicsForMismatchedArgument(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]())

	assert.PanicsWithError(t,
		"the only argument of func(*di.binderTestImpl) must be *di.binderTestImpl",
		func() {
			binding.WithDisposer(func(*binderTestImpl) {})
		},
	)
}

func TestBindingToFactoryAllowsErrorReturningFactory(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]())

	assert.NotPanics(t, func() {
		binding.ToFactory(func() (*binderTestResource, error) {
			return nil, errors.New("boom")
		})
	})

	assert.NotNil(t, binding.factory)
}

func TestBindingToAbstractFactoryPanicsForNonInterfaceTarget(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[*binderTestResource]())

	assert.PanicsWithError(t,
		"only interface target type can bind to abstract factory",
		func() {
			binding.ToAbstractFactory(func(ResolveContext) *binderTestResource {
				return &binderTestResource{}
			})
		},
	)
}

func TestBindingToAbstractFactoryConflictsWithRegularFactoryMode(t *testing.T) {
	injector := &_PlainInjector{bindings: map[reflect.Type]*Binding{}}
	binding := newBinder(injector).Bind(T[binderTestService]()).ToAbstractFactory(func(ResolveContext) binderTestService {
		return &binderTestImpl{}
	})

	assert.PanicsWithError(t,
		"binding mode of di.binderTestService conflicts with existing configuration",
		func() {
			binding.ToImplementation(T[*binderTestImpl]())
		},
	)
}

func TestBindingRejectsChangesAfterInjectorInitialization(t *testing.T) {
	var resourceBinding *Binding
	var serviceBinding *Binding
	NewInjector(func(b *Binder) {
		resourceBinding = b.Bind(T[*binderTestResource]())
		serviceBinding = b.Bind(T[binderTestService]())
	})

	tests := []struct {
		name    string
		binding *Binding
		change  func()
	}{
		{name: "scope", binding: resourceBinding, change: func() { resourceBinding.In(ExecutionScope) }},
		{name: "nullable", binding: resourceBinding, change: func() { resourceBinding.AsNullable() }},
		{name: "factory", binding: resourceBinding, change: func() {
			resourceBinding.ToFactory(func() *binderTestResource { return &binderTestResource{} })
		}},
		{name: "instance", binding: resourceBinding, change: func() {
			resourceBinding.ToInstance(&binderTestResource{})
		}},
		{name: "disposer", binding: resourceBinding, change: func() {
			resourceBinding.WithDisposer(func(*binderTestResource) {})
		}},
		{name: "implementation", binding: serviceBinding, change: func() {
			serviceBinding.ToImplementation(T[*binderTestImpl]())
		}},
		{name: "abstract factory", binding: serviceBinding, change: func() {
			serviceBinding.ToAbstractFactory(func(ResolveContext) binderTestService { return &binderTestImpl{} })
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.True(t, test.binding.frozen)
			assert.PanicsWithError(t,
				"binding of "+test.binding.targetType.String()+" is already frozen",
				test.change,
			)
		})
	}
}
