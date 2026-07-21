package log

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

type _MuteSuccessLogMethodKey struct {
	serviceSkelName string
	methodSkelName  string
}

var muteSuccessLogMethodKeys = map[_MuteSuccessLogMethodKey]struct{}{}

func MuteSuccessLog(method any) {
	methodPointer := reflect.ValueOf(method).Pointer()
	serviceSkelName, methodSkelName, ok := spec.GetMethodSkelNamesByPointer(methodPointer)
	vpre.Check(ok, "unknown rpc service method")
	muteSuccessLogMethodKeys[_MuteSuccessLogMethodKey{
		serviceSkelName: serviceSkelName,
		methodSkelName:  methodSkelName,
	}] = struct{}{}
}

func IsSuccessLogMuted(methodInfo spec.MethodInfo) bool {
	if methodInfo == nil {
		return false
	}
	_, ok := muteSuccessLogMethodKeys[_MuteSuccessLogMethodKey{
		serviceSkelName: methodInfo.Service().SkelName(),
		methodSkelName:  methodInfo.SkelName(),
	}]
	return ok
}
