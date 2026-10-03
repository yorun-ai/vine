package spec

func registerTestService(service *ServiceSpec) ServiceInfo {
	registry := NewRegistry()
	registry.Register(service)
	return service.Methods[0].Info().Service()
}
