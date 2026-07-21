package di

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type boundTestDependency struct {
	SingletonScoped
}

type boundTestResource struct {
	ExecutionScoped
	Dependency  *boundTestDependency `inject:""`
	initialized bool
}

func (r *boundTestResource) DIInit() {
	r.initialized = true
}

type boundTestDisposable struct {
	TransientScoped
	disposed *[]string
}

func (r *boundTestDisposable) DIDispose() {
	*r.disposed = append(*r.disposed, "default")
}

type boundTestGateway interface {
	Run() string
}

type boundTestGatewayImpl struct {
	SingletonScoped
}

func (g *boundTestGatewayImpl) Run() string {
	return "gateway"
}

type _BoundFactoryDisposable interface {
	Run()
}

type _BoundFactoryDisposableImpl struct {
	disposed *bool
}

func (*_BoundFactoryDisposableImpl) Run() {}

func (i *_BoundFactoryDisposableImpl) DIDispose() {
	*i.disposed = true
}

func TestBoundUsesDeclaredScopeOverDefaultScope(t *testing.T) {
	binding := newBinding(&_PlainInjector{fallbackScope: TransientScope}, T[*boundTestResource](), false)
	binding.In(SingletonScope)

	bound := newBound(binding)

	assert.Equal(t, SingletonScope, bound.ResolveScope(TransientScope))
	assert.Equal(t, ExecutionScope, bound.declaredScope)
}

func TestBoundUsesInjectorFallbackScopeWhenNoDeclaredOrExplicitScope(t *testing.T) {
	binding := newBinding(&_PlainInjector{fallbackScope: ExecutionScope}, T[*scopeNoMarker](), false)

	bound := newBound(binding)

	assert.Equal(t, noScope, bound.DeclaredScope())
	assert.Equal(t, ExecutionScope, bound.ResolveScope(ExecutionScope))
}

func TestBoundDepsIncludeInjectedFieldsAndFactoryDeps(t *testing.T) {
	binding := newBinding(&_PlainInjector{}, T[boundTestGateway](), false)
	binding.ToFactory(func(dep *boundTestDependency, impl *boundTestGatewayImpl) boundTestGateway {
		return impl
	})

	bound := newBound(binding)

	assert.ElementsMatch(t, []reflect.Type{
		T[*boundTestDependency](),
		T[*boundTestGatewayImpl](),
	}, bound.Dependencies())
}

func TestBoundFactoryDoesNotDependOnInjectedFieldsOfTargetType(t *testing.T) {
	binding := newBinding(&_PlainInjector{}, T[*boundTestResource](), false)
	binding.ToFactory(func() *boundTestResource {
		return &boundTestResource{}
	})

	bound := newBound(binding)

	assert.Empty(t, bound.Dependencies())
}

func TestBoundBuildDefInstantiateFuncInjectsFieldsAndInit(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*boundTestDependency]())
		b.Bind(T[*boundTestResource]())
	}).(*_PlainInjector)

	bound := injector.bounds[T[*boundTestResource]()]
	instance := bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestResource]()}, T[*boundTestResource]())
	resource := instance.Interface().(*boundTestResource)

	assert.NotNil(t, resource.Dependency)
	assert.True(t, resource.initialized)
}

func TestBoundBuildFactoryInstantiateFuncCallsInit(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*boundTestResource]()).ToFactory(func() *boundTestResource {
			return &boundTestResource{}
		})
	}).(*_PlainInjector)

	bound := injector.bounds[T[*boundTestResource]()]
	instance := bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestResource]()}, T[*boundTestResource]())
	resource := instance.Interface().(*boundTestResource)

	assert.True(t, resource.initialized)
}

func TestBoundBuildFactoryInstantiateFuncAllowsNullableNilPointer(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*boundTestResource]()).ToFactory(func() *boundTestResource {
			return nil
		}).AsNullable()
	}).(*_PlainInjector)

	bound := injector.bounds[T[*boundTestResource]()]
	instance := bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestResource]()}, T[*boundTestResource]())

	assert.True(t, instance.IsNil())
}

