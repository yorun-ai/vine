package ctr

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
)

type Execution interface {
	Execute(args []any, seedingFuncs ...di.SeedApplier)
	Results() []any
}

type _Execution struct {
	injector     di.PlainInjector
	seedingFuncs []di.SeedApplier
	filterTypes  []reflect.Type
	context      *Context

	executionInjector di.ExecutionInjector
	filterIndex       int
}

func (c *_Container) NewExecution(targetType reflect.Type, targetMethod reflect.Method) Execution {
	vpre.CheckNotNil(targetType, "target type cannot be nil")
	vpre.CheckNotEmpty(targetMethod.Name, "target method cannot be empty")
	return &_Execution{
		injector:    c.injector,
		filterTypes: c.filterTypes,
		context:     newContextWithMethod(targetType, targetMethod),
	}
}

func (e *_Execution) Execute(args []any, seedingFuncs ...di.SeedApplier) {
	e.seedingFuncs = append(seedingFuncs, func(s *di.Seeder) {
		s.SeedInstance(e.context)
	})
	e.context.SetArguments(args)
	e.execute()
}

func (e *_Execution) Results() []any {
	return e.context.Results()
}

func (e *_Execution) execute() {
	e.executionInjector = e.injector.StartExecution(e.seedingFuncs...)
	defer e.executionInjector.CompleteExecution()

	e.filterIndex = -1
	e.filterNext()
}

func (e *_Execution) filterNext() {
	e.filterIndex++
	if e.filterIndex >= len(e.filterTypes) {
		return
	}

	filterType := e.filterTypes[e.filterIndex]
	filter := e.executionInjector.Get(filterType).Interface().(Filter)
	filter.Filter(e.filterNext)
}
