package server

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

type DefaultExecutorOption interface {
	apply(*_DefaultExecutor)
}

type _DefaultExecutor struct {
	instanceByType map[reflect.Type]reflect.Value
}

type _DefaultExecutorOptionFunc func(*_DefaultExecutor)

type _InjectionFunc func(*_DefaultExecutor, reflect.Value, spec.Context)

var injectionFuncByType = map[reflect.Type]_InjectionFunc{}

func NewDefaultExecutor(options ...DefaultExecutorOption) Executor {
	executor := &_DefaultExecutor{
		instanceByType: map[reflect.Type]reflect.Value{},
	}
	for _, option := range options {
		option.apply(executor)
	}
	return executor
}

func (f _DefaultExecutorOptionFunc) apply(executor *_DefaultExecutor) {
	f(executor)
}

func With(instance any) DefaultExecutorOption {
	vpre.CheckNotNil(instance, "default executor instance cannot be nil")
	return WithAs(reflect.TypeOf(instance), instance)
}

func WithAs(targetType reflect.Type, instance any) DefaultExecutorOption {
	vpre.CheckNotNil(targetType, "default executor target type cannot be nil")
	vpre.CheckNotNil(instance, "default executor instance cannot be nil")

	instanceValue := reflect.ValueOf(instance)
	vpre.Check(instanceValue.Type().AssignableTo(targetType), "instance type %s cannot be assigned to %s", instanceValue.Type(), targetType)
	return _DefaultExecutorOptionFunc(func(executor *_DefaultExecutor) {
		executor.instanceByType[targetType] = instanceValue
	})
}

func (e *_DefaultExecutor) Init(infoDict spec.ImplDict) {
	infoDict.IterateServiceImpl(func(serviceImpl spec.ServiceImpl) {
		implType := serviceImpl.Type()
		if _, ok := injectionFuncByType[implType]; ok {
			return
		}
		injectionFuncByType[implType] = newInjectionFunc(implType)
	})
}

func (e *_DefaultExecutor) Execute(rpcContext spec.Context, methodImpl spec.MethodImpl, args []any) (any, ex.Error) {
	methodInfo := methodImpl.Info()
	methodValue := reflect.New(methodImpl.Type().Elem())
	methodArgs := []reflect.Value{methodValue}
	for _, arg := range args {
		methodArgs = append(methodArgs, reflect.ValueOf(arg))
	}

	e.inject(methodValue, rpcContext)
	results := methodImpl.Method().Func.Call(methodArgs)
	if methodImpl.IsERType() {
		if err, _ := results[len(results)-1].Interface().(ex.Error); err != nil {
			return nil, err
		}
	}

	if !methodInfo.HasResult() {
		return nil, nil
	}

	result := results[0].Interface()
	if err := methodInfo.ValidateResult(result); err != nil {
		vpre.Panicf("%s", err.Error())
	}
	return result, nil
}

func (e *_DefaultExecutor) inject(methodValue reflect.Value, rpcContext spec.Context) {
	injectionFuncByType[methodValue.Type()](e, methodValue, rpcContext)
}

func newInjectionFunc(implType reflect.Type) _InjectionFunc {
	structValue := reflect.New(implType.Elem()).Elem()
	matchIndex := -1
	var injectionFuncs []_InjectionFunc
	for fieldIndex := 0; fieldIndex < structValue.NumField(); fieldIndex++ {
		fieldType := structValue.Type().Field(fieldIndex).Type
		fieldValue := structValue.Field(fieldIndex)
		if fieldType == reflect.TypeFor[spec.Context]() {
			vpre.Check(matchIndex == -1, "multiple spec.Context fields found on %s", structValue.Type())
			vpre.Check(fieldValue.CanSet(), "spec.Context field on %s cannot be set", structValue.Type())
			matchIndex = fieldIndex
			injectionFuncs = append(injectionFuncs, newContextInjectionFunc(fieldIndex))
			continue
		}

		injectionFuncs = append(injectionFuncs, newInstanceInjectionFunc(fieldIndex, fieldType))
	}
	if len(injectionFuncs) == 0 {
		return func(*_DefaultExecutor, reflect.Value, spec.Context) {}
	}

	return func(executor *_DefaultExecutor, methodValue reflect.Value, rpcContext spec.Context) {
		for _, injectionFunc := range injectionFuncs {
			injectionFunc(executor, methodValue, rpcContext)
		}
	}
}

func newContextInjectionFunc(fieldIndex int) _InjectionFunc {
	return func(_ *_DefaultExecutor, methodValue reflect.Value, rpcContext spec.Context) {
		methodValue.Elem().Field(fieldIndex).Set(reflect.ValueOf(rpcContext))
	}
}

func newInstanceInjectionFunc(fieldIndex int, fieldType reflect.Type) _InjectionFunc {
	return func(executor *_DefaultExecutor, methodValue reflect.Value, _ spec.Context) {
		instanceValue := executor.instanceByType[fieldType]
		if !instanceValue.IsValid() {
			return
		}
		methodValue.Elem().Field(fieldIndex).Set(instanceValue)
	}
}
