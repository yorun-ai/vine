package di

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vmath"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type _Bound struct {
	binding *Binding

	declaredScope         Scope
	hasDeclaredInitMethod bool

	injectedFields      []*_InjectedField
	factoryDependencies []reflect.Type
	factoryMayError     bool
	mayDispose          bool
	disposeMayError     bool
}

func newBound(binding *Binding) *_Bound {
	b := &_Bound{binding: binding}
	b.init()
	b.analyzeFactory()
	b.analyzeDisposer()
	return b
}

func (b *_Bound) init() {
	if !reflectutil.IsStructPointerType(b.binding.targetType) {
		return
	}

	b.declaredScope = scanDeclaredScope(b.binding.targetType)
	b.hasDeclaredInitMethod = b.binding.targetType.Implements(initDefinitionType)

	b.injectedFields = scanInjectedFields(b.binding.targetType)
}

func (b *_Bound) analyzeFactory() {
	if b.binding.factory == nil {
		return
	}

	factoryType := b.binding.factory.Type()
	vpre.Check(vmath.InRange(factoryType.NumOut(), 1, 3), "%s must return (value) or (value, error)", factoryType)
	resultType := factoryType.Out(0)
	targetType := b.binding.targetType
	vpre.Check(resultType.AssignableTo(targetType) || targetType.AssignableTo(resultType),
		"first return type of %s must be compatible with %s, but %s found", factoryType, targetType, resultType)
	if factoryType.NumOut() == 2 {
		b.factoryMayError = true
		errType := factoryType.Out(1)
		vpre.Check(reflectutil.IsErrorType(errType), "second return type of %s must be error, but %s found", factoryType, errType)
	}

	if b.binding.implementationType != nil {
		b.factoryDependencies = []reflect.Type{b.binding.implementationType}
		return
	}

	for in := range factoryType.Ins() {
		b.factoryDependencies = append(b.factoryDependencies, in)
	}
}

func (b *_Bound) analyzeDisposer() {
	if b.binding.disposer == nil {
		// A factory result can implement DIDispose dynamically. ToImplementation
		// borrows its result, so the concrete binding owns that cleanup.
		b.mayDispose = b.binding.implementationType == nil &&
			(b.binding.factory != nil || b.TargetType().Implements(disposeDefinitionType))
		return
	}

	b.mayDispose = true
	disposerType := b.binding.disposer.Type()
	vpre.Check(disposerType.NumOut() < 2, "%s must return nothing or (error)", disposerType)
	if disposerType.NumOut() == 1 {
		errType := disposerType.Out(0)
		vpre.Check(reflectutil.IsErrorType(errType), "return type of %s must be error, but %s found", disposerType, errType)
		b.disposeMayError = true
	}
}

func (b *_Bound) TargetType() reflect.Type {
	return b.binding.targetType
}

func (b *_Bound) isImplicit() bool {
	return b.binding.isImplicit
}

func (b *_Bound) DeclaredScope() Scope {
	if b.binding.explicitScope != noScope {
		return b.binding.explicitScope
	}
	if b.declaredScope != noScope {
		return b.declaredScope
	}
	return noScope
}

func (b *_Bound) ResolveScope(fallbackScope Scope) Scope {
	scope := b.DeclaredScope()
	if scope != noScope {
		return scope
	}
	return fallbackScope
}

func (b *_Bound) Dependencies() []reflect.Type {
	var dependencies []reflect.Type
	if b.binding.factory == nil {
		dependencies = vslice.Map(b.injectedFields, func(f *_InjectedField) reflect.Type {
			return f.kind
		})
	}
	dependencies = append(dependencies, b.factoryDependencies...)
	dependencies = append(dependencies, b.binding.extraDependencies...)
	if b.binding.isAbstract {
		dependencies = vslice.Filter(dependencies, func(dep reflect.Type) bool {
			return dep != T[ResolveContext]()
		})
	}
	return vslice.Unique(dependencies)
}

func (b *_Bound) String() string {
	return fmt.Sprintf("tarType=%s, isImplicit=%t, declared=%s (def=%s, in=%s)",
		b.TargetType(), b.isImplicit(), b.DeclaredScope(), b.declaredScope, b.binding.explicitScope)
}

