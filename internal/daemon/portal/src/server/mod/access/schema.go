package access

import (
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

// Actor

func (a *Access) actorSchema(actorSkelName string) (*watched.SchemaActor, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	actor, ok := a.actorsBySkelName[actorSkelName]
	return actor, ok
}

func (a *Access) loadActors() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatSchemaActorPrefix(), a.handleActorEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		actor := decodeActor(value)
		a.setActorLocked(key, actor)
	}
	subscription.Start()
}

func (a *Access) handleActorEvent(event hubwatch.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubwatch.EventKindDelete {
		a.removeActorLocked(event.Key)
		return
	}
	a.setActorLocked(event.Key, decodeActor(event.Value))
}

func (a *Access) setActorLocked(key string, actor *watched.SchemaActor) {
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

func (a *Access) watchActorAuthServiceLocked(key string, actor *watched.SchemaActor) {
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

func (a *Access) watchActorPermServiceLocked(key string, actor *watched.SchemaActor) {
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

func decodeActor(value string) *watched.SchemaActor {
	return vcode.MustUnmarshalJsonS[*watched.SchemaActor](value)
}

// Service

func (a *Access) serviceSchema(serviceSkelName string) (*watched.SchemaService, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	service, ok := a.servicesBySkelName[serviceSkelName]
	return service, ok
}

func (a *Access) loadServices() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatSchemaServicePrefix(), a.handleServiceEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		a.setServiceLocked(key, decodeService(value))
	}
	subscription.Start()
}

func (a *Access) handleServiceEvent(event hubwatch.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubwatch.EventKindDelete {
		a.removeServiceLocked(event.Key)
		return
	}
	a.setServiceLocked(event.Key, decodeService(event.Value))
}

func (a *Access) setServiceLocked(key string, service *watched.SchemaService) {
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

func decodeService(value string) *watched.SchemaService {
	return vcode.MustUnmarshalJsonS[*watched.SchemaService](value)
}

// Resource

func (a *Access) loadResources() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatSchemaResourcePrefix(), a.handleResourceEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		a.setResourceLocked(key, decodeResource(value))
	}
	subscription.Start()
}

func (a *Access) handleResourceEvent(event hubwatch.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubwatch.EventKindDelete {
		a.removeResourceLocked(event.Key)
		return
	}
	a.setResourceLocked(event.Key, decodeResource(event.Value))
}

func (a *Access) setResourceLocked(key string, resource *watched.SchemaResource) {
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

func (a *Access) watchResourceCheckServiceLocked(key string, resource *watched.SchemaResource) {
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

func decodeResource(value string) *watched.SchemaResource {
	return vcode.MustUnmarshalJsonS[*watched.SchemaResource](value)
}
