package spec

import (
	"reflect"
)

type ServiceSpecType string

const (
	ServiceSpecTypeClient ServiceSpecType = "client"
	ServiceSpecTypeServer ServiceSpecType = "server"
	ServiceSpecTypeBoth   ServiceSpecType = "both"
)

func isValidServiceSpecType(serviceSpecType ServiceSpecType) bool {
	return serviceSpecType == ServiceSpecTypeClient ||
		serviceSpecType == ServiceSpecTypeServer ||
		serviceSpecType == ServiceSpecTypeBoth
}

func (st ServiceSpecType) setServer() bool {
	return st == ServiceSpecTypeServer || st == ServiceSpecTypeBoth
}

func (st ServiceSpecType) setClient() bool {
	return st == ServiceSpecTypeClient || st == ServiceSpecTypeBoth
}

type ServiceSpec struct {
	Type ServiceSpecType

	Name     string
	SkelName string
	Hash     string

	ServerType        reflect.Type
	DefaultServerType reflect.Type
	ClientType        reflect.Type
	ClientCtor        any

	ERServerType        reflect.Type
	WrapperERServerCtor any
	DefaultERServerType reflect.Type
	ERClientType        reflect.Type
	ERClientCtor        any

	Methods []*MethodSpec
}

type MethodSpec struct {
	Name     string
	SkelName string

	ArgumentsType               reflect.Type
	ValidateArguments           func(any) error
	ResultType                  reflect.Type
	ValidateResult              func(any) error
	ArgumentsContainsBinaryType bool
	ResultContainsBinaryType    bool
	MethodFuncs                 []any

	info *_MethodInfo
}

func (m *MethodSpec) Info() MethodInfo {
	return m.info
}

// method pointer

type _MethodKey struct {
	serviceSkelName string
	methodSkelName  string
}

var methodSkelNamesByPointer = map[uintptr]_MethodKey{}

func registerMethodPointer(serviceSpec *ServiceSpec, methodSpec *MethodSpec) {
	methodKey := _MethodKey{
		serviceSkelName: serviceSpec.SkelName,
		methodSkelName:  methodSpec.SkelName,
	}
	for _, methodFunc := range methodSpec.MethodFuncs {
		methodPointer := reflect.ValueOf(methodFunc).Pointer()
		if methodPointer == 0 {
			continue
		}
		methodSkelNamesByPointer[methodPointer] = methodKey
	}
}

func GetMethodSkelNamesByPointer(methodPointer uintptr) (serviceSkelName string, methodSkelName string, ok bool) {
	methodKey, ok := methodSkelNamesByPointer[methodPointer]
	return methodKey.serviceSkelName, methodKey.methodSkelName, ok
}