func (b *_Bound) BuildInstantiateFunc(injector *_BaseInjector) _InstantiateFunc {
	construct := b.buildInstantiateFunc(injector)
	track := injector.trackDispose
	if !b.mayDispose {
		track = nil
	}
	onResolved := b.binding.onDependenciesResolved
	if onResolved == nil && track == nil {
		return construct
	}
	return func(stack _BuildStack, requestedType reflect.Type) reflect.Value {
		instance := construct(stack, requestedType)
		if onResolved != nil {
			dependencies := make([]reflect.Value, len(b.binding.extraDependencies))
			for i, kind := range b.binding.extraDependencies {
				dependencies[i] = injector.get(kind, stack.push(kind))
			}
			onResolved(instance, dependencies)
		}
		// All dependencies have been constructed and registered before their
		// dependent. Cached reads never re-enter this constructor.
		if track != nil {
			if dispose := b.BuildDisposeFunc(instance); dispose != nil {
				track(dispose)
			}
		}
		return instance
	}
}

func (b *_Bound) buildInstantiateFunc(injector *_BaseInjector) _InstantiateFunc {
	if b.binding.factory == nil {
		return b.buildDefaultInstantiateFunc(injector)
	}

	return func(stack _BuildStack, requestedType reflect.Type) reflect.Value {
		arguments := vslice.Map(b.factoryDependencies, func(k reflect.Type) reflect.Value {
			if b.binding.isAbstract && k == T[ResolveContext]() {
				return reflect.ValueOf(ResolveContext{TargetType: requestedType})
			}
			return injector.get(k, stack.push(k))
		})

		results := b.binding.factory.Call(arguments)
		if b.factoryMayError && !results[1].IsNil() {
			vpre.Panicf("error occurs when construct %s, build stack=%s, error=%s", b.binding.targetType.Name(), stack, results[1].Interface())
			return reflect.Value{}
		}

		instance := b.normalizeFactoryInstance(results[0], requestedType)
		vpre.Check(instance.IsValid() && instance.Type().AssignableTo(requestedType),
			"factory of %s returned %s, which is not assignable to %s",
			b.binding.targetType, instance.Type(), requestedType)
		if b.binding.initFactoryResult {
			initInstance(instance)
		}
		return instance
	}
}

func (b *_Bound) normalizeFactoryInstance(instance reflect.Value, requestedType reflect.Type) reflect.Value {
	if reflectutil.IsNilValue(instance) {
		vpre.Check(b.binding.isNullable, "factory of %s returned nil", b.binding.targetType)
		return reflect.Zero(requestedType)
	}
	if instance.Kind() == reflect.Interface {
		instance = instance.Elem()
	}
	return instance
}

func (b *_Bound) buildDefaultInstantiateFunc(injector *_BaseInjector) _InstantiateFunc {
	if reflectutil.IsInterface(b.binding.targetType) {
		return func(stack _BuildStack, _ reflect.Type) reflect.Value {
			vpre.Panicf("factory of %s not found, build stack=%s", b.binding.targetType, stack)
			return reflect.Value{}
		}
	}

	return func(stack _BuildStack, _ reflect.Type) reflect.Value {
		instance := reflect.New(b.binding.targetType.Elem())

		for _, injectedField := range b.injectedFields {
			fieldValue := injector.get(injectedField.kind, stack.push(injectedField.kind))
			instance.Elem().FieldByIndex(injectedField.index).Set(fieldValue)
		}

		initInstance(instance)
		return instance
	}
}

func initInstance(instance reflect.Value) {
	if reflectutil.IsNilValue(instance) {
		return
	}

	if initializer, ok := reflect.TypeAssert[InitDefinition](instance); ok {
		initializer.DIInit()
	}
}

func (b *_Bound) BuildDisposeFunc(instance reflect.Value) _DisposeFunc {
	if reflectutil.IsNilValue(instance) {
		return nil
	}

	if b.binding.disposer != nil {
		return func() {
			rets := b.binding.disposer.Call([]reflect.Value{instance})
			if b.disposeMayError && !rets[0].IsNil() {
				vpre.Panicf("error occurs when dispose %s, error=%s", b.binding.targetType.Name(), rets[0].Interface())
			}
		}
	}

	if disposable, ok := reflect.TypeAssert[DisposeDefinition](instance); ok {
		return disposable.DIDispose
	}
	return nil
}
