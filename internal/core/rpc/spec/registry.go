package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

var serviceInfoBySkelName = map[string]*_ServiceInfo{}
var serviceInfoByDefaultEmbeddedType = map[reflect.Type]*_ServiceInfo{}
var erDefaultEmbeddedTypes = map[reflect.Type]struct{}{}

func GetMethodInfo(serviceSkelName string, methodSkelName string) (MethodInfo, bool) {
	serviceInfo := serviceInfoBySkelName[serviceSkelName]
	if serviceInfo == nil || !serviceInfo.serverRegistered {
		return nil, false
	}
	for _, methodInfo := range serviceInfo.Methods() {
		if methodInfo.SkelName() == methodSkelName {
			return methodInfo, true
		}
	}
	return nil, false
}

func RegisteredClientFactories() []any {
	var factories []any
	for _, serviceInfo := range serviceInfoBySkelName {
		if serviceInfo.ClientCtor() != nil {
			factories = append(factories, serviceInfo.ClientCtor())
		}
		if serviceInfo.ERClientCtor() != nil {
			factories = append(factories, serviceInfo.ERClientCtor())
		}
	}
	return factories
}

func GetServiceInfoByClientType(clientType reflect.Type) (ServiceInfo, bool) {
	for _, serviceInfo := range serviceInfoBySkelName {
		if serviceInfo.ClientType() == clientType {
			return serviceInfo, true
		}
	}
	return nil, false
}

func GetServiceInfoByERClientType(erClientType reflect.Type) (ServiceInfo, bool) {
	for _, serviceInfo := range serviceInfoBySkelName {
		if serviceInfo.ERClientType() == erClientType {
			return serviceInfo, true
		}
	}
	return nil, false
}

func Register(spec *ServiceSpec) {
	vpre.Check(isValidServiceSpecType(spec.Type), "invalid service spec type")

	serviceInfo, ok := serviceInfoBySkelName[spec.SkelName]
	if !ok {
		serviceInfo = &_ServiceInfo{
			name:     spec.Name,
			skelName: spec.SkelName,
			hash:     spec.Hash,
		}
		serviceInfoBySkelName[spec.SkelName] = serviceInfo
	}

	if spec.Type.setServer() {
		vpre.Check(!serviceInfo.serverRegistered, "service %s server already registered", spec.SkelName)
		vpre.Check((spec.DefaultServerType == nil) == (spec.DefaultERServerType == nil),
			"default server type and default er server type must both exist or both be nil")

		serviceInfo.serverRegistered = true
		serviceInfo.serverType = spec.ServerType
		serviceInfo.defaultServerType = spec.DefaultServerType
		serviceInfo.erServerType = spec.ERServerType
		serviceInfo.wrapperERServerCtor = spec.WrapperERServerCtor
		serviceInfo.defaultERServerType = spec.DefaultERServerType
		if spec.DefaultServerType != nil {
			registerDefaultEmbeddedTypes(spec.DefaultServerType, serviceInfo, false)
			registerDefaultEmbeddedTypes(spec.DefaultERServerType, serviceInfo, true)
		}
	}

	if spec.Type.setClient() {
		vpre.Check(!serviceInfo.clientRegistered, "service %s client already registered", spec.SkelName)

		serviceInfo.clientRegistered = true
		serviceInfo.clientType = spec.ClientType
		serviceInfo.clientCtor = spec.ClientCtor
		serviceInfo.erClientType = spec.ERClientType
		serviceInfo.erClientCtor = spec.ERClientCtor
	}

	registerMethodInfos(spec, serviceInfo)
}

func registerMethodInfos(serviceSpec *ServiceSpec, serviceInfo *_ServiceInfo) {
	if !serviceInfo.methodRegistered {
		serviceInfo.methodRegistered = true
		serviceInfo.methods = initMethodInfos(serviceSpec, serviceInfo)
		return
	}

	vpre.Check(
		len(serviceSpec.Methods) == len(serviceInfo.methods),
		"service %s method already registered",
		serviceSpec.SkelName,
	)

	methodInfosBySkelName := make(map[string]MethodInfo, len(serviceInfo.methods))
	for _, methodInfo := range serviceInfo.methods {
		methodInfosBySkelName[methodInfo.SkelName()] = methodInfo
	}
	for _, methodSpec := range serviceSpec.Methods {
		methodInfo := methodInfosBySkelName[methodSpec.SkelName]
		vpre.CheckNotNil(methodInfo, "service %s method already registered", serviceSpec.SkelName)
		methodSpec.info = methodInfo.(*_MethodInfo)
		registerMethodPointer(serviceSpec, methodSpec)
	}
}

func registerDefaultEmbeddedTypes(defaultServerType reflect.Type, serviceInfo *_ServiceInfo, isERType bool) {
	embeddedType := defaultServerType.Elem()
	serviceInfoByDefaultEmbeddedType[embeddedType] = serviceInfo
	if isERType {
		erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func initMethodInfos(serviceSpec *ServiceSpec, serviceInfo *_ServiceInfo) []MethodInfo {
	methodInfos := make([]MethodInfo, 0, len(serviceSpec.Methods))
	for _, methodSpec := range serviceSpec.Methods {
		validateArguments := methodSpec.ValidateArguments
		if validateArguments == nil {
			validateArguments = noopValidateArguments
		}
		validateResult := methodSpec.ValidateResult
		if validateResult == nil {
			validateResult = noopValidateResult
		}
		methodInfo := &_MethodInfo{
			name:                        methodSpec.Name,
			skelName:                    methodSpec.SkelName,
			fromedService:               serviceInfo,
			fullURLPath:                 "/" + serviceInfo.skelName + "/" + methodSpec.SkelName,
			argumentsType:               methodSpec.ArgumentsType,
			validateArguments:           validateArguments,
			resultType:                  methodSpec.ResultType,
			validateResult:              validateResult,
			argumentsContainsBinaryType: methodSpec.ArgumentsContainsBinaryType,
			resultContainsBinaryType:    methodSpec.ResultContainsBinaryType,
		}
		if methodInfo.HasArguments() {
			methodInfo.argumentFieldInfos = buildArgumentFieldInfos(methodInfo.argumentsType)
		}
		methodSpec.info = methodInfo
		registerMethodPointer(serviceSpec, methodSpec)
		methodInfos = append(methodInfos, methodInfo)
	}
	return methodInfos
}

func getServiceInfoByImplType(implType reflect.Type) (*_ServiceInfo, bool) {
	var serviceInfo *_ServiceInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		if info := serviceInfoByDefaultEmbeddedType[embeddedType]; info != nil {
			vpre.CheckNil(serviceInfo, "multiple embedded default server type found on %s", implType)
			serviceInfo = info
			_, isERType = erDefaultEmbeddedTypes[embeddedType]
		}
	}
	vpre.CheckNotNil(serviceInfo, "no embedded default server type found on %s", implType)
	return serviceInfo, isERType
}
