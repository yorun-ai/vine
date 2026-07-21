package di

import (
	"reflect"
	"sync"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type _BaseInjector struct {
	resolvers map[reflect.Type]_Resolver

	dynamicMutex     sync.RWMutex
	dynamicResolvers map[reflect.Type]_Resolver

	resolveDynamic func(reflect.Type) _Resolver
	trackDispose   func(_DisposeFunc)
}

func (i *_BaseInjector) GetResolver(targetType reflect.Type) _Resolver {
	if resolver, ok := i.resolvers[targetType]; ok {
		return resolver
	}
	return i.getDynamicResolver(targetType)
}

func (i *_BaseInjector) getDynamicResolver(targetType reflect.Type) _Resolver {
	resolveDynamic := i.resolveDynamic
	if resolveDynamic == nil {
		return nil
	}

	i.dynamicMutex.RLock()
	if resolver, ok := i.dynamicResolvers[targetType]; ok {
		i.dynamicMutex.RUnlock()
		return resolver
	}
	i.dynamicMutex.RUnlock()

	resolver := resolveDynamic(targetType)
	if resolver == nil {
		return nil
	}

	i.dynamicMutex.Lock()
	defer i.dynamicMutex.Unlock()
	if cachedResolver, ok := i.dynamicResolvers[targetType]; ok {
		return cachedResolver
	}

	i.dynamicResolvers[targetType] = resolver
	return resolver
}

func (i *_BaseInjector) Get(targetType reflect.Type) reflect.Value {
	vpre.Check(i.canGet(targetType), "type %s is not bind to injector", targetType)
	return i.get(targetType, _BuildStack{targetType})
}

func (i *_BaseInjector) get(targetType reflect.Type, stack _BuildStack) reflect.Value {
	instance, _ := i.getWithDispose(targetType, stack)
	return instance
}

func (i *_BaseInjector) getWithDispose(targetType reflect.Type, stack _BuildStack) (reflect.Value, _DisposeFunc) {
	if stack.contains(targetType) && stack[len(stack)-1] != targetType {
		vpre.Panicf("cycle dependency detected at runtime, build stack=%s", stack.push(targetType))
	}

	instance, dispose := i.GetResolver(targetType).GetInstance(stack, targetType)
	if i.trackDispose != nil {
		i.trackDispose(dispose)
	}
	return instance, dispose
}

func (i *_BaseInjector) canGet(targetType reflect.Type) bool {
	return i.GetResolver(targetType) != nil
}

func (i *_BaseInjector) Resolve(targetPtr any) {
	targetType := parseResolveTargetType(targetPtr)
	vpre.Check(i.canGet(targetType), "type %s is not bind to injector", targetType)
	reflect.ValueOf(targetPtr).Elem().Set(i.Get(targetType))
}

func (i *_BaseInjector) Invoke(method any) []reflect.Value {
	mtdType := reflect.TypeOf(method)
	vpre.Check(reflectutil.IsFuncType(mtdType), "%s must be function", mtdType)

	var args []reflect.Value
	for idx := 0; idx < mtdType.NumIn(); idx++ {
		args = append(args, i.Get(mtdType.In(idx)))
	}

	mtdValue := reflect.ValueOf(method)
	return mtdValue.Call(args)
}
