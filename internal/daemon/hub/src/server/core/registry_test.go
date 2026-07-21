package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
)

type registryRepoSpy struct {
	calls []string

	appStatus        *AppStatus
	appStatusOK      bool
	keepAppStatusOK  bool
	keepRpcOK        map[string]bool
	keepWebOK        map[string]bool
	rpcRegistrations []*RpcServiceRegistration
	webRegistrations []*WebRegistration
}

type schemaRepoSpy struct {
	domainSchemas []*skel.DomainSchema
	saved         []string
	released      []string
}

func (s *schemaRepoSpy) SaveDomainSchemas(ownerName string, ownerId string, schemas []*skel.DomainSchema) {
	s.saved = append(s.saved, ownerName+":"+ownerId)
	s.domainSchemas = append([]*skel.DomainSchema{}, schemas...)
}

func (s *schemaRepoSpy) SaveDomainSchemasJSON(ownerName string, ownerId string, schemas []skel.JSON) {
	s.saved = append(s.saved, ownerName+":"+ownerId)
	s.domainSchemas = make([]*skel.DomainSchema, 0, len(schemas))
	for _, schemaJson := range schemas {
		s.domainSchemas = append(s.domainSchemas, vcode.MustUnmarshalJsonS[*skel.DomainSchema](string(schemaJson)))
	}
}

func (s *schemaRepoSpy) ReleaseDomainSchemas(ownerName string, ownerId string) {
	s.released = append(s.released, ownerName+":"+ownerId)
}

func (*schemaRepoSpy) ListDomainSchemaViews() []DomainSchemaView {
	return nil
}

func (*schemaRepoSpy) ListVineHubSchemaViews() []DomainSchemaView {
	return nil
}

func (*schemaRepoSpy) ListActorSchemaVersions() []SchemaVersion[*skel.ActorSchema] {
	return nil
}

func (*schemaRepoSpy) ListConfigSchemaVersions() []SchemaVersion[*skel.ConfigSchema] {
	return nil
}

func (*schemaRepoSpy) ListDataSchemaVersions() []SchemaVersion[*skel.DataSchema] {
	return nil
}

func (*schemaRepoSpy) ListEnumSchemaVersions() []SchemaVersion[*skel.EnumSchema] {
	return nil
}

func (*schemaRepoSpy) ListEventSchemaVersions() []SchemaVersion[*skel.EventSchema] {
	return nil
}

func (*schemaRepoSpy) ListResourceSchemaVersions() []SchemaVersion[*skel.ResourceSchema] {
	return nil
}

func (*schemaRepoSpy) ListServiceSchemaVersions() []SchemaVersion[*skel.ServiceSchema] {
	return nil
}

func (*schemaRepoSpy) ListTaskSchemaVersions() []SchemaVersion[*skel.TaskSchema] {
	return nil
}

func (*schemaRepoSpy) ListWebSchemaVersions() []SchemaVersion[*skel.WebSchema] {
	return nil
}

func (*schemaRepoSpy) ListAppConfigSchemas() []*skel.ConfigSchema {
	return nil
}

func (*schemaRepoSpy) ListActorSchemas() []*skel.ActorSchema {
	return nil
}

func (*schemaRepoSpy) ListEnumSchemas() []*skel.EnumSchema {
	return nil
}

func (*schemaRepoSpy) ListServiceSchemas() []*skel.ServiceSchema {
	return nil
}

func (*schemaRepoSpy) ListWebSchemas() []*skel.WebSchema {
	return nil
}

func (s *registryRepoSpy) SaveAppStatus(status *AppStatus) {
	s.calls = append(s.calls, "SaveAppStatus:"+status.InstanceId)
}

func (s *registryRepoSpy) ListAppStatuses() []*AppStatus {
	s.calls = append(s.calls, "ListAppStatuses")
	if s.appStatus == nil {
		return []*AppStatus{}
	}
	return []*AppStatus{s.appStatus}
}

func (s *registryRepoSpy) GetAppStatus(appName string, instanceId string) (*AppStatus, bool) {
	s.calls = append(s.calls, "GetAppStatus:"+appName+":"+instanceId)
	return s.appStatus, s.appStatusOK
}

func (s *registryRepoSpy) KeepAppStatus(appName string, instanceId string) bool {
	s.calls = append(s.calls, "KeepAppStatus:"+appName+":"+instanceId)
	return s.keepAppStatusOK
}

func (s *registryRepoSpy) RemoveAppStatus(appName string, instanceId string) {
	s.calls = append(s.calls, "RemoveAppStatus:"+appName+":"+instanceId)
}

func (*registryRepoSpy) PopExpiredAppLeases() []AppHeartbeat {
	return nil
}

func (s *registryRepoSpy) SaveRpcServiceRegistration(registration *RpcServiceRegistration) {
	s.calls = append(s.calls, "SaveRpcServiceRegistration:"+registration.ServiceName+":"+registration.AppName+":"+registration.AppInstanceId)
	s.rpcRegistrations = append(s.rpcRegistrations, registration)
}

