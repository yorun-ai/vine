package access

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
)

type Access struct {
	app.BaseModule

	Context context.Context  `inject:""`
	Redis   *hubredis.Client `inject:""`
	Epmgr   *epmgr.Manager   `inject:""`

	mutex                             sync.RWMutex
	actorNamesByKey                   map[string]string
	serviceNamesByKey                 map[string]string
	resourceNamesByKey                map[string]string
	actorsBySkelName                  map[string]*redised.SchemaActor
	servicesBySkelName                map[string]*redised.SchemaService
	resourcesBySkelName               map[string]*redised.SchemaResource
	authServiceWatchersByActorKey     map[string]*epmgr.Watcher
	permServiceWatchersByActorKey     map[string]*epmgr.Watcher
	checkServiceWatchersByResourceKey map[string]*epmgr.Watcher
}

func (a *Access) DIInit() {
	a.actorNamesByKey = map[string]string{}
	a.serviceNamesByKey = map[string]string{}
	a.resourceNamesByKey = map[string]string{}
	a.actorsBySkelName = map[string]*redised.SchemaActor{}
	a.servicesBySkelName = map[string]*redised.SchemaService{}
	a.resourcesBySkelName = map[string]*redised.SchemaResource{}
	a.authServiceWatchersByActorKey = map[string]*epmgr.Watcher{}
	a.permServiceWatchersByActorKey = map[string]*epmgr.Watcher{}
	a.checkServiceWatchersByResourceKey = map[string]*epmgr.Watcher{}
	a.loadActors()
	a.loadServices()
	a.loadResources()
}

func (a *Access) AllowRpc(operation *RpcOperation) bool {
	operation.endpointManager = a.Epmgr

	actorSchema, ok := a.actorSchema(operation.ActorVia.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorSchema = actorSchema

	serviceSchema, ok := a.serviceSchema(operation.ServiceName)
	if !ok {
		operation.writeError(ex.ServiceUnavailable, "rpc service schema is not found: "+operation.ServiceName)
		return false
	}
	operation.serviceSchema = serviceSchema
	if !operation.loadMethodSchema() {
		return false
	}
	if !serviceSchema.HasAudience(operation.ActorVia.ActorSkelName, skel.ActorVia(operation.ActorVia.ActorVia)) {
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

	if operation.Request.Header.Get(headerAuthorization) == "" {
		operation.setActor(meta.NewAnonymousActor())
		return true
	}

	actorSchema, ok := a.actorSchema(operation.ActorVia.ActorSkelName)
	if !ok {
		operation.writeError(ex.ClientForbidden, "not allowed")
		return false
	}
	operation.actorSchema = actorSchema

	if !operation.actorSchema.AuthEnabled {
		operation.writeError(ex.ClientForbidden, "web actor auth is not enabled")
		return false
	}

	return operation.Auth()
}
