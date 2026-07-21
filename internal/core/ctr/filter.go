package ctr

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
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

	args := vslice.Map(f.Context.Arguments(), reflect.ValueOf)
	resultValues := method.Call(args)
	results := vslice.Map(resultValues, func(v reflect.Value) any {
		return v.Interface()
	})

	f.Context.markFinished()
	f.Context.SetResults(results)
}
