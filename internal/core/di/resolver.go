package di

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/goutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type _DisposeFunc func()
type _InstantiateFunc func(_BuildStack, reflect.Type) reflect.Value

type _Resolver interface {
	GetInstance(stack _BuildStack, requestedType reflect.Type) reflect.Value
}

type _InjectorResolver struct {
	injector Injector
}

func newInjectorResolver(injector Injector) *_InjectorResolver {
	return &_InjectorResolver{injector: injector}
}

func (r *_InjectorResolver) GetInstance(_BuildStack, reflect.Type) reflect.Value {
	return reflect.ValueOf(r.injector)
}

// _RedirectionResolver resolves instance from source injector.
type _RedirectionResolver struct {
	bound          *_Bound
	sourceInjector *_PlainInjector
}

func newRedirectionResolver(bound *_Bound) *_RedirectionResolver {
	return &_RedirectionResolver{
		bound:          bound,
		sourceInjector: bound.binding.injector,
	}
}

func (r *_RedirectionResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) reflect.Value {
	sourceType := r.bound.TargetType()
	if r.bound.binding.isAbstract {
		sourceType = requestedType
	}
	instance := r.sourceInjector.get(sourceType, stack)
	if requestedType != r.bound.TargetType() {
		vpre.Check(instance.Type().AssignableTo(requestedType), "redirected instance %s is not assignable to %s", instance.Type(), requestedType)
	}
	return instance
}

// _PersistedResolver resolves instance from scoped persist storage.
type _PersistedResolver struct {
	bound   *_Bound
	storage *vmap.MutexMap[reflect.Type, reflect.Value]

	instantiateOnce goutil.RecoverableOnce
	instantiateFunc _InstantiateFunc
}

func newPersistedResolver(bound *_Bound, storage *vmap.MutexMap[reflect.Type, reflect.Value], injector *_BaseInjector) *_PersistedResolver {
	return &_PersistedResolver{
		bound:           bound,
		storage:         storage,
		instantiateFunc: bound.BuildInstantiateFunc(injector),
	}
}

func (r *_PersistedResolver) setInstance(instance reflect.Value) {
	r.storage.Store(r.bound.TargetType(), instance)
}

func (r *_PersistedResolver) getInstance() reflect.Value {
	instance, _ := r.storage.Load(r.bound.TargetType())
	return instance
}

func (r *_PersistedResolver) hasInstance() bool {
	_, ok := r.storage.Load(r.bound.TargetType())
	return ok
}

func (r *_PersistedResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) reflect.Value {
	r.instantiateOnce.Do(func() {
		instance := r.instantiateFunc(stack, requestedType)
		r.setInstance(instance)
	})
	return r.getInstance()
}

func (r *_PersistedResolver) SeedInstance(instance reflect.Value) {
	vpre.Check(!r.hasInstance(), "type %s was already instantiated, cannot be seeded", r.bound.TargetType())
	r.instantiateFunc = func(_BuildStack, reflect.Type) reflect.Value {
		return instance
	}
}

// _IllegalExecutionResolver panics when resolving instance without execution state.
type _IllegalExecutionResolver struct {
	bound *_Bound
}

func newIllegalExecutionResolver(bound *_Bound) *_IllegalExecutionResolver {
	return &_IllegalExecutionResolver{bound: bound}
}

func (r *_IllegalExecutionResolver) GetInstance(stack _BuildStack, _ reflect.Type) reflect.Value {
	vpre.Panicf("get execution scoped type %s under non-execution mode, build stack=%s", r.bound.TargetType(), stack)
	return reflect.Value{}
}

// _TransientResolver resolves a new instance on each request.
type _TransientResolver struct {
	instantiateFunc _InstantiateFunc
}

func newTransientResolver(bound *_Bound, injector *_BaseInjector) *_TransientResolver {
	return &_TransientResolver{
		instantiateFunc: bound.BuildInstantiateFunc(injector),
	}
}

func (r *_TransientResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) reflect.Value {
	return r.instantiateFunc(stack, requestedType)
}

type _AbstractPersistedEntry struct {
	once     goutil.RecoverableOnce
	instance reflect.Value
}

type _AbstractResolver struct {
	bound   *_Bound
	storage *vmap.MutexMap[reflect.Type, *_AbstractPersistedEntry]

	fallbackScope   Scope
	instantiateFunc _InstantiateFunc
}

func newAbstractResolver(bound *_Bound, injector *_BaseInjector, fallbackScope Scope) *_AbstractResolver {
	return &_AbstractResolver{
		bound:           bound,
		storage:         vmap.NewMutexMap[reflect.Type, *_AbstractPersistedEntry](),
		fallbackScope:   fallbackScope,
		instantiateFunc: bound.BuildInstantiateFunc(injector),
	}
}

func (r *_AbstractResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) reflect.Value {
	vpre.Check(requestedType != r.bound.TargetType(),
		"abstract binding %s cannot resolve direct interface requests", r.bound.TargetType())

	switch r.bound.ResolveScope(r.fallbackScope) {
	case SingletonScope, ExecutionScope:
		entry := r.getEntry(requestedType)
		entry.once.Do(func() {
			entry.instance = r.instantiateFunc(stack, requestedType)
		})
		return entry.instance
	case TransientScope:
		return r.instantiateFunc(stack, requestedType)
	}

	vpre.MustNotReach()
	return reflect.Value{}
}

func (r *_AbstractResolver) getEntry(requestedType reflect.Type) *_AbstractPersistedEntry {
	entry, _ := r.storage.LoadOrStore(requestedType, &_AbstractPersistedEntry{})
	return entry
}
