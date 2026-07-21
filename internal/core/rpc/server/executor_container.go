package server

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

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
	implDict.IterateServiceImpl(func(handler spec.ServiceImpl) {
		info := handler.Info()
		bindAppliers = append(bindAppliers, func(b *di.Binder) {
			b.Bind(handler.Type()).In(di.ExecutionScope)
			if handler.IsERType() {
				b.Bind(info.ERServerType()).ToImplementation(handler.Type()).In(di.ExecutionScope)
				return
			}
			b.Bind(info.ServerType()).ToImplementation(handler.Type()).In(di.ExecutionScope)
			b.Bind(info.ERServerType()).ToFactory(info.WrapperERServerCtor()).In(di.ExecutionScope)
		})
	})

	bindAppliers = append(bindAppliers, func(b *di.Binder) {
		b.Bind(di.T[spec.Context]()).In(di.ExecutionScope)
		b.Bind(di.T[spec.MethodInfo]()).In(di.ExecutionScope)
	})
	bindAppliers = append(bindAppliers, e.bindAppliers...)

	e.container = ctr.NewContainer(ctr.Option{
		BindAppliers: bindAppliers,
		FilterTypes:  e.filterTypes,
	})
}

func (e *_ContainerExecutor) Execute(rpcContext spec.Context, methodImpl spec.MethodImpl, args []any) (any, ex.Error) {
	methodInfo := methodImpl.Info()
	execution := e.container.NewExecution(methodInfo.Service().ERServerType(), methodImpl.Method())
	execution.Execute(args, func(s *di.Seeder) {
		s.Seed(di.T[spec.Context](), rpcContext)
		s.Seed(di.T[spec.MethodInfo](), methodInfo)
	})

	results := execution.Results()
	vpre.Check(len(results) > 0, "execution returned no results")

	if err, _ := results[len(results)-1].(ex.Error); err != nil {
		return nil, err
	}

	if !methodInfo.HasResult() {
		return nil, nil
	}

	result := results[0]
	if err := methodInfo.ValidateResult(result); err != nil {
		vpre.Panicf("%s", err.Error())
	}
	return result, nil
}
