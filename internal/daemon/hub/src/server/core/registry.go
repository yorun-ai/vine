package core

type RegistryCore struct {
	RegistryRepo RegistryRepo `inject:""`
	SchemaRepo   SchemaRepo   `inject:""`
}

func (m *RegistryCore) Register(reg AppRegistration) {
	m.RegistryRepo.SaveAppStatus(&AppStatus{
		Name:            reg.Name,
		InstanceId:      reg.InstanceId,
		Version:         reg.Version,
		Endpoint:        reg.Endpoint,
		ServiceHandlers: reg.ServiceHandlers,
		WebHandlers:     reg.WebHandlers,
		EventListeners:  reg.EventListeners,
		TaskRunners:     reg.TaskRunners,
	})
	for _, serviceHandler := range reg.ServiceHandlers {
		m.RegistryRepo.SaveRpcServiceRegistration(&RpcServiceRegistration{
			Endpoint:      serviceHandler.Endpoint,
			ServiceName:   serviceHandler.ServiceSkelName,
			AppName:       reg.Name,
			AppVersion:    reg.Version,
			AppInstanceId: reg.InstanceId,
		})
	}
	for _, webHandler := range reg.WebHandlers {
		m.RegistryRepo.SaveWebRegistration(&WebRegistration{
			Endpoint:      webHandler.Endpoint,
			WebSkelName:   webHandler.WebSkelName,
			AppName:       reg.Name,
			AppVersion:    reg.Version,
			AppInstanceId: reg.InstanceId,
		})
	}
	m.SchemaRepo.SaveDomainSchemasJSON(reg.Name, reg.InstanceId, reg.DomainSchemas)
}

func (m *RegistryCore) Unregister(appName string, instanceId string) {
	status, ok := m.RegistryRepo.GetAppStatus(appName, instanceId)
	if !ok {
		return
	}

	for _, serviceHandler := range status.ServiceHandlers {
		m.RegistryRepo.RemoveRpcServiceRegistration(serviceHandler.ServiceSkelName, status.Name, instanceId)
	}
	for _, webHandler := range status.WebHandlers {
		m.RegistryRepo.RemoveWebRegistration(webHandler.WebSkelName, status.Name, instanceId)
	}
	m.SchemaRepo.ReleaseDomainSchemas(appName, instanceId)
	m.RegistryRepo.RemoveAppStatus(appName, instanceId)
}

func (m *RegistryCore) Heartbeat(heartbeat AppHeartbeat) bool {
	status, ok := m.RegistryRepo.GetAppStatus(heartbeat.Name, heartbeat.InstanceId)
	if !ok {
		return false
	}

	if !m.RegistryRepo.KeepAppStatus(heartbeat.Name, heartbeat.InstanceId) {
		return false
	}
	for _, serviceHandler := range status.ServiceHandlers {
		if !m.RegistryRepo.KeepRpcServiceRegistration(serviceHandler.ServiceSkelName, status.Name, heartbeat.InstanceId) {
			return false
		}
	}
	for _, webHandler := range status.WebHandlers {
		if !m.RegistryRepo.KeepWebRegistration(webHandler.WebSkelName, status.Name, heartbeat.InstanceId) {
			return false
		}
	}
	return true
}
