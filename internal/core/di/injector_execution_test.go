package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type _ExecutionCompletedResource struct {
	ExecutionScoped
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
	disposeStarted := make(chan struct{})
	releaseDispose := make(chan struct{})
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
	select {
	case <-secondCompleted:
		t.Fatal("concurrent completion returned before cleanup finished")
	default:
	}

	close(releaseDispose)
	<-firstCompleted
	<-secondCompleted
}

func TestExecutionInjectorWaitsForActiveOperationBeforeDisposal(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
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
	select {
	case <-completed:
		t.Fatal("execution completed before active resolution finished")
	default:
	}

	close(release)
	<-resolved
	<-completed
	<-disposed
}
