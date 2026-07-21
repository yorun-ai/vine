package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type ExecutionInjector interface {
	Injector
	CompleteExecution()
}

type _ExecutionState int

const (
	executionStateSeeding _ExecutionState = iota
	executionStateActive
	executionStateCompleting
	executionStateCompleted
)

type _ExecutionInjector struct {
	_BaseInjector

	// executor is the plain injector that starts this execution and owns the bound graph.
	executor *_PlainInjector

	fallbackScope Scope

	storage *vmap.MutexMap[reflect.Type, reflect.Value]

	stateMutex     sync.Mutex
	state          _ExecutionState
	completionDone chan struct{}
	operations     sync.WaitGroup

	seedMutex sync.Mutex

	disposeMutex sync.Mutex
	disposeFuncs []_DisposeFunc
}

func (i *_PlainInjector) StartExecution(seedAppliers ...SeedApplier) ExecutionInjector {
	execution := &_ExecutionInjector{
		executor:       i,
		fallbackScope:  ExecutionScope,
		storage:        vmap.NewMutexMap[reflect.Type, reflect.Value](),
		state:          executionStateSeeding,
		completionDone: make(chan struct{}),
		disposeFuncs:   []_DisposeFunc{},
	}
	execution.trackDispose = execution.trackDisposable
	execution.initResolvers()
	for _, seedApplier := range seedAppliers {
		seedApplier(newSeeder(execution))
	}
	execution.finishSeeding()
	return execution
}

func (e *_ExecutionInjector) initResolvers() {
	e.resolvers = map[reflect.Type]_Resolver{injectorType: newInjectorResolver(e)}
	e.dynamicResolvers = map[reflect.Type]_Resolver{}
	e.resolveDynamic = e.resolveDynamicResolver
	for _, bound := range e.executor.visibleBounds() {
		e.resolvers[bound.TargetType()] = e.buildResolver(bound)
	}
}

func (e *_ExecutionInjector) buildResolver(bound *_Bound) _Resolver {
	if bound.binding.isAbstract {
		if bound.ResolveScope(e.fallbackScope) == SingletonScope {
			return newRedirectionResolver(bound)
		}
		return newAbstractResolver(bound, &e._BaseInjector, e.fallbackScope)
	}
	switch bound.ResolveScope(e.fallbackScope) {
	case SingletonScope:
		return newRedirectionResolver(bound)
	case ExecutionScope:
		return newPersistedResolver(bound, e.storage, &e._BaseInjector)
	case TransientScope:
		return newTransientResolver(bound, &e._BaseInjector)
	}

	vpre.MustNotReach()
	return nil
}

func (e *_ExecutionInjector) resolveDynamicResolver(targetType reflect.Type) _Resolver {
	bound := e.executor.lookupAbstractBound(targetType)
	if bound == nil {
		return nil
	}
	return e.buildResolver(bound)
}

func (e *_ExecutionInjector) CompleteExecution() {
	e.stateMutex.Lock()
	switch e.state {
	case executionStateCompleting:
		done := e.completionDone
		e.stateMutex.Unlock()
		<-done
		return
	case executionStateCompleted:
		e.stateMutex.Unlock()
		return
	case executionStateSeeding:
		e.stateMutex.Unlock()
		vpre.Panicf("execution injector is still seeding")
	}
	e.state = executionStateCompleting
	e.stateMutex.Unlock()

	e.operations.Wait()
	defer e.finishCompletion()
	e.disposeExecutionInstances()
}

func (e *_ExecutionInjector) finishCompletion() {
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()
	e.storage = nil
	e.state = executionStateCompleted
	close(e.completionDone)
}

func (e *_ExecutionInjector) trackDisposable(dispose _DisposeFunc) {
	e.disposeMutex.Lock()
	defer e.disposeMutex.Unlock()
	e.disposeFuncs = append(e.disposeFuncs, dispose)
}

func (e *_ExecutionInjector) disposeExecutionInstances() {
	e.disposeMutex.Lock()
	disposeFuncs := e.disposeFuncs
	e.disposeFuncs = nil
	e.disposeMutex.Unlock()

	var recoveredValues []any
	for index := len(disposeFuncs) - 1; index >= 0; index-- {
		if recovered := runDispose(disposeFuncs[index]); recovered != nil {
			recoveredValues = append(recoveredValues, recovered)
		}
	}
	if len(recoveredValues) == 1 {
		panic(recoveredValues[0])
	}
	if len(recoveredValues) > 1 {
		disposeErrors := make([]error, 0, len(recoveredValues))
		for _, recovered := range recoveredValues {
			if err, ok := recovered.(error); ok {
				disposeErrors = append(disposeErrors, err)
				continue
			}
			disposeErrors = append(disposeErrors, fmt.Errorf("%v", recovered))
		}
		panic(errors.Join(disposeErrors...))
	}
}

func (e *_ExecutionInjector) Get(targetType reflect.Type) reflect.Value {
	done := e.beginOperation()
	defer done()
	return e._BaseInjector.Get(targetType)
}

func (e *_ExecutionInjector) Resolve(targetPtr any) {
	done := e.beginOperation()
	defer done()
	e._BaseInjector.Resolve(targetPtr)
}

func (e *_ExecutionInjector) Invoke(method any) []reflect.Value {
	done := e.beginOperation()
	defer done()
	return e._BaseInjector.Invoke(method)
}

func (e *_ExecutionInjector) beginOperation() func() {
	e.stateMutex.Lock()
	if e.state != executionStateActive {
		e.stateMutex.Unlock()
		vpre.Panicf("execution injector is already completed")
	}
	e.operations.Add(1)
	e.stateMutex.Unlock()
	return e.operations.Done
}

func (e *_ExecutionInjector) beginSeed() func() {
	e.seedMutex.Lock()
	e.stateMutex.Lock()
	if e.state != executionStateSeeding {
		e.stateMutex.Unlock()
		e.seedMutex.Unlock()
		vpre.Panicf("execution injector is no longer accepting seeds")
	}
	e.stateMutex.Unlock()
	return e.seedMutex.Unlock
}

func (e *_ExecutionInjector) finishSeeding() {
	e.seedMutex.Lock()
	defer e.seedMutex.Unlock()
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()
	e.state = executionStateActive
}

func runDispose(dispose _DisposeFunc) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	dispose()
	return nil
}
