package di

import (
	"reflect"
	"sync"

	"go.yorun.ai/vine/internal/util/goutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type _DisposeFunc func()
type _InstantiateFunc func(_BuildStack, reflect.Type) reflect.Value

var emptyDispose _DisposeFunc = func() {}

type _Resolver interface {
	GetInstance(stack _BuildStack, requestedType reflect.Type) (reflect.Value, _DisposeFunc)
}

type _InjectorResolver struct {
	injector Injector
}

func newInjectorResolver(injector Injector) *_InjectorResolver {
	return &_InjectorResolver{injector: injector}
}

func (r *_InjectorResolver) GetInstance(_BuildStack, reflect.Type) (reflect.Value, _DisposeFunc) {
	return reflect.ValueOf(r.injector), emptyDispose
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

func (r *_RedirectionResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) (reflect.Value, _DisposeFunc) {
	sourceType := r.bound.TargetType()
	if r.bound.binding.isAbstract {
		sourceType = requestedType
	}
	instance, _ := r.sourceInjector.getWithDispose(sourceType, stack)
	if requestedType != r.bound.TargetType() {
		vpre.Check(instance.Type().AssignableTo(requestedType), "redirected instance %s is not assignable to %s", instance.Type(), requestedType)
	}
	return instance, emptyDispose
}

// _PersistedResolver resolves instance from scoped persist storage.
type _PersistedResolver struct {
	bound   *_Bound
	storage *vmap.MutexMap[reflect.Type, reflect.Value]

	isSeeded bool

	instantiateOnce goutil.RecoverableOnce
	instantiateFunc _InstantiateFunc

	disposeOnce sync.Once
	disposeFunc _DisposeFunc
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

func (r *_PersistedResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) (reflect.Value, _DisposeFunc) {
	r.instantiateOnce.Do(func() {
		instance := r.instantiateFunc(stack, requestedType)
		r.setInstance(instance)
		if r.isSeeded {
			r.disposeFunc = emptyDispose
			return
		}
		r.disposeFunc = r.bound.BuildDisposeFunc(instance)
	})
	return r.getInstance(), r.dispose
}

func (r *_PersistedResolver) SeedInstance(instance reflect.Value) {
	vpre.Check(!r.hasInstance(), "type %s was already instantiated, cannot be seeded", r.bound.TargetType())
	r.isSeeded = true
	r.instantiateFunc = func(_BuildStack, reflect.Type) reflect.Value {
		return instance
	}
}

func (r *_PersistedResolver) dispose() {
	r.disposeOnce.Do(func() {
		r.disposeFunc()
	})
}

// _IllegalExecutionResolver panics when resolving instance without execution state.
type _IllegalExecutionResolver struct {
	bound *_Bound
}

func newIllegalExecutionResolver(bound *_Bound) *_IllegalExecutionResolver {
	return &_IllegalExecutionResolver{bound: bound}
}

func (r *_IllegalExecutionResolver) GetInstance(stack _BuildStack, _ reflect.Type) (reflect.Value, _DisposeFunc) {
	vpre.Panicf("get execution scoped type %s under non-execution mode, build stack=%s", r.bound.TargetType(), stack)
	return reflect.Value{}, emptyDispose
}

// _TransientResolver resolves a new instance on each request.
type _TransientResolver struct {
	bound           *_Bound
	instantiateFunc _InstantiateFunc
}

func newTransientResolver(bound *_Bound, injector *_BaseInjector) *_TransientResolver {
	return &_TransientResolver{
		bound:           bound,
		instantiateFunc: bound.BuildInstantiateFunc(injector),
	}
}

func (r *_TransientResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) (reflect.Value, _DisposeFunc) {
	instance := r.instantiateFunc(stack, requestedType)
	return instance, r.bound.BuildDisposeFunc(instance)
}

type _abstractPersistedEntry struct {
	once        goutil.RecoverableOnce
	disposeOnce sync.Once
	instance    reflect.Value
	disposeFunc _DisposeFunc
}

type _AbstractResolver struct {
	bound   *_Bound
	storage *vmap.MutexMap[reflect.Type, *_abstractPersistedEntry]

	fallbackScope   Scope
	instantiateFunc _InstantiateFunc
}

func newAbstractResolver(bound *_Bound, injector *_BaseInjector, fallbackScope Scope) *_AbstractResolver {
	return &_AbstractResolver{
		bound:           bound,
		storage:         vmap.NewMutexMap[reflect.Type, *_abstractPersistedEntry](),
		fallbackScope:   fallbackScope,
		instantiateFunc: bound.BuildInstantiateFunc(injector),
	}
}

func (r *_AbstractResolver) GetInstance(stack _BuildStack, requestedType reflect.Type) (reflect.Value, _DisposeFunc) {
	vpre.Check(requestedType != r.bound.TargetType(),
		"abstract binding %s cannot resolve direct interface requests", r.bound.TargetType())

	switch r.bound.ResolveScope(r.fallbackScope) {
	case SingletonScope, ExecutionScope:
		entry := r.getEntry(requestedType)
		entry.once.Do(func() {
			entry.instance = r.instantiateFunc(stack, requestedType)
			entry.disposeFunc = r.bound.BuildDisposeFunc(entry.instance)
		})
		return entry.instance, entry.dispose
	case TransientScope:
		instance := r.instantiateFunc(stack, requestedType)
		return instance, r.bound.BuildDisposeFunc(instance)
	}

	vpre.MustNotReach()
	return reflect.Value{}, emptyDispose
}

func (r *_AbstractResolver) getEntry(requestedType reflect.Type) *_abstractPersistedEntry {
	entry, _ := r.storage.LoadOrStore(requestedType, &_abstractPersistedEntry{})
	return entry
}

func (e *_abstractPersistedEntry) dispose() {
	e.disposeOnce.Do(func() {
		if e.disposeFunc != nil {
			e.disposeFunc()
		}
	})
}
