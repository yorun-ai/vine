package access

import (
	"context"
	"sync"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/mtls"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
)

type Access struct {
	app.BaseModule

	Context  context.Context       `inject:""`
	Watch    hubapiwatch.ClientOps `inject:""`
	Epmgr    *epmgr.Manager        `inject:""`
	Identity *mtls.Identity        `inject:""`

	mutex                             sync.RWMutex
	actorNamesByKey                   map[string]string
	serviceNamesByKey                 map[string]string
	webNamesByKey                     map[string]string
	resourceNamesByKey                map[string]string
	actorsBySkelName                  map[string]*watched.DescriptorActor
	servicesBySkelName                map[string]*watched.DescriptorService
	websBySkelName                    map[string]*watched.DescriptorWeb
	resourcesBySkelName               map[string]*watched.DescriptorResource
	authServiceWatchersByActorKey     map[string]*epmgr.Watcher
	permServiceWatchersByActorKey     map[string]*epmgr.Watcher
	checkServiceWatchersByResourceKey map[string]*epmgr.Watcher
}

func (a *Access) DIInit() {
	a.actorNamesByKey = map[string]string{}
	a.serviceNamesByKey = map[string]string{}
	a.webNamesByKey = map[string]string{}
	a.resourceNamesByKey = map[string]string{}
	a.actorsBySkelName = map[string]*watched.DescriptorActor{}
	a.servicesBySkelName = map[string]*watched.DescriptorService{}
	a.websBySkelName = map[string]*watched.DescriptorWeb{}
	a.resourcesBySkelName = map[string]*watched.DescriptorResource{}
	a.authServiceWatchersByActorKey = map[string]*epmgr.Watcher{}
	a.permServiceWatchersByActorKey = map[string]*epmgr.Watcher{}
	a.checkServiceWatchersByResourceKey = map[string]*epmgr.Watcher{}
	a.loadActors()
	a.loadServices()
	a.loadWebs()
	a.loadResources()
}

func (a *Access) AllowRpc(operation *RpcOperation) bool {
	operation.endpointManager = a.Epmgr
	operation.identity = a.Identity
	if !operation.readRequestBody() {
		return false
	}

	actorDescriptor, ok := a.actorDescriptor(operation.ActorVia.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorDescriptor = actorDescriptor

	serviceDescriptor, ok := a.serviceDescriptor(operation.ServiceName)
	if !ok {
		operation.writeError(ex.ServiceUnavailable, "rpc service descriptor is not found: "+operation.ServiceName)
		return false
	}
	operation.serviceDescriptor = serviceDescriptor
	if !operation.loadMethodDescriptor() {
		return false
	}
	if !serviceDescriptor.HasAudience(operation.ActorVia.ActorSkelName, skeldesc.ActorViaKind(operation.ActorVia.ActorVia)) {
		operation.writeError(ex.ClientForbidden, "rpc service does not allow actor via")
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
		descriptor, ok := a.webDescriptor(operation.WebName)
		if !ok {
			operation.writeError(ex.ServiceUnavailable, "web descriptor is not found: "+operation.WebName)
			return false
		}
		mode = descriptor.AuthMode
	}
	actorDescriptor, ok := a.actorDescriptor(operation.ActorVia.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorDescriptor = actorDescriptor
	return operation.authenticate(mode, operation.writeError, operation.setActor)
}
