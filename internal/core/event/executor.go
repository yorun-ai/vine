package event

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type Executor interface {
	Init(implDict spec.ListenerImplDict)
	Execute(eventContext spec.Context, listenerImpl spec.ListenerImpl, eventPayload any) ex.Error
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

func (e *_ContainerExecutor) Init(implDict spec.ListenerImplDict) {
	var bindAppliers []di.BindApplier
	implDict.IterateListenerImpl(func(listenerImpl spec.ListenerImpl) {
		eventInfo := listenerImpl.Info()
		bindAppliers = append(bindAppliers, func(b *di.Binder) {
			b.Bind(listenerImpl.Type()).In(di.ExecutionScope)
			if listenerImpl.IsERType() {
				b.Bind(eventInfo.ERListenerType()).ToImplementation(listenerImpl.Type()).In(di.ExecutionScope)
				return
			}
			b.Bind(eventInfo.ListenerType()).ToImplementation(listenerImpl.Type()).In(di.ExecutionScope)
			b.Bind(eventInfo.ERListenerType()).ToFactory(eventInfo.WrapperERListenerCtor()).In(di.ExecutionScope)
		})
	})

	bindAppliers = append(bindAppliers, func(b *di.Binder) {
		b.Bind(di.T[spec.Context]()).In(di.ExecutionScope)
		b.Bind(di.T[spec.EventInfo]()).In(di.ExecutionScope)
	})
	bindAppliers = append(bindAppliers, e.bindAppliers...)

	e.container = ctr.NewContainer(ctr.Option{
		BindAppliers: bindAppliers,
		FilterTypes:  e.filterTypes,
	})
}

func (e *_ContainerExecutor) Execute(eventContext spec.Context, listenerImpl spec.ListenerImpl, eventPayload any) ex.Error {
	eventInfo := listenerImpl.Info()
	execution := e.container.NewExecution(eventInfo.ERListenerType(), listenerImpl.Method())
	execution.Execute([]any{eventPayload}, func(s *di.Seeder) {
		s.Seed(di.T[spec.Context](), eventContext)
		s.Seed(di.T[spec.EventInfo](), eventInfo)
	})

	results := execution.Results()
	vpre.Check(len(results) > 0, "execution returned no results")
	if results[len(results)-1] == nil {
		return nil
	}

	err := results[len(results)-1].(ex.Error)
	vpre.Check(!reflectutil.IsNil(err), "event %s returned typed nil error", eventInfo.Name())
	return err
}
