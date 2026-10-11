package access

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
)

type Access struct {
	app.BaseModule

	Context  context.Context `inject:""`
	Desc     *desc.Manager   `inject:""`
	Epmgr    *epmgr.Manager  `inject:""`
	Identity *mtls.Identity  `inject:""`

	mutex          sync.Mutex
	serviceWatches map[string]*_ServiceWatch
	watchCancel    context.CancelFunc
	stopped        bool
}

type _ServiceWatch struct {
	name    string
	watcher *epmgr.Watcher
}

func (a *Access) DIInit() {
	a.serviceWatches = map[string]*_ServiceWatch{}
	ctx, cancel := context.WithCancel(a.Context)
	a.watchCancel = cancel
	a.Desc.ObserveAdmission(ctx, a.handleAdmission)
}

func (a *Access) AllowRpc(operation *RpcOperation) bool {
	operation.endpointManager = a.Epmgr
	operation.identity = a.Identity
	if !operation.readRequestBody() {
		return false
	}

	actorDescriptor, ok := a.Desc.Actor(operation.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorDescriptor = actorDescriptor

	serviceDescriptor, ok := a.Desc.Service(operation.ServiceName)
	if !ok {
		operation.writeError(ex.ServiceUnavailable, "rpc service descriptor is not found: "+operation.ServiceName)
		return false
	}
	operation.serviceDescriptor = serviceDescriptor
	if !operation.loadMethodDescriptor() {
		return false
	}
	if !skel.HasActorAudience(serviceDescriptor.Audiences, operation.ActorSkelName) {
		operation.writeError(ex.ClientForbidden, "rpc service does not allow actor")
		return false
	}

	if !operation.Auth() {
		return false
	}
	return operation.Check()
}

func (a *Access) AuthWeb(operation *WebOperation) bool {
	operation.endpointManager = a.Epmgr
	operation.identity = a.Identity

	mode := defaultAuthMode
	if operation.WebName != "" {
		descriptor, ok := a.Desc.Web(operation.WebName)
		if !ok {
			operation.writeError(ex.ServiceUnavailable, "web descriptor is not found: "+operation.WebName)
			return false
		}
		mode = descriptor.AuthMode
	}
	actorDescriptor, ok := a.Desc.Actor(operation.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorDescriptor = actorDescriptor
	return operation.authenticate(mode, operation.writeError, operation.setActor)
}

func (a *Access) handleAdmission(snapshot desc.AdmissionSnapshot) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.stopped {
		return
	}
	desired := map[string]string{}
	for name, actor := range snapshot.Actors {
		if actor.Auth != nil {
			desired["auth:"+name] = actor.Auth.Service.SkelName
		}
		if actor.Permission != nil {
			desired["permission:"+name] = actor.Permission.Service.SkelName
		}
	}
	for name, resource := range snapshot.Resources {
		if resource.CheckService != nil {
			desired["check:"+name] = resource.CheckService.SkelName
		}
	}
	for key, watch := range a.serviceWatches {
		if desired[key] != watch.name {
			watch.watcher.Release()
			delete(a.serviceWatches, key)
		}
	}
	for key, name := range desired {
		if a.serviceWatches[key] == nil {
			a.serviceWatches[key] = new(_ServiceWatch{name: name, watcher: a.Epmgr.WatchRpc(name)})
		}
	}
}

// AfterAppStop releases admission observers and endpoint subscriptions.
func (a *Access) AfterAppStop() {
	a.watchCancel()
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.stopped = true
	for _, watch := range a.serviceWatches {
		watch.watcher.Release()
	}
	clear(a.serviceWatches)
}