func (s *registryRepoSpy) GetRpcServiceRegistration(serviceName string, appName string, instanceId string) (*RpcServiceRegistration, bool) {
	s.calls = append(s.calls, "GetRpcServiceRegistration:"+serviceName+":"+appName+":"+instanceId)
	return nil, false
}

func (s *registryRepoSpy) KeepRpcServiceRegistration(serviceName string, appName string, appInstanceId string) bool {
	s.calls = append(s.calls, "KeepRpcServiceRegistration:"+serviceName+":"+appName+":"+appInstanceId)
	if s.keepRpcOK == nil {
		return true
	}
	return s.keepRpcOK[serviceName]
}

func (s *registryRepoSpy) RemoveRpcServiceRegistration(serviceName string, appName string, appInstanceId string) {
	s.calls = append(s.calls, "RemoveRpcServiceRegistration:"+serviceName+":"+appName+":"+appInstanceId)
}

func (s *registryRepoSpy) SaveWebRegistration(registration *WebRegistration) {
	s.calls = append(s.calls, "SaveWebRegistration:"+registration.WebSkelName+":"+registration.AppName+":"+registration.AppInstanceId)
	s.webRegistrations = append(s.webRegistrations, registration)
}

func (s *registryRepoSpy) GetWebRegistration(name string, appName string, instanceId string) (*WebRegistration, bool) {
	s.calls = append(s.calls, "GetWebRegistration:"+name+":"+appName+":"+instanceId)
	return nil, false
}

func (s *registryRepoSpy) KeepWebRegistration(name string, appName string, appInstanceId string) bool {
	s.calls = append(s.calls, "KeepWebRegistration:"+name+":"+appName+":"+appInstanceId)
	if s.keepWebOK == nil {
		return true
	}
	return s.keepWebOK[name]
}

func (s *registryRepoSpy) RemoveWebRegistration(name string, appName string, appInstanceId string) {
	s.calls = append(s.calls, "RemoveWebRegistration:"+name+":"+appName+":"+appInstanceId)
}

func newRegistryCoreForTest(repo RegistryRepo, schemaRepo SchemaRepo) *RegistryCore {
	return &RegistryCore{
		RegistryRepo: repo,
		SchemaRepo:   schemaRepo,
	}
}

func TestRegistryCoreRegister(t *testing.T) {
	repo := &registryRepoSpy{}
	schemaRepo := &schemaRepoSpy{}
	core := newRegistryCoreForTest(repo, schemaRepo)

	core.Register(AppRegistration{
		InstanceId: "instance-1",
		Name:       "demo.app",
		Version:    "1.2.3",
		Endpoint:   "http://127.0.0.1:23001",
		ServiceHandlers: []ServiceHandlerRegistration{
			{ServiceSkelName: "svc.alpha", Endpoint: "http://127.0.0.1:23001/rpc/proxy/in"},
			{ServiceSkelName: "svc.beta", Endpoint: "http://127.0.0.1:23001/rpc/proxy/in"},
		},
		WebHandlers: []WebHandlerRegistration{
			{WebSkelName: "default@demo.app", Endpoint: "http://127.0.0.1:23001/web/proxy/in/instance-1/default@demo.app"},
		},
	})

	assert.Equal(t, []string{
		"SaveAppStatus:instance-1",
		"SaveRpcServiceRegistration:svc.alpha:demo.app:instance-1",
		"SaveRpcServiceRegistration:svc.beta:demo.app:instance-1",
		"SaveWebRegistration:default@demo.app:demo.app:instance-1",
	}, repo.calls)
	assert.Len(t, repo.rpcRegistrations, 2)
	assert.Equal(t, "http://127.0.0.1:23001/rpc/proxy/in", repo.rpcRegistrations[0].Endpoint)
	assert.Len(t, repo.webRegistrations, 1)
	assert.Equal(t, "http://127.0.0.1:23001/web/proxy/in/instance-1/default@demo.app", repo.webRegistrations[0].Endpoint)
	assert.Empty(t, schemaRepo.domainSchemas)
}

func TestRegistryCoreRegisterKeepsProvidedProxyEndpoints(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &schemaRepoSpy{})

	core.Register(AppRegistration{
		InstanceId:      "instance-1",
		Name:            "demo.app",
		Version:         "1.2.3",
		Endpoint:        "",
		ServiceHandlers: []ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha", Endpoint: "/rpc/proxy/in"}},
		WebHandlers:     []WebHandlerRegistration{{WebSkelName: "default@demo.app", Endpoint: "/web/proxy/in/instance-1/default@demo.app"}},
	})

	assert.Len(t, repo.rpcRegistrations, 1)
	assert.Equal(t, "/rpc/proxy/in", repo.rpcRegistrations[0].Endpoint)
	assert.Len(t, repo.webRegistrations, 1)
	assert.Equal(t, "/web/proxy/in/instance-1/default@demo.app", repo.webRegistrations[0].Endpoint)
}

