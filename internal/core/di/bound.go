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

	if b.binding.factoryDependenciesOverridden != nil {
		b.factoryDependencies = append([]reflect.Type{}, b.binding.factoryDependenciesOverridden...)
		return
	}

	for index := 0; index < factoryType.NumIn(); index++ {
		b.factoryDependencies = append(b.factoryDependencies, factoryType.In(index))
	}
}

func (b *_Bound) analyzeDisposer() {
	if b.binding.disposer == nil {
		return
	}

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

	if initializer, ok := instance.Interface().(InitDefinition); ok {
		initializer.DIInit()
	}
}

func (b *_Bound) BuildDisposeFunc(instance reflect.Value) _DisposeFunc {
	if reflectutil.IsNilValue(instance) {
		return emptyDispose
	}

	_, hasDisposeMethod := instance.Interface().(DisposeDefinition)
	if b.binding.disposer == nil && !hasDisposeMethod {
		return emptyDispose
	}

	if b.binding.disposer != nil {
		return func() {
			rets := b.binding.disposer.Call([]reflect.Value{instance})
			if b.disposeMayError && !rets[0].IsNil() {
				vpre.Panicf("error occurs when dispose %s, error=%s", b.binding.targetType.Name(), rets[0].Interface())
			}
		}
	}

	return func() {
		instance.Interface().(DisposeDefinition).DIDispose()
	}
}
