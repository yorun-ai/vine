package di

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type resolverSingletonResource struct {
	SingletonScoped
}

func (r *resolverSingletonResource) DIDispose() {
	cleanupOrder = append(cleanupOrder, "resolver-singleton")
}

type resolverExecutionResource struct {
	ExecutionScoped
}

func (r *resolverExecutionResource) DIDispose() {
	cleanupOrder = append(cleanupOrder, "resolver-execution")
}

type resolverConcurrentSingletonA struct {
	SingletonScoped
}

type resolverConcurrentSingletonB struct {
	SingletonScoped
}

type resolverConcurrentSingletonC struct {
	SingletonScoped
}

type resolverConcurrentSingletonD struct {
	SingletonScoped
}

type resolverConcurrentSingletonE struct {
	SingletonScoped
}

type resolverConcurrentSingletonF struct {
	SingletonScoped
}

type resolverConcurrentSingletonG struct {
	SingletonScoped
}

type resolverConcurrentSingletonH struct {
	SingletonScoped
}

type resolverTransientResource struct {
	TransientScoped
	marker bool
}

func (r *resolverTransientResource) DIDispose() {
	cleanupOrder = append(cleanupOrder, "resolver-transient")
}

func TestRedirectionResolverReturnsEmptyDispose(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*resolverSingletonResource]())
	}).(*_PlainInjector)

	execution := injector.StartExecution().(*_ExecutionInjector)

	resolver, ok := execution.resolvers[T[*resolverSingletonResource]()].(*_RedirectionResolver)
	assert.True(t, ok)

	instance, dispose := resolver.GetInstance(_BuildStack{T[*resolverSingletonResource]()}, T[*resolverSingletonResource]())
	assert.NotNil(t, instance.Interface())

	dispose()
	assert.Empty(t, cleanupOrder)
}

func TestPersistedResolverSeededInstanceReturnsEmptyDispose(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*resolverExecutionResource]())
	}).(*_PlainInjector)

	execution := injector.StartExecution().(*_ExecutionInjector)
	resolver, ok := execution.resolvers[T[*resolverExecutionResource]()].(*_PersistedResolver)
	assert.True(t, ok)

	seeded := &resolverExecutionResource{}
	resolver.SeedInstance(reflect.ValueOf(seeded))

	instance, dispose := resolver.GetInstance(_BuildStack{T[*resolverExecutionResource]()}, T[*resolverExecutionResource]())
	assert.Same(t, seeded, instance.Interface())

	dispose()
	assert.Empty(t, cleanupOrder)
}

func TestPersistedResolverStorageSupportsConcurrentSingletonFirstResolution(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*resolverConcurrentSingletonA]())
		b.Bind(T[*resolverConcurrentSingletonB]())
		b.Bind(T[*resolverConcurrentSingletonC]())
		b.Bind(T[*resolverConcurrentSingletonD]())
		b.Bind(T[*resolverConcurrentSingletonE]())
		b.Bind(T[*resolverConcurrentSingletonF]())
		b.Bind(T[*resolverConcurrentSingletonG]())
		b.Bind(T[*resolverConcurrentSingletonH]())
	})

	resolveFuncs := []func(){
		func() {
			var resolved *resolverConcurrentSingletonA
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonB
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonC
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonD
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonE
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonF
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonG
			injector.Resolve(&resolved)
		},
		func() {
			var resolved *resolverConcurrentSingletonH
			injector.Resolve(&resolved)
		},
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, resolve := range resolveFuncs {
		wg.Add(1)
		go func(resolve func()) {
			defer wg.Done()
			<-start
			resolve()
		}(resolve)
	}
	close(start)
	wg.Wait()
}

func TestTransientResolverReturnsDisposeForEachInstance(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*resolverTransientResource]())
	}).(*_PlainInjector)

	resolver, ok := injector.resolvers[T[*resolverTransientResource]()].(*_TransientResolver)
	assert.True(t, ok)

	first, firstDispose := resolver.GetInstance(_BuildStack{T[*resolverTransientResource]()}, T[*resolverTransientResource]())
	second, secondDispose := resolver.GetInstance(_BuildStack{T[*resolverTransientResource]()}, T[*resolverTransientResource]())

	firstInstance := first.Interface().(*resolverTransientResource)
	secondInstance := second.Interface().(*resolverTransientResource)
	assert.NotSame(t, firstInstance, secondInstance)

	firstDispose()
	secondDispose()

	assert.Equal(t, []string{"resolver-transient", "resolver-transient"}, cleanupOrder)
}

func TestIllegalExecutionResolverPanics(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*resolverExecutionResource]())
	}).(*_PlainInjector)

	resolver, ok := injector.resolvers[T[*resolverExecutionResource]()].(*_IllegalExecutionResolver)
	assert.True(t, ok)

	assert.Panics(t, func() {
		_, _ = resolver.GetInstance(_BuildStack{T[*resolverExecutionResource]()}, T[*resolverExecutionResource]())
	})
}
