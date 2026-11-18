package core

import (
	"slices"
	"sync"
	"time"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/util/goutil"
)

var registryLogger = logger.New("daemon:hub:registry")

type RegistryCore struct {
	di.SingletonScoped
	locks          _InstanceLocks
	RegistryRepo   RegistryRepo   `inject:""`
	DescriptorRepo DescriptorRepo `inject:""`
}

func (m *RegistryCore) Register(reg AppRegistration) {
	unlock := m.locks.lock(reg.Name, reg.InstanceId)
	defer unlock()
	// Reject invalid descriptors before publishing application status or endpoints.
	m.DescriptorRepo.SaveDomainDescriptors(reg.Name, reg.InstanceId, reg.DomainDescriptors)
	// Withdraw omitted endpoints before replacing their cleanup metadata. They no
	// longer expire independently, and a failed removal must remain retryable.
	if previous, ok := m.RegistryRepo.GetAppStatus(reg.Name, reg.InstanceId); ok {
		for _, handler := range previous.ServiceHandlers {
			if !slices.ContainsFunc(reg.ServiceHandlers, func(next ServiceHandlerRegistration) bool {
				return next.ServiceSkelName == handler.ServiceSkelName
			}) {
				m.RegistryRepo.RemoveRpcServiceRegistration(handler.ServiceSkelName, reg.Name, reg.InstanceId)
			}
		}
		for _, handler := range previous.WebHandlers {
			if !slices.ContainsFunc(reg.WebHandlers, func(next WebHandlerRegistration) bool {
				return next.WebSkelName == handler.WebSkelName
			}) {
				m.RegistryRepo.RemoveWebRegistration(handler.WebSkelName, reg.Name, reg.InstanceId)
			}
		}
	}
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
	unlock := m.locks.lock(appName, instanceId)
	defer unlock()
	m.unregister(appName, instanceId)
}

func (m *RegistryCore) unregister(appName string, instanceId string) {
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
	unlock := m.locks.lock(heartbeat.Name, heartbeat.InstanceId)
	defer unlock()
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
	unlock := m.locks.lock(ownerName, ownerId)
	defer unlock()
	m.DescriptorRepo.SaveDomainDescriptors(ownerName, ownerId, descriptors)
}

// SweepExpiredLeases retries indexed candidates independently. A failed cleanup
// retains its status and lease and does not block other expired instances.
func (m *RegistryCore) SweepExpiredLeases() bool {
	removed := false
	for _, lease := range m.RegistryRepo.ListExpiredAppLeases() {
		goutil.RunWithRecover(func(err any) {
			registryLogger.Error("expired application cleanup failed", "app", lease.Name, "instanceId", lease.InstanceId, "error", err)
		}, func() {
			unlock := m.locks.lock(lease.Name, lease.InstanceId)
			defer unlock()
			status, ok := m.RegistryRepo.GetAppStatus(lease.Name, lease.InstanceId)
			if !ok {
				// A prior attempt may have removed status before acknowledging the index.
				m.RegistryRepo.RemoveAppStatus(lease.Name, lease.InstanceId)
				removed = true
				return
			}
			if status.ExpiresAt.IsZero() || time.Now().Before(status.ExpiresAt) {
				return
			}
			m.unregister(lease.Name, lease.InstanceId)
			removed = true
		})
	}
	return removed
}

// _InstanceLocks serializes each identity while allowing unrelated instances to
// renew concurrently. Idle locks are discarded after their last waiter exits.
type _InstanceLocks struct {
	mutex sync.Mutex
	items map[[2]string]*_InstanceLock
}
type _InstanceLock struct {
	mutex sync.Mutex
	refs  int
}

func (l *_InstanceLocks) lock(name string, instanceId string) func() {
	key := [2]string{name, instanceId}
	l.mutex.Lock()
	if l.items == nil {
		l.items = make(map[[2]string]*_InstanceLock)
	}
	item := l.items[key]
	if item == nil {
		item = new(_InstanceLock)
		l.items[key] = item
	}
	item.refs++
	l.mutex.Unlock()
	item.mutex.Lock()
	return func() {
		item.mutex.Unlock()
		l.mutex.Lock()
		item.refs--
		if item.refs == 0 {
			delete(l.items, key)
		}
		l.mutex.Unlock()
	}
}
