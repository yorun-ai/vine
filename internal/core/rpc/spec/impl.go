package spec

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type ImplDict struct {
	serviceByName map[string]*_ServiceImpl
	methodByInfo  map[MethodInfo]*_MethodImpl
}

func NewImplDict() *ImplDict {
	return &ImplDict{
		serviceByName: map[string]*_ServiceImpl{},
		methodByInfo:  map[MethodInfo]*_MethodImpl{},
	}
}

func (d *ImplDict) Add(implType reflect.Type) {
	checkImplType(implType)
	serviceInfo, isERType := getServiceInfoByImplType(implType)
	vpre.CheckNil(d.serviceByName[serviceInfo.SkelName()], "service %s already added", serviceInfo.SkelName())

	serviceImpl := &_ServiceImpl{
		kind:     implType,
		isERType: isERType,
		info:     serviceInfo,
		methods:  map[string]*_MethodImpl{},
	}
	for _, methodInfo := range serviceInfo.Methods() {
		method, ok := implType.MethodByName(methodInfo.Name())
		vpre.Check(ok, "method %s not found on %s", methodInfo.Name(), implType)
		methodImpl := &_MethodImpl{
			kind:     implType,
			method:   method,
			isERType: isERType,
			info:     methodInfo,
		}
		serviceImpl.methods[methodInfo.SkelName()] = methodImpl
		d.methodByInfo[methodInfo] = methodImpl
	}
	d.serviceByName[serviceInfo.SkelName()] = serviceImpl
}

func checkImplType(implType reflect.Type) {
	vpre.Check(implType.Kind() == reflect.Pointer && implType.Elem().Kind() == reflect.Struct, "rpc impl type %s must be a pointer to struct", implType)
}

func (d *ImplDict) GetMethodImpl(serviceSkelName string, methodSkelName string) (MethodImpl, error) {
	serviceImpl, ok := d.serviceByName[serviceSkelName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceSkelName)
	}
	return serviceImpl.MethodImpl(methodSkelName)
}

func (d *ImplDict) GetMethodImplByInfo(methodInfo MethodInfo) (MethodImpl, error) {
	if methodInfo == nil {
		return nil, fmt.Errorf("method info is nil")
	}

	methodImpl, ok := d.methodByInfo[methodInfo]
	if !ok {
		return nil, fmt.Errorf("method %s not found", methodInfo.SkelName())
	}
	return methodImpl, nil
}

func (d *ImplDict) IterateServiceImpl(iterate func(info ServiceImpl)) {
	vmap.ForEach(d.serviceByName, func(_ string, info *_ServiceImpl) {
		iterate(info)
	})
}

type ServiceImpl interface {
	Type() reflect.Type
	IsERType() bool
	Info() ServiceInfo
	MethodImpl(methodSkelName string) (MethodImpl, error)
}

type _ServiceImpl struct {
	kind     reflect.Type
	isERType bool
	info     ServiceInfo
	methods  map[string]*_MethodImpl
}

func (i *_ServiceImpl) Type() reflect.Type {
	return i.kind
}

func (i *_ServiceImpl) IsERType() bool {
	return i.isERType
}

func (i *_ServiceImpl) Info() ServiceInfo {
	return i.info
}

func (i *_ServiceImpl) MethodImpl(methodSkelName string) (MethodImpl, error) {
	methodImpl, ok := i.methods[methodSkelName]
	if !ok {
		return nil, fmt.Errorf("method %s not found", methodSkelName)
	}
	return methodImpl, nil
}

type MethodImpl interface {
	Type() reflect.Type
	Method() reflect.Method
	IsERType() bool
	Info() MethodInfo
}

type _MethodImpl struct {
	kind     reflect.Type
	method   reflect.Method
	isERType bool
	info     MethodInfo
}

func (i *_MethodImpl) Type() reflect.Type {
	return i.kind
}

func (i *_MethodImpl) Method() reflect.Method {
	return i.method
}

func (i *_MethodImpl) IsERType() bool {
	return i.isERType
}

func (i *_MethodImpl) Info() MethodInfo {
	return i.info
}
