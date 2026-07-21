package task

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type Executor interface {
	Init(infoDict spec.ImplDict)
	Execute(taskContext spec.Context, triggerImpl spec.TriggerImpl, arguments []any) ex.Error
}

type _ContainerExecutor struct {
	filterTypes  []reflect.Type
	bindAppliers []di.BindApplier
	container    ctr.Container
}

func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return &_ContainerExecutor{
		filterTypes:  filterTypes,
		bindAppliers: bindAppliers,
	}
}

func (e *_ContainerExecutor) Init(implDict spec.ImplDict) {
	var bindAppliers []di.BindApplier
	implDict.IterateTaskImpl(func(taskImpl spec.TaskImpl) {
		taskInfo := taskImpl.Info()
		bindAppliers = append(bindAppliers, func(b *di.Binder) {
			b.Bind(taskImpl.Type()).In(di.ExecutionScope)
			if taskImpl.IsERType() {
				b.Bind(taskInfo.ERRunnerType()).ToImplementation(taskImpl.Type()).In(di.ExecutionScope)
				return
			}
			b.Bind(taskInfo.RunnerType()).ToImplementation(taskImpl.Type()).In(di.ExecutionScope)
			b.Bind(taskInfo.ERRunnerType()).ToFactory(taskInfo.WrapperERRunnerCtor()).In(di.ExecutionScope)
		})
	})

	bindAppliers = append(bindAppliers, func(b *di.Binder) {
		b.Bind(di.T[spec.Context]()).In(di.ExecutionScope)
		b.Bind(di.T[spec.TriggerInfo]()).In(di.ExecutionScope)
	})
	bindAppliers = append(bindAppliers, e.bindAppliers...)

	e.container = ctr.NewContainer(ctr.Option{
		BindAppliers: bindAppliers,
		FilterTypes:  e.filterTypes,
	})
}

func (e *_ContainerExecutor) Execute(taskContext spec.Context, triggerImpl spec.TriggerImpl, args []any) ex.Error {
	triggerInfo := triggerImpl.Info()
	execution := e.container.NewExecution(triggerInfo.Task().ERRunnerType(), triggerImpl.Method())
	execution.Execute(args, func(s *di.Seeder) {
		s.Seed(di.T[spec.Context](), taskContext)
		s.Seed(di.T[spec.TriggerInfo](), triggerInfo)
	})

	results := execution.Results()
	vpre.Check(len(results) > 0, "execution returned no results")
	if results[len(results)-1] == nil {
		return nil
	}
	err := results[len(results)-1].(ex.Error)
	vpre.Check(!reflectutil.IsNil(err), "task trigger %s returned typed nil error", triggerInfo.Name())
	return err
}
