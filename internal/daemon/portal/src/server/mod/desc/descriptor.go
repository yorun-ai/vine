package desc

import (
	"strings"

	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

// Actor

// Actor returns an immutable actor descriptor.
func (m *Manager) Actor(actorSkelName string) (*watched.DescriptorActor, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	actor, ok := m.actorsBySkelName[actorSkelName]
	return actor, ok
}

func (m *Manager) loadActors() {
	valuesByKey, subscription := m.Watch.LoadListAndSubscribe(m.watchContext, watched.FormatDescriptorActorPrefix(), m.handleActorEvent)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range valuesByKey {
		actor := decodeActor(value)
		m.setActorLocked(key, actor)
	}
	subscription.Start()
}

func (m *Manager) handleActorEvent(event hubwatch.Event) {
	m.admissionMutex.Lock()
	defer m.admissionMutex.Unlock()
	m.mutex.Lock()
	if m.stopped {
		m.mutex.Unlock()
		return
	}
	if event.Kind == hubwatch.EventKindDelete {
		m.removeActorLocked(event.Key)
	} else {
		m.setActorLocked(event.Key, decodeActor(event.Value))
	}
	m.mutex.Unlock()
	m.notifyAdmission()
}

func (m *Manager) setActorLocked(key string, actor *watched.DescriptorActor) {
	m.removeActorLocked(key)
	m.actorNamesByKey[key] = actor.SkelName
	m.actorsBySkelName[actor.SkelName] = actor
}

func (m *Manager) removeActorLocked(key string) {
	if name, ok := m.actorNamesByKey[key]; ok {
		delete(m.actorsBySkelName, name)
		delete(m.actorNamesByKey, key)
	}
}

func decodeActor(value string) *watched.DescriptorActor {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorActor](value)
}

// Service

// Service returns an immutable API service descriptor.
func (m *Manager) Service(serviceSkelName string) (*watched.DescriptorService, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	service, ok := m.servicesBySkelName[serviceSkelName]
	return service, ok
}

func (m *Manager) loadServices() {
	snapshot := m.Watch.LoadList("descriptor")
	valuesByKey, subscription := m.Watch.LoadListAndSubscribe(m.watchContext, watched.FormatDescriptorServicePrefix(), m.handleServiceEvent)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range snapshot {
		if strings.HasPrefix(key, watched.FormatDescriptorDataPrefix()+":") || strings.HasPrefix(key, watched.FormatDescriptorEnumPrefix()+":") {
			entry := newTypeDescriptor(key)
			vpre.CheckNilError(entry.setValue(value), "decode descriptor type failed")
			m.typeInputs[key] = entry
		}
	}
	for key, value := range valuesByKey {
		m.setServiceLocked(key, decodeService(value))
	}
	m.requestTypeSyncLocked()
	subscription.Start()
}

func (m *Manager) handleServiceEvent(event hubwatch.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.stopped {
		return
	}

	if event.Kind == hubwatch.EventKindDelete {
		m.removeServiceLocked(event.Key)
		m.requestTypeSyncLocked()
		return
	}
	m.setServiceLocked(event.Key, decodeService(event.Value))
	m.requestTypeSyncLocked()
}

func (m *Manager) setServiceLocked(key string, service *watched.DescriptorService) {
	m.removeServiceLocked(key)
	if !service.Api {
		return
	}
	m.serviceNamesByKey[key] = service.SkelName
	m.servicesBySkelName[service.SkelName] = service
}

func (m *Manager) removeServiceLocked(key string) {
	if name, ok := m.serviceNamesByKey[key]; ok {
		delete(m.servicesBySkelName, name)
		delete(m.serviceNamesByKey, key)
	}
}

func decodeService(value string) *watched.DescriptorService {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorService](value)
}

// Resource

func (m *Manager) loadResources() {
	valuesByKey, subscription := m.Watch.LoadListAndSubscribe(m.watchContext, watched.FormatDescriptorResourcePrefix(), m.handleResourceEvent)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range valuesByKey {
		m.setResourceLocked(key, decodeResource(value))
	}
	subscription.Start()
}

func (m *Manager) handleResourceEvent(event hubwatch.Event) {
	m.admissionMutex.Lock()
	defer m.admissionMutex.Unlock()
	m.mutex.Lock()
	if m.stopped {
		m.mutex.Unlock()
		return
	}
	if event.Kind == hubwatch.EventKindDelete {
		m.removeResourceLocked(event.Key)
	} else {
		m.setResourceLocked(event.Key, decodeResource(event.Value))
	}
	m.mutex.Unlock()
	m.notifyAdmission()
}

func (m *Manager) setResourceLocked(key string, resource *watched.DescriptorResource) {
	m.removeResourceLocked(key)
	m.resourceNamesByKey[key] = resource.SkelName
	m.resourcesBySkelName[resource.SkelName] = resource
}

func (m *Manager) removeResourceLocked(key string) {
	if name, ok := m.resourceNamesByKey[key]; ok {
		delete(m.resourcesBySkelName, name)
		delete(m.resourceNamesByKey, key)
	}
}

func decodeResource(value string) *watched.DescriptorResource {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorResource](value)
}

// Web

// Web returns an immutable Web descriptor.
func (m *Manager) Web(webSkelName string) (*watched.DescriptorWeb, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	web, ok := m.websBySkelName[webSkelName]
	return web, ok
}

func (m *Manager) loadWebs() {
	valuesByKey, subscription := m.Watch.LoadListAndSubscribe(m.watchContext, watched.FormatDescriptorWebPrefix(), m.handleWebEvent)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range valuesByKey {
		m.setWebLocked(key, decodeWeb(value))
	}
	subscription.Start()
}

func (m *Manager) handleWebEvent(event hubwatch.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.stopped {
		return
	}

	if event.Kind == hubwatch.EventKindDelete {
		m.removeWebLocked(event.Key)
		return
	}
	m.setWebLocked(event.Key, decodeWeb(event.Value))
}

func (m *Manager) setWebLocked(key string, web *watched.DescriptorWeb) {
	m.removeWebLocked(key)
	m.webNamesByKey[key] = web.SkelName
	m.websBySkelName[web.SkelName] = web
}

func (m *Manager) removeWebLocked(key string) {
	if name, ok := m.webNamesByKey[key]; ok {
		delete(m.websBySkelName, name)
		delete(m.webNamesByKey, key)
	}
}

func decodeWeb(value string) *watched.DescriptorWeb {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorWeb](value)
}

// Resource returns an immutable resource descriptor.
func (m *Manager) Resource(name string) (*watched.DescriptorResource, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, ok := m.resourcesBySkelName[name]
	return value, ok
}