func TestRegistryCoreRegisterSavesDomainSchemas(t *testing.T) {
	repo := &registryRepoSpy{}
	schemaRepo := &schemaRepoSpy{}
	core := newRegistryCoreForTest(repo, schemaRepo)
	domainSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "pkg-hash-1",
	}

	core.Register(AppRegistration{
		InstanceId:    "instance-1",
		Name:          "demo.app",
		Version:       "1.2.3",
		DomainSchemas: []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(domainSchema))},
	})

	assert.Len(t, schemaRepo.domainSchemas, 1)
	assert.Equal(t, domainSchema, schemaRepo.domainSchemas[0])
	assert.Equal(t, []string{"demo.app:instance-1"}, schemaRepo.saved)
	assert.Empty(t, schemaRepo.released)
}

func TestRegistryCoreRegisterSavesDomainSchemasWithoutModeBranch(t *testing.T) {
	repo := &registryRepoSpy{}
	schemaRepo := &schemaRepoSpy{}
	core := newRegistryCoreForTest(repo, schemaRepo)
	domainSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "pkg-hash-1",
	}

	core.Register(AppRegistration{
		InstanceId:    "instance-1",
		Name:          "demo.app",
		DomainSchemas: []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(domainSchema))},
	})

	assert.Empty(t, schemaRepo.released)
	assert.Equal(t, []string{"demo.app:instance-1"}, schemaRepo.saved)
	assert.Equal(t, []*skel.DomainSchema{domainSchema}, schemaRepo.domainSchemas)
}

func TestRegistryCoreUnregisterWithStatus(t *testing.T) {
	repo := &registryRepoSpy{
		appStatus: &AppStatus{
			InstanceId:      "instance-1",
			Name:            "demo.app",
			ServiceHandlers: []ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha"}, {ServiceSkelName: "svc.beta"}},
			WebHandlers:     []WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		},
		appStatusOK:     true,
		keepAppStatusOK: true,
	}
	schemaRepo := &schemaRepoSpy{}
	core := newRegistryCoreForTest(repo, schemaRepo)

	core.Unregister("demo.app", "instance-1")

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
		"RemoveRpcServiceRegistration:svc.alpha:demo.app:instance-1",
		"RemoveRpcServiceRegistration:svc.beta:demo.app:instance-1",
		"RemoveWebRegistration:default@demo.app:demo.app:instance-1",
		"RemoveAppStatus:demo.app:instance-1",
	}, repo.calls)
	assert.Equal(t, []string{"demo.app:instance-1"}, schemaRepo.released)
}

func TestRegistryCoreUnregisterWithoutStatus(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &schemaRepoSpy{})

	core.Unregister("demo.app", "instance-1")

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
	}, repo.calls)
}

func TestRegistryCoreHeartbeatWithStatus(t *testing.T) {
	repo := &registryRepoSpy{
		appStatus: &AppStatus{
			InstanceId:      "instance-1",
			Name:            "demo.app",
			ServiceHandlers: []ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha"}, {ServiceSkelName: "svc.beta"}},
			WebHandlers:     []WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		},
		appStatusOK:     true,
		keepAppStatusOK: true,
	}
	core := newRegistryCoreForTest(repo, &schemaRepoSpy{})

	registered := core.Heartbeat(AppHeartbeat{Name: "demo.app", InstanceId: "instance-1"})

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
		"KeepAppStatus:demo.app:instance-1",
		"KeepRpcServiceRegistration:svc.alpha:demo.app:instance-1",
		"KeepRpcServiceRegistration:svc.beta:demo.app:instance-1",
		"KeepWebRegistration:default@demo.app:demo.app:instance-1",
	}, repo.calls)
	assert.True(t, registered)
}

func TestRegistryCoreHeartbeatReturnsFalseWhenKeepFails(t *testing.T) {
	repo := &registryRepoSpy{
		appStatus: &AppStatus{
			InstanceId:      "instance-1",
			Name:            "demo.app",
			ServiceHandlers: []ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha"}},
		},
		appStatusOK:     true,
		keepAppStatusOK: true,
		keepRpcOK:       map[string]bool{"svc.alpha": false},
	}
	core := newRegistryCoreForTest(repo, &schemaRepoSpy{})

	registered := core.Heartbeat(AppHeartbeat{Name: "demo.app", InstanceId: "instance-1"})

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
		"KeepAppStatus:demo.app:instance-1",
		"KeepRpcServiceRegistration:svc.alpha:demo.app:instance-1",
	}, repo.calls)
	assert.False(t, registered)
}

func TestRegistryCoreHeartbeatWithoutStatus(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &schemaRepoSpy{})

	registered := core.Heartbeat(AppHeartbeat{Name: "demo.app", InstanceId: "instance-1"})

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
	}, repo.calls)
	assert.False(t, registered)
}
