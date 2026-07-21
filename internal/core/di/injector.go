package di

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

// Injector is the injectable view of the DI container.
type Injector interface {
	Get(targetType reflect.Type) reflect.Value
	Resolve(targetPtr any)
	Invoke(method any) []reflect.Value
}

type PlainInjector interface {
	Injector
	SubInjector(bindAppliers ...BindApplier) PlainInjector
	StartExecution(seedAppliers ...SeedApplier) ExecutionInjector
}

type _PlainInjector struct {
	_BaseInjector

	parent *_PlainInjector

	fallbackScope Scope
	bindAppliers  []BindApplier
	bindings      map[reflect.Type]*Binding
	bounds        map[reflect.Type]*_Bound

	singletons *vmap.MutexMap[reflect.Type, reflect.Value]
}

var injectorType = T[Injector]()

func NewInjector(bindAppliers ...BindApplier) PlainInjector {
	injector := &_PlainInjector{
		fallbackScope: TransientScope,
		bindAppliers:  bindAppliers,
	}
	injector.init()
	return injector
}

func (i *_PlainInjector) init() {
	i.initBindings()
	i.initBounds()
	i.completeImplicits()
	i.freezeBindings()
	i.checkDependencies()
	i.initSingletonStorage()
	i.initResolvers()
}

func (i *_PlainInjector) freezeBindings() {
	for _, binding := range i.bindings {
		binding.freeze()
	}
}

func (i *_PlainInjector) initBindings() {
	i.bindings = map[reflect.Type]*Binding{}
	for _, bindApplier := range i.bindAppliers {
		bindApplier(newBinder(i))
	}
}

func (i *_PlainInjector) addBinding(binding *Binding) {
	vpre.CheckNil(i.lookupBinding(binding.targetType), "type %s has already bound", binding.targetType)
	i.bindings[binding.targetType] = binding
}

func (i *_PlainInjector) lookupBinding(targetType reflect.Type) *Binding {
	if binding, exists := i.bindings[targetType]; exists {
		return binding
	}
	if i.parent != nil {
		return i.parent.lookupBinding(targetType)
	}
	return nil
}

func (i *_PlainInjector) initBounds() {
	i.bounds = map[reflect.Type]*_Bound{}
	for _, binding := range vmap.Values(i.bindings) {
		i.bounds[binding.targetType] = newBound(binding)
	}
}

func (i *_PlainInjector) visibleBounds() []*_Bound {
	bounds := vmap.Values(i.bounds)
	if i.parent != nil {
		return append(bounds, i.parent.visibleBounds()...)
	}
	return bounds
}

func (i *_PlainInjector) completeImplicits() {
	for {
		implicitTypes := []reflect.Type{}
		for _, bound := range vmap.Values(i.bounds) {
			for _, depsType := range bound.Dependencies() {
				if i.lookupBinding(depsType) == nil {
					implicitTypes = append(implicitTypes, depsType)
				}
			}
		}

		implicitTypes = vslice.Unique(implicitTypes)
		if len(implicitTypes) == 0 {
			break
		}

		addedImplicit := false
		for _, implicitType := range implicitTypes {
			if implicitType == injectorType {
				continue
			}
			vpre.Check(reflectutil.IsStructPointerType(implicitType),
				"implicit binding only supports struct pointer, but %s found", implicitType)

			binding := newBinder(i).bindImplicit(implicitType)
			i.bounds[binding.targetType] = newBound(binding)
			addedImplicit = true
		}

		if !addedImplicit {
			break
		}
	}

	for _, bound := range vmap.Values(i.bounds) {
		vpre.Check(bound.ResolveScope(i.fallbackScope) != noScope, "bound type=%s has no scope", bound.TargetType().String())
	}
}

func (i *_PlainInjector) initSingletonStorage() {
	i.singletons = vmap.NewMutexMap[reflect.Type, reflect.Value]()
}

func (i *_PlainInjector) initResolvers() {
	i.resolvers = map[reflect.Type]_Resolver{injectorType: newInjectorResolver(i)}
	i.dynamicResolvers = map[reflect.Type]_Resolver{}
	i.resolveDynamic = i.resolveDynamicResolver
	for _, bound := range i.visibleBounds() {
		i.resolvers[bound.TargetType()] = i.buildResolver(bound)
	}
}

func (i *_PlainInjector) buildResolver(bound *_Bound) _Resolver {
	scope := bound.ResolveScope(i.fallbackScope)
	if scope == ExecutionScope {
		return newIllegalExecutionResolver(bound)
	}

	if bound.binding.isAbstract {
		if scope == SingletonScope && i != bound.binding.injector {
			return newRedirectionResolver(bound)
		}
		return newAbstractResolver(bound, &i._BaseInjector, i.fallbackScope)
	}

	if scope == SingletonScope {
		if i != bound.binding.injector {
			return newRedirectionResolver(bound)
		}
		return newPersistedResolver(bound, i.singletons, &i._BaseInjector)
	}

	// scope == TransientScope
	return newTransientResolver(bound, &i._BaseInjector)
}

func (i *_PlainInjector) resolveDynamicResolver(targetType reflect.Type) _Resolver {
	bound := i.lookupAbstractBound(targetType)
	if bound == nil {
		return nil
	}
	return i.buildResolver(bound)
}

func (i *_PlainInjector) lookupAbstractBound(targetType reflect.Type) *_Bound {
	var matched *_Bound
	for _, bound := range i.visibleBounds() {
		if !bound.binding.isAbstract {
			continue
		}
		if targetType == bound.TargetType() {
			continue
		}
		if !targetType.Implements(bound.TargetType()) {
			continue
		}
		vpre.Check(matched == nil, "multiple abstract bindings matched for %s", targetType)
		matched = bound
	}
	return matched
}

// SubInjector

func (i *_PlainInjector) SubInjector(bindAppliers ...BindApplier) PlainInjector {
	subInjector := &_PlainInjector{
		parent:        i,
		fallbackScope: TransientScope,
		bindAppliers:  bindAppliers,
	}
	subInjector.init()
	return subInjector
}