func TestBoundBuildFactoryInstantiateFuncAllowsNullableNilInterface(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[boundTestGateway]()).ToFactory(func() boundTestGateway {
			return nil
		}).AsNullable()
	}).(*_PlainInjector)

	bound := injector.bounds[T[boundTestGateway]()]
	instance := bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[boundTestGateway]()}, T[boundTestGateway]())

	assert.True(t, instance.IsNil())
}

func TestBoundBuildFactoryInstantiateFuncPanicsForNilWithoutNullable(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*boundTestResource]()).ToFactory(func() *boundTestResource {
			return nil
		})
	}).(*_PlainInjector)

	bound := injector.bounds[T[*boundTestResource]()]

	assert.PanicsWithError(t,
		"factory of *di.boundTestResource returned nil",
		func() {
			_ = bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestResource]()}, T[*boundTestResource]())
		},
	)
}

func TestBoundBuildFactoryInstantiateFuncSkipsInitForBoundInstance(t *testing.T) {
	resource := &boundTestResource{}
	injector := NewInjector(func(b *Binder) {
		b.BindInstance(resource)
	}).(*_PlainInjector)

	bound := injector.bounds[T[*boundTestResource]()]
	instance := bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestResource]()}, T[*boundTestResource]())

	assert.Same(t, resource, instance.Interface().(*boundTestResource))
	assert.False(t, resource.initialized)
}

func TestBoundBuildInstantiateFuncPanicsOnFactoryError(t *testing.T) {
	injector := &_PlainInjector{
		bindings: map[reflect.Type]*Binding{},
	}
	binding := newBinding(injector, T[*boundTestDependency](), false)
	binding.ToFactory(func() (*boundTestDependency, error) {
		return nil, errors.New("boom")
	})

	bound := newBound(binding)

	assert.Panics(t, func() {
		_ = bound.BuildInstantiateFunc(&injector._BaseInjector)(_BuildStack{T[*boundTestDependency]()}, T[*boundTestDependency]())
	})
}

func TestBoundRejectsIncompatibleFactoryResultTypeDuringInitialization(t *testing.T) {
	assert.PanicsWithError(t,
		"first return type of func() string must be compatible with *di.boundTestDependency, but string found",
		func() {
			NewInjector(func(b *Binder) {
				b.Bind(T[*boundTestDependency]()).ToFactory(func() string {
					return "invalid"
				})
			})
		},
	)
}

func TestBoundRejectsIncompatibleDynamicFactoryResultAtResolution(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*boundTestDependency]()).ToFactory(func() any {
			return "invalid"
		})
	})

	assert.PanicsWithError(t,
		"factory of *di.boundTestDependency returned string, which is not assignable to *di.boundTestDependency",
		func() {
			var dependency *boundTestDependency
			injector.Resolve(&dependency)
		},
	)
}

func TestBoundDisposesConcreteFactoryResultThroughInterfaceBinding(t *testing.T) {
	disposed := false
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[_BoundFactoryDisposable]()).ToFactory(func() _BoundFactoryDisposable {
			return &_BoundFactoryDisposableImpl{disposed: &disposed}
		}).In(ExecutionScope)
	})
	execution := injector.StartExecution()

	var instance _BoundFactoryDisposable
	execution.Resolve(&instance)
	execution.CompleteExecution()

	assert.True(t, disposed)
}

func TestBoundBuildDisposeFuncHandlesDefaultAndCustomOnlyCases(t *testing.T) {
	defaultOrder := []string{}
	defaultOnly := newBound(newBinding(&_PlainInjector{}, T[*boundTestDisposable](), false))
	defaultOnly.BuildDisposeFunc(reflect.ValueOf(&boundTestDisposable{disposed: &defaultOrder}))()
	assert.Equal(t, []string{"default"}, defaultOrder)

	customOrder := []string{}
	customOnlyBinding := newBinding(&_PlainInjector{}, T[*boundTestDependency](), false)
	customOnlyBinding.WithDisposer(func(*boundTestDependency) {
		customOrder = append(customOrder, "custom")
	})
	customOnly := newBound(customOnlyBinding)
	customOnly.BuildDisposeFunc(reflect.ValueOf(&boundTestDependency{}))()
	assert.Equal(t, []string{"custom"}, customOrder)
}
