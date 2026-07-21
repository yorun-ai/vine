package access

import (
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vcode"
)

// Actor

func (a *Access) actorSchema(actorSkelName string) (*redised.SchemaActor, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	actor, ok := a.actorsBySkelName[actorSkelName]
	return actor, ok
}

func (a *Access) loadActors() {
	valuesByKey := a.Redis.LoadListAndSubscribe(a.Context, redised.FormatSchemaActorPrefix(), a.handleActorEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		actor := decodeActor(value)
		a.setActorLocked(key, actor)
	}
}

func (a *Access) handleActorEvent(event hubredis.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubredis.EventKindDelete {
		a.removeActorLocked(event.Key)
		return
	}
	a.setActorLocked(event.Key, decodeActor(event.Value))
}

func (a *Access) setActorLocked(key string, actor *redised.SchemaActor) {
	a.removeActorLocked(key)
	a.actorNamesByKey[key] = actor.SkelName
	a.actorsBySkelName[actor.SkelName] = actor
	a.watchActorAuthServiceLocked(key, actor)
	a.watchActorPermServiceLocked(key, actor)
}

func (a *Access) removeActorLocked(key string) {
	a.releaseActorAuthServiceLocked(key)
	a.releaseActorPermServiceLocked(key)
	if name, ok := a.actorNamesByKey[key]; ok {
		delete(a.actorsBySkelName, name)
		delete(a.actorNamesByKey, key)
	}
}

func (a *Access) watchActorAuthServiceLocked(key string, actor *redised.SchemaActor) {
	if actor.AuthService == nil {
		return
	}
	a.authServiceWatchersByActorKey[key] = a.Epmgr.WatchRpc(actor.AuthService.SkelName)
}

func (a *Access) releaseActorAuthServiceLocked(key string) {
	watcher := a.authServiceWatchersByActorKey[key]
	if watcher == nil {
		return
	}
	watcher.Release()
	delete(a.authServiceWatchersByActorKey, key)
}

func (a *Access) watchActorPermServiceLocked(key string, actor *redised.SchemaActor) {
	if !actor.PermEnabled || actor.PermService == nil {
		return
	}
	a.permServiceWatchersByActorKey[key] = a.Epmgr.WatchRpc(actor.PermService.SkelName)
}

func (a *Access) releaseActorPermServiceLocked(key string) {
	watcher := a.permServiceWatchersByActorKey[key]
	if watcher == nil {
		return
	}
	watcher.Release()
	delete(a.permServiceWatchersByActorKey, key)
}

func decodeActor(value string) *redised.SchemaActor {
	return vcode.MustUnmarshalJsonS[*redised.SchemaActor](value)
}

// Service

func (a *Access) serviceSchema(serviceSkelName string) (*redised.SchemaService, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	service, ok := a.servicesBySkelName[serviceSkelName]
	return service, ok
}

func (a *Access) loadServices() {
	valuesByKey := a.Redis.LoadListAndSubscribe(a.Context, redised.FormatSchemaServicePrefix(), a.handleServiceEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		a.setServiceLocked(key, decodeService(value))
	}
}

func (a *Access) handleServiceEvent(event hubredis.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubredis.EventKindDelete {
		a.removeServiceLocked(event.Key)
		return
	}
	a.setServiceLocked(event.Key, decodeService(event.Value))
}

func (a *Access) setServiceLocked(key string, service *redised.SchemaService) {
	a.removeServiceLocked(key)
	a.serviceNamesByKey[key] = service.SkelName
	a.servicesBySkelName[service.SkelName] = service
}

func (a *Access) removeServiceLocked(key string) {
	if name, ok := a.serviceNamesByKey[key]; ok {
		delete(a.servicesBySkelName, name)
		delete(a.serviceNamesByKey, key)
	}
}

func decodeService(value string) *redised.SchemaService {
	return vcode.MustUnmarshalJsonS[*redised.SchemaService](value)
}

// Resource

func (a *Access) loadResources() {
	valuesByKey := a.Redis.LoadListAndSubscribe(a.Context, redised.FormatSchemaResourcePrefix(), a.handleResourceEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		a.setResourceLocked(key, decodeResource(value))
	}
}

func (a *Access) handleResourceEvent(event hubredis.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubredis.EventKindDelete {
		a.removeResourceLocked(event.Key)
		return
	}
	a.setResourceLocked(event.Key, decodeResource(event.Value))
}

func (a *Access) setResourceLocked(key string, resource *redised.SchemaResource) {
	a.removeResourceLocked(key)
	a.resourceNamesByKey[key] = resource.SkelName
	a.resourcesBySkelName[resource.SkelName] = resource
	a.watchResourceCheckServiceLocked(key, resource)
}

func (a *Access) removeResourceLocked(key string) {
	a.releaseResourceCheckServiceLocked(key)
	if name, ok := a.resourceNamesByKey[key]; ok {
		delete(a.resourcesBySkelName, name)
		delete(a.resourceNamesByKey, key)
	}
}

func (a *Access) watchResourceCheckServiceLocked(key string, resource *redised.SchemaResource) {
	if resource.CheckService == nil {
		return
	}
	a.checkServiceWatchersByResourceKey[key] = a.Epmgr.WatchRpc(resource.CheckService.SkelName)
}

func (a *Access) releaseResourceCheckServiceLocked(key string) {
	watcher := a.checkServiceWatchersByResourceKey[key]
	if watcher == nil {
		return
	}
	watcher.Release()
	delete(a.checkServiceWatchersByResourceKey, key)
}

func decodeResource(value string) *redised.SchemaResource {
	return vcode.MustUnmarshalJsonS[*redised.SchemaResource](value)
}
