package core

import (
	"time"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon"
)

type RegistryCore struct {
	RegistryRepo   RegistryRepo   `inject:""`
	DescriptorRepo DescriptorRepo `inject:""`
}

func (m *RegistryCore) Register(reg AppRegistration) {
	// Reject invalid descriptors before publishing application status or endpoints.
	m.DescriptorRepo.SaveDomainDescriptors(reg.Name, reg.InstanceId, reg.DomainDescriptors)
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
	apiServices := map[string]bool{}
	for _, domain := range reg.DomainDescriptors {
		for _, service := range domain.Services {
			if service.Api {
				apiServices[service.SkelName] = true
			}
		}
	}
	for _, serviceHandler := range reg.ServiceHandlers {
		m.RegistryRepo.SaveRpcServiceRegistration(&RpcServiceRegistration{
			Endpoint:       serviceHandler.Endpoint,
			ServerIdentity: daemon.LinkIdentity,
			ServiceName:    serviceHandler.ServiceSkelName,
			Api:            apiServices[serviceHandler.ServiceSkelName],
			AppName:        reg.Name,
			AppVersion:     reg.Version,
			AppInstanceId:  reg.InstanceId,
		})
	}
	for _, webHandler := range reg.WebHandlers {
		m.RegistryRepo.SaveWebRegistration(&WebRegistration{
			Endpoint:       webHandler.Endpoint,
			ServerIdentity: daemon.LinkIdentity,
			WebSkelName:    webHandler.WebSkelName,
			AppName:        reg.Name,
			AppVersion:     reg.Version,
			AppInstanceId:  reg.InstanceId,
		})
	}
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
	m.DescriptorRepo.ReleaseDomainDescriptors(appName, instanceId)
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

// RegisterDescriptors registers descriptor ownership without creating application endpoints.
func (m *RegistryCore) RegisterDescriptors(ownerName string, ownerId string, descriptors []*skeldesc.Domain) {
	m.DescriptorRepo.SaveDomainDescriptors(ownerName, ownerId, descriptors)
}

// SweepExpiredLeases unregisters expired instances, ignoring renewed or missing statuses.
func (m *RegistryCore) SweepExpiredLeases() bool {
	removed := false
	for {
		leases := m.RegistryRepo.PopExpiredAppLeases()
		if len(leases) == 0 {
			return removed
		}
		for _, lease := range leases {
			status, ok := m.RegistryRepo.GetAppStatus(lease.Name, lease.InstanceId)
			if !ok || time.Now().Before(status.ExpiresAt) {
				continue
			}
			m.Unregister(lease.Name, lease.InstanceId)
			removed = true
		}
	}
}
