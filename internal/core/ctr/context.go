package ctr

import (
	"reflect"

	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type Context struct {
	targetType        reflect.Type
	targetMethodName  string
	targetMethod      reflect.Method
	forceMethodByName bool
	arguments         []any
	results           []any

	isFinished bool
}

func newContext(targetType reflect.Type, targetMethodName string) *Context {
	return &Context{
		targetType:        targetType,
		targetMethodName:  targetMethodName,
		forceMethodByName: true,
	}
}

func newContextWithMethod(targetType reflect.Type, targetMethod reflect.Method) *Context {
	return &Context{
		targetType:        targetType,
		targetMethodName:  targetMethod.Name,
		targetMethod:      targetMethod,
		forceMethodByName: false,
	}
}

func (c *Context) TargetType() reflect.Type {
	return c.targetType
}

func (c *Context) SetTargetType(targetType reflect.Type) {
	vpre.Check(!c.isFinished, "can't change TargetType after execution finished")
	c.targetType = targetType
	c.forceMethodByName = true
}

func (c *Context) TargetMethodName() string {
	return c.targetMethodName
}

func (c *Context) SetTargetMethodName(targetMethod string) {
	vpre.Check(!c.isFinished, "can't change TargetMethodName after execution finished")
	c.targetMethodName = targetMethod
	c.forceMethodByName = true
}

func (c *Context) Arguments() []any {
	return vslice.Clone(c.arguments)
}

func (c *Context) SetArguments(arguments []any) {
	vpre.Check(!c.isFinished, "can't change Arguments after execution finished")
	c.arguments = vslice.Clone(arguments)
}

func (c *Context) Results() []any {
	return vslice.Clone(c.results)
}

// SetResults stays available after finish so filters can still rewrite returned values.
func (c *Context) SetResults(results []any) {
	c.results = vslice.Clone(results)
}

func (c *Context) TargetMethodValue(instance reflect.Value) reflect.Value {
	if !c.forceMethodByName {
		return instance.Method(c.targetMethod.Index)
	}
	method := getMethodByName(instance.Type(), c.targetMethodName)
	c.targetMethod = method
	c.forceMethodByName = false
	return instance.Method(method.Index)
}

func (c *Context) markFinished() {
	c.isFinished = true
}
