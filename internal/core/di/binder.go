package di

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type BindApplier func(b *Binder)

type Binder struct {
	injector *_PlainInjector
}

func newBinder(injector *_PlainInjector) *Binder {
	return &Binder{injector: injector}
}

func (b *Binder) Bind(targetType reflect.Type) *Binding {
	return b.bind(targetType, false)
}

func (b *Binder) bindImplicit(targetType reflect.Type) *Binding {
	return b.bind(targetType, true)
}

func (b *Binder) bind(targetType reflect.Type, isImplicit bool) *Binding {
	checkBindType(targetType)
	binding := newBinding(b.injector, targetType, isImplicit)
	b.injector.addBinding(binding)
	return binding
}

func (b *Binder) BindFactory(factory any) *Binding {
	factoryType := reflect.TypeOf(factory)
	vpre.Check(reflectutil.IsFuncType(factoryType), "factory must be function")
	vpre.Check(factoryType.NumOut() > 0, "factory must return at least one value")
	return b.Bind(factoryType.Out(0)).ToFactory(factory)
}

func (b *Binder) BindInstance(instance any) *Binding {
	return b.Bind(reflect.TypeOf(instance)).ToInstance(instance)
}

type Binding struct {
	injector   *_PlainInjector
	targetType reflect.Type
	isImplicit bool
	frozen     bool
	isNullable bool
	isAbstract bool

	explicitScope     Scope
	factory           *reflect.Value
	disposer          *reflect.Value
	initFactoryResult bool

	factoryDependenciesOverridden []reflect.Type
}

func (b *Binding) ensureMutable() {
	vpre.Check(!b.frozen, "binding of %s is already frozen", b.targetType)
}

func (b *Binding) freeze() {
	b.frozen = true
}

func newBinding(injector *_PlainInjector, targetType reflect.Type, isImplicit bool) *Binding {
	return &Binding{
		injector:   injector,
		targetType: targetType,
		isImplicit: isImplicit,
	}
}

func (b *Binding) setScope(scope Scope) {
	vpre.Check(isValidScope(scope), "invalid scope %s", scope)
	vpre.Check(b.explicitScope == noScope, "scope of %s was already set", b.targetType)
	b.explicitScope = scope
}

func (b *Binding) In(scope Scope) *Binding {
	b.ensureMutable()
	b.setScope(scope)
	return b
}

func (b *Binding) AsNullable() *Binding {
	b.ensureMutable()
	vpre.Check(isNilableType(b.targetType), "type %s cannot be nullable", b.targetType)
	b.isNullable = true
	return b
}

func (b *Binding) setFactory(factory any) {
	vpre.CheckNil(b.factory, "factory of %s was already set", b.targetType)

	factoryType := reflect.TypeOf(factory)
	vpre.Check(reflectutil.IsFuncType(factoryType), "%s must be function", factoryType)

	b.factory = new(reflect.ValueOf(factory))
	b.factoryDependenciesOverridden = nil
}

func (b *Binding) ensureFactoryMode(isAbstract bool) {
	vpre.Check(b.isAbstract == isAbstract || b.factory == nil,
		"binding mode of %s conflicts with existing configuration", b.targetType)
}

func (b *Binding) ToFactory(factory any) *Binding {
	b.ensureMutable()
	b.ensureFactoryMode(false)
	b.setFactory(factory)
	b.isAbstract = false
	b.initFactoryResult = true
	return b
}

func (b *Binding) ToAbstractFactory(factory any) *Binding {
	b.ensureMutable()
	vpre.Check(reflectutil.IsInterfaceType(b.targetType), "only interface target type can bind to abstract factory")
	b.ensureFactoryMode(true)
	b.setFactory(factory)
	b.isAbstract = true
	b.initFactoryResult = true
	return b
}

func (b *Binding) ToInstance(instance any) *Binding {
	b.ensureMutable()
	instanceType := reflect.TypeOf(instance)
	vpre.Check(b.targetType == instanceType || instanceType.Implements(b.targetType), "instance type is not compatible with %s", b.targetType)
	b.setFactory(func() any { return instance })
	b.initFactoryResult = false
	b.setScope(SingletonScope)
	return b
}

func (b *Binding) ToImplementation(implType reflect.Type) *Binding {
	b.ensureMutable()
	vpre.Check(reflectutil.IsInterfaceType(b.targetType), "only interface target type can bind to implementation")
	vpre.Check(reflectutil.IsStructPointerType(implType), "only struct pointer type can be bind as implementation")
	vpre.Check(implType.Implements(b.targetType), "the struct type %s does not implement %s interface", implType, b.targetType)

	b.ensureFactoryMode(false)
	b.setFactory(func(implInstance any) any { return implInstance })
	b.isAbstract = false
	b.initFactoryResult = false
	b.factoryDependenciesOverridden = []reflect.Type{implType}
	return b
}

func (b *Binding) setDisposer(disposer any) {
	vpre.CheckNil(b.disposer, "disposer of %s is already set", b.targetType)
	vpre.Check(!b.targetType.Implements(disposeDefinitionType),
		"type %s already implements %s, manual disposer is not allowed", b.targetType, disposeDefinitionType)

	disposerType := reflect.TypeOf(disposer)
	vpre.Check(reflectutil.IsFuncType(disposerType), "%s must be function", disposerType)
	vpre.Check(disposerType.NumIn() == 1, "%s must takes one argument", disposerType)

	instanceType := disposerType.In(0)
	vpre.Check(instanceType == b.targetType, "the only argument of %s must be %s", disposerType, instanceType)

	b.disposer = new(reflect.ValueOf(disposer))
}

func (b *Binding) WithDisposer(dtorFunc any) *Binding {
	b.ensureMutable()
	b.setDisposer(dtorFunc)
	return b
}
