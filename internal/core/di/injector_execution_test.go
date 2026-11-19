package di

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
)

type _ExecutionCompletedResource struct {
	ExecutionScoped
}

func TestExecutionCreatesResolversOnDemand(t *testing.T) {
	parent := NewInjector(func(b *Binder) {
		b.Bind(T[*RequestMetrics]())
		b.Bind(T[*SeededRequestContext]())
	})
	execution := parent.SubInjector().StartExecution(func(s *Seeder) {
		s.SeedInstance(new(SeededRequestContext))
	}).(*_ExecutionInjector)
	defer execution.CompleteExecution()

	assert.NotContains(t, execution.resolvers, T[*RequestMetrics]())
	assert.NotContains(t, execution.dynamicResolvers, T[*RequestMetrics]())
	assert.Contains(t, execution.dynamicResolvers, T[*SeededRequestContext]())
	execution.Get(T[*RequestMetrics]())
	assert.Contains(t, execution.dynamicResolvers, T[*RequestMetrics]())
}

func TestExecutionHonorsExplicitInjectorBinding(t *testing.T) {
	custom := NewInjector()
	parent := NewInjector(func(b *Binder) {
		b.Bind(injectorType).ToInstance(custom)
	})
	execution := parent.SubInjector().StartExecution()
	defer execution.CompleteExecution()

	var resolved Injector
	execution.Resolve(&resolved)
	assert.Same(t, custom, resolved)
}

func TestExecutionConcurrentFirstResolutionCreatesOneInstancePerExecution(t *testing.T) {
	var created atomic.Int32
	var disposed atomic.Int32
	injector := NewInjector(func(b *Binder) {
		b.BindFactory(func() *abstractConfigA {
			created.Add(1)
			return new(abstractConfigA)
		}).In(ExecutionScope).WithDisposer(func(*abstractConfigA) {
			disposed.Add(1)
		})
	})
	first := injector.StartExecution()
	second := injector.StartExecution()
	instances := make([]*abstractConfigA, 32)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range instances {
		wg.Go(func() {
			<-start
			first.Resolve(&instances[i])
		})
	}
	close(start)
	wg.Wait()
	for _, instance := range instances {
		assert.Same(t, instances[0], instance)
	}
	var other *abstractConfigA
	second.Resolve(&other)
	assert.NotSame(t, instances[0], other)
	assert.EqualValues(t, 2, created.Load())
	first.CompleteExecution()
	assert.EqualValues(t, 1, disposed.Load())
	second.CompleteExecution()
	assert.EqualValues(t, 2, disposed.Load())
}

func BenchmarkExecutionResolveOne(b *testing.B) {
	for _, count := range []int{1, 32, 128} {
		b.Run(fmt.Sprintf("%d_bindings", count), func(b *testing.B) {
			var target reflect.Type
			injector := NewInjector(func(binder *Binder) {
				for i := range count {
					field := reflect.StructField{Name: fmt.Sprintf("Field%d", i), Type: T[int]()}
					target = reflect.PointerTo(reflect.StructOf([]reflect.StructField{field}))
					binder.Bind(target).In(ExecutionScope)
				}
			})
			b.ReportAllocs()
			for b.Loop() {
				execution := injector.StartExecution()
				execution.Get(target)
				execution.CompleteExecution()
			}
		})
	}
}

func TestExecutionInjectorContinuesDisposalAfterPanic(t *testing.T) {
	disposeOrder := []string{}
	execution := NewInjector().StartExecution().(*_ExecutionInjector)
	execution.disposeFuncs = []_DisposeFunc{
		func() {
			disposeOrder = append(disposeOrder, "first")
		},
		func() {
			disposeOrder = append(disposeOrder, "second")
			panic("dispose failed")
		},
		func() {
			disposeOrder = append(disposeOrder, "third")
		},
	}

	assert.PanicsWithValue(t, "dispose failed", execution.CompleteExecution)
	assert.Equal(t, []string{"third", "second", "first"}, disposeOrder)
}

func TestExecutionInjectorCompleteIsIdempotent(t *testing.T) {
	execution := NewInjector().StartExecution()

	assert.NotPanics(t, execution.CompleteExecution)
	assert.NotPanics(t, execution.CompleteExecution)
}

func TestExecutionInjectorRejectsOperationsAfterCompletion(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*_ExecutionCompletedResource]())
	})
	execution := injector.StartExecution()
	execution.CompleteExecution()

	assert.PanicsWithError(t, "execution injector is already completed", func() {
		execution.Get(T[*_ExecutionCompletedResource]())
	})
	assert.PanicsWithError(t, "execution injector is already completed", func() {
		var resource *_ExecutionCompletedResource
		execution.Resolve(&resource)
	})
	assert.PanicsWithError(t, "execution injector is already completed", func() {
		execution.Invoke(func(*_ExecutionCompletedResource) {})
	})
}

func TestExecutionInjectorRejectsSeedAfterCompletion(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*_ExecutionCompletedResource]())
	})
	var seeder *Seeder
	execution := injector.StartExecution(func(current *Seeder) {
		seeder = current
	})
	execution.CompleteExecution()

	assert.PanicsWithError(t, "execution injector is no longer accepting seeds", func() {
		seeder.Seed(T[*_ExecutionCompletedResource](), &_ExecutionCompletedResource{})
	})
}

func TestExecutionInjectorConcurrentCompleteWaitsForCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		disposeStarted := make(chan struct{})
		releaseDispose := make(chan struct{})
		releaseBlocked := sync.OnceFunc(func() { close(releaseDispose) })
		defer releaseBlocked()
		injector := NewInjector(func(b *Binder) {
			b.Bind(T[*_ExecutionCompletedResource]()).WithDisposer(func(*_ExecutionCompletedResource) {
				close(disposeStarted)
				<-releaseDispose
			})
		})
		execution := injector.StartExecution()
		execution.Get(T[*_ExecutionCompletedResource]())

		firstCompleted := make(chan struct{})
		go func() {
			defer close(firstCompleted)
			execution.CompleteExecution()
		}()
		<-disposeStarted

		secondCompleted := make(chan struct{})
		go func() {
			defer close(secondCompleted)
			execution.CompleteExecution()
		}()
		synctest.Wait()
		select {
		case <-secondCompleted:
			t.Fatal("concurrent completion returned before cleanup finished")
		default:
		}

		releaseBlocked()
		<-firstCompleted
		<-secondCompleted
	})
}

func TestExecutionInjectorWaitsForActiveOperationBeforeDisposal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		releaseBlocked := sync.OnceFunc(func() { close(release) })
		defer releaseBlocked()
		disposed := make(chan struct{})
		injector := NewInjector(func(b *Binder) {
			b.BindFactory(func() *_ExecutionCompletedResource {
				close(started)
				<-release
				return &_ExecutionCompletedResource{}
			}).In(ExecutionScope).WithDisposer(func(*_ExecutionCompletedResource) {
				close(disposed)
			})
		})
		execution := injector.StartExecution()
		resolved := make(chan struct{})
		go func() {
			defer close(resolved)
			execution.Get(reflect.TypeFor[*_ExecutionCompletedResource]())
		}()
		<-started

		completed := make(chan struct{})
		go func() {
			defer close(completed)
			execution.CompleteExecution()
		}()
		synctest.Wait()
		select {
		case <-completed:
			t.Fatal("execution completed before active resolution finished")
		default:
		}

		releaseBlocked()
		<-resolved
		<-completed
		<-disposed
	})
}
