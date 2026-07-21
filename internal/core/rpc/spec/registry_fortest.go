package spec

func ConvertSpecToInfoForTest(serviceSpec *ServiceSpec) ServiceInfo {
	serviceInfo := &_ServiceInfo{
		name:     serviceSpec.Name,
		skelName: serviceSpec.SkelName,
		hash:     serviceSpec.Hash,

		serverType:        serviceSpec.ServerType,
		defaultServerType: serviceSpec.DefaultServerType,
		clientType:        serviceSpec.ClientType,
		clientCtor:        serviceSpec.ClientCtor,

		erServerType:        serviceSpec.ERServerType,
		wrapperERServerCtor: serviceSpec.WrapperERServerCtor,
		defaultERServerType: serviceSpec.DefaultERServerType,
		erClientType:        serviceSpec.ERClientType,
		erClientCtor:        serviceSpec.ERClientCtor,
	}
	serviceInfo.methods = initMethodInfos(serviceSpec, serviceInfo)
	return serviceInfo
}
