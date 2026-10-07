package access

import (
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

// Actor

func (a *Access) actorDescriptor(actorSkelName string) (*watched.DescriptorActor, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	actor, ok := a.actorsBySkelName[actorSkelName]
	return actor, ok
}

func (a *Access) loadActors() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatDescriptorActorPrefix(), a.handleActorEvent)

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

func (a *Access) setActorLocked(key string, actor *watched.DescriptorActor) {
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

func (a *Access) watchActorAuthServiceLocked(key string, actor *watched.DescriptorActor) {
	if actor.Auth == nil {
		return
	}
	a.authServiceWatchersByActorKey[key] = a.Epmgr.WatchRpc(actor.Auth.Service.SkelName)
}

func (a *Access) releaseActorAuthServiceLocked(key string) {
	watcher := a.authServiceWatchersByActorKey[key]
	if watcher == nil {
		return
	}
	watcher.Release()
	delete(a.authServiceWatchersByActorKey, key)
}

func (a *Access) watchActorPermServiceLocked(key string, actor *watched.DescriptorActor) {
	if actor.Permission == nil {
		return
	}
	a.permServiceWatchersByActorKey[key] = a.Epmgr.WatchRpc(actor.Permission.Service.SkelName)
}

func (a *Access) releaseActorPermServiceLocked(key string) {
	watcher := a.permServiceWatchersByActorKey[key]
	if watcher == nil {
		return
	}
	watcher.Release()
	delete(a.permServiceWatchersByActorKey, key)
}

func decodeActor(value string) *watched.DescriptorActor {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorActor](value)
}

// Service

func (a *Access) serviceDescriptor(serviceSkelName string) (*watched.DescriptorService, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	service, ok := a.servicesBySkelName[serviceSkelName]
	return service, ok
}

func (a *Access) loadServices() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatDescriptorServicePrefix(), a.handleServiceEvent)

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

func (a *Access) setServiceLocked(key string, service *watched.DescriptorService) {
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

func decodeService(value string) *watched.DescriptorService {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorService](value)
}

// Resource

func (a *Access) loadResources() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatDescriptorResourcePrefix(), a.handleResourceEvent)

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

func (a *Access) setResourceLocked(key string, resource *watched.DescriptorResource) {
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

func (a *Access) watchResourceCheckServiceLocked(key string, resource *watched.DescriptorResource) {
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

func decodeResource(value string) *watched.DescriptorResource {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorResource](value)
}

// Web

func (a *Access) webDescriptor(webSkelName string) (*watched.DescriptorWeb, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	web, ok := a.websBySkelName[webSkelName]
	return web, ok
}

func (a *Access) loadWebs() {
	valuesByKey, subscription := a.Watch.LoadListAndSubscribe(a.Context, watched.FormatDescriptorWebPrefix(), a.handleWebEvent)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for key, value := range valuesByKey {
		a.setWebLocked(key, decodeWeb(value))
	}
	subscription.Start()
}

func (a *Access) handleWebEvent(event hubwatch.Event) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if event.Kind == hubwatch.EventKindDelete {
		a.removeWebLocked(event.Key)
		return
	}
	a.setWebLocked(event.Key, decodeWeb(event.Value))
}

func (a *Access) setWebLocked(key string, web *watched.DescriptorWeb) {
	a.removeWebLocked(key)
	a.webNamesByKey[key] = web.SkelName
	a.websBySkelName[web.SkelName] = web
}

func (a *Access) removeWebLocked(key string) {
	if name, ok := a.webNamesByKey[key]; ok {
		delete(a.websBySkelName, name)
		delete(a.webNamesByKey, key)
	}
}

func decodeWeb(value string) *watched.DescriptorWeb {
	return vcode.MustUnmarshalJsonS[*watched.DescriptorWeb](value)
}
