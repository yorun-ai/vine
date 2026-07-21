package ctr

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type Option struct {
	BindAppliers []di.BindApplier
	FilterTypes  []reflect.Type
}

type Container interface {
	NewExecution(targetType reflect.Type, targetMethod reflect.Method) Execution
}

type _Container struct {
	option *Option

	filterTypes []reflect.Type
	injector    di.PlainInjector
}

func NewContainer(option Option) Container {
	c := &_Container{option: &option}
	c.init()
	return c
}

func (c *_Container) init() {
	c.filterTypes = vslice.Clone(c.option.FilterTypes)
	c.filterTypes = append(c.filterTypes, reflect.TypeFor[*_TargetInvocationFilter]())
	c.checkFilterTypes()

	bindAppliers := []di.BindApplier{c.bindContextAndFilters}
	bindAppliers = append(bindAppliers, c.option.BindAppliers...)
	c.injector = di.NewInjector(bindAppliers...)
}

func (c *_Container) checkFilterTypes() {
	filterType := reflect.TypeFor[Filter]()
	for _, kind := range c.filterTypes {
		vpre.Check(reflectutil.IsStructPointerType(kind), "filter type %s must be pointer to struct", kind)
		vpre.Check(kind.Implements(filterType), "filter type %s must implement %s", kind, filterType)
	}
}

func (c *_Container) bindContextAndFilters(b *di.Binder) {
	b.Bind(di.T[*Context]())
	for _, filterType := range c.filterTypes {
		b.Bind(filterType)
	}
}
