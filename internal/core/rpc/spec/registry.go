package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type Registry struct {
	serviceInfoBySkelName            map[string]*_ServiceInfo
	serviceInfoByDefaultEmbeddedType map[reflect.Type]*_ServiceInfo
	erDefaultEmbeddedTypes           map[reflect.Type]struct{}
	methodSkelNamesByPointer         map[uintptr]_MethodKey
}

func NewRegistry() *Registry {
	return &Registry{
		serviceInfoBySkelName:            map[string]*_ServiceInfo{},
		serviceInfoByDefaultEmbeddedType: map[reflect.Type]*_ServiceInfo{},
		erDefaultEmbeddedTypes:           map[reflect.Type]struct{}{},
		methodSkelNamesByPointer:         map[uintptr]_MethodKey{},
	}
}

var defaultRegistry = NewRegistry()

func GetMethodInfo(serviceSkelName string, methodSkelName string) (MethodInfo, bool) {
	return defaultRegistry.GetMethodInfo(serviceSkelName, methodSkelName)
}

func (r *Registry) GetMethodInfo(serviceSkelName string, methodSkelName string) (MethodInfo, bool) {
	serviceInfo := r.serviceInfoBySkelName[serviceSkelName]
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
	return defaultRegistry.RegisteredClientFactories()
}

func (r *Registry) RegisteredClientFactories() []any {
	var factories []any
	for _, serviceInfo := range r.serviceInfoBySkelName {
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
	return defaultRegistry.GetServiceInfoByClientType(clientType)
}

func (r *Registry) GetServiceInfoByClientType(clientType reflect.Type) (ServiceInfo, bool) {
	for _, serviceInfo := range r.serviceInfoBySkelName {
		if serviceInfo.ClientType() == clientType {
			return serviceInfo, true
		}
	}
	return nil, false
}

func GetServiceInfoByERClientType(erClientType reflect.Type) (ServiceInfo, bool) {
	return defaultRegistry.GetServiceInfoByERClientType(erClientType)
}

func (r *Registry) GetServiceInfoByERClientType(erClientType reflect.Type) (ServiceInfo, bool) {
	for _, serviceInfo := range r.serviceInfoBySkelName {
		if serviceInfo.ERClientType() == erClientType {
			return serviceInfo, true
		}
	}
	return nil, false
}

func Register(spec *ServiceSpec) {
	defaultRegistry.Register(spec)
}

func (r *Registry) Register(spec *ServiceSpec) {
	vpre.Check(isValidServiceSpecType(spec.Type), "invalid service spec type")

	serviceInfo, ok := r.serviceInfoBySkelName[spec.SkelName]
	if !ok {
		serviceInfo = &_ServiceInfo{
			name:     spec.Name,
			skelName: spec.SkelName,
			hash:     spec.Hash,
		}
		r.serviceInfoBySkelName[spec.SkelName] = serviceInfo
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
			r.registerDefaultEmbeddedTypes(spec.DefaultServerType, serviceInfo, false)
			r.registerDefaultEmbeddedTypes(spec.DefaultERServerType, serviceInfo, true)
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

	r.registerMethodInfos(spec, serviceInfo)
}

func (r *Registry) registerMethodInfos(serviceSpec *ServiceSpec, serviceInfo *_ServiceInfo) {
	if !serviceInfo.methodRegistered {
		serviceInfo.methodRegistered = true
		serviceInfo.methods = r.initMethodInfos(serviceSpec, serviceInfo)
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
		r.registerMethodPointer(serviceSpec, methodSpec)
	}
}

func (r *Registry) registerDefaultEmbeddedTypes(defaultServerType reflect.Type, serviceInfo *_ServiceInfo, isERType bool) {
	embeddedType := defaultServerType.Elem()
	r.serviceInfoByDefaultEmbeddedType[embeddedType] = serviceInfo
	if isERType {
		r.erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func (r *Registry) initMethodInfos(serviceSpec *ServiceSpec, serviceInfo *_ServiceInfo) []MethodInfo {
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
			argumentsSensitive:          methodSpec.ArgumentsSensitive,
			validateArguments:           validateArguments,
			resultType:                  methodSpec.ResultType,
			resultSensitive:             methodSpec.ResultSensitive,
			validateResult:              validateResult,
			argumentsContainsBinaryType: methodSpec.ArgumentsContainsBinaryType,
			resultContainsBinaryType:    methodSpec.ResultContainsBinaryType,
			cloneArguments:              methodSpec.CloneArguments,
			cloneResult:                 methodSpec.CloneResult,
		}
		if methodInfo.HasArguments() {
			methodInfo.argumentFieldInfos = buildArgumentFieldInfos(methodInfo.argumentsType)
		}
		methodSpec.info = methodInfo
		r.registerMethodPointer(serviceSpec, methodSpec)
		methodInfos = append(methodInfos, methodInfo)
	}
	return methodInfos
}

func getServiceInfoByImplType(implType reflect.Type) (*_ServiceInfo, bool) {
	return defaultRegistry.getServiceInfoByImplType(implType)
}

func (r *Registry) getServiceInfoByImplType(implType reflect.Type) (*_ServiceInfo, bool) {
	var serviceInfo *_ServiceInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		if info := r.serviceInfoByDefaultEmbeddedType[embeddedType]; info != nil {
			vpre.CheckNil(serviceInfo, "multiple embedded default server type found on %s", implType)
			serviceInfo = info
			_, isERType = r.erDefaultEmbeddedTypes[embeddedType]
		}
	}
	vpre.CheckNotNil(serviceInfo, "no embedded default server type found on %s", implType)
	return serviceInfo, isERType
}
