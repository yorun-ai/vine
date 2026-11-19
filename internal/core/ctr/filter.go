package ctr

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type FilterNext func()

type Filter interface {
	Filter(next FilterNext)
}

type _TargetInvocationFilter struct {
	Injector di.Injector `inject:""`
	Context  *Context    `inject:""`
}

func (f *_TargetInvocationFilter) Filter(next FilterNext) {
	instance := f.Injector.Get(f.Context.TargetType())
	method := f.Context.TargetMethodValue(instance)

	arguments := f.Context.Arguments()
	args := make([]reflect.Value, len(arguments))
	for i, arg := range arguments {
		args[i] = reflect.ValueOf(arg)
		if arg == nil {
			args[i] = nilArgument(method.Type(), i)
		}
	}
	resultValues := method.Call(args)
	results := vslice.Map(resultValues, func(v reflect.Value) any {
		return v.Interface()
	})

	f.Context.markFinished()
	f.Context.SetResults(results)
}

// reflect.Call expects variadic arguments individually, including nil elements.
func nilArgument(method reflect.Type, index int) reflect.Value {
	var param reflect.Type
	if method.IsVariadic() && index >= method.NumIn()-1 {
		param = method.In(method.NumIn() - 1).Elem()
	} else {
		param = method.In(index)
	}
	switch param.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflect.Zero(param)
	default:
		vpre.Panicf("cannot pass nil to parameter %d of type %s", index, param)
		return reflect.Value{}
	}
}
