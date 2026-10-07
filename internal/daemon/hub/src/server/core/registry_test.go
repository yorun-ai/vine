package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/daemon"
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

type descriptorRepoSpy struct {
	domainDescriptors []*skeldesc.Domain
	webs              []*skeldesc.Web
	saved             []string
	released          []string
}

func (s *descriptorRepoSpy) SaveDomainDescriptors(ownerName string, ownerId string, descriptors []*skeldesc.Domain) {
	s.saved = append(s.saved, ownerName+":"+ownerId)
	s.domainDescriptors = append([]*skeldesc.Domain{}, descriptors...)
}

func (s *descriptorRepoSpy) SaveDomainDescriptorsJSON(ownerName string, ownerId string, descriptors []skeltype.JSON) {
	s.saved = append(s.saved, ownerName+":"+ownerId)
	s.domainDescriptors = make([]*skeldesc.Domain, 0, len(descriptors))
	for _, descriptorJson := range descriptors {
		s.domainDescriptors = append(s.domainDescriptors, vcode.MustUnmarshalJsonS[*skeldesc.Domain](string(descriptorJson)))
	}
}

func (s *descriptorRepoSpy) ReleaseDomainDescriptors(ownerName string, ownerId string) {
	s.released = append(s.released, ownerName+":"+ownerId)
}

func (*descriptorRepoSpy) ListDomainDescriptorViews() []DomainDescriptorView {
	return nil
}

func (*descriptorRepoSpy) ListVineHubDescriptorViews() []DomainDescriptorView {
	return nil
}

func (*descriptorRepoSpy) ListActorDescriptorVersions() []DescriptorVersion[*skeldesc.Actor] {
	return nil
}

func (*descriptorRepoSpy) ListConfigDescriptorVersions() []DescriptorVersion[*skeldesc.Config] {
	return nil
}

func (*descriptorRepoSpy) ListDataDescriptorVersions() []DescriptorVersion[*skeldesc.Data] {
	return nil
}

func (*descriptorRepoSpy) ListEnumDescriptorVersions() []DescriptorVersion[*skeldesc.Enum] {
	return nil
}

func (*descriptorRepoSpy) ListEventDescriptorVersions() []DescriptorVersion[*skeldesc.Event] {
	return nil
}

func (*descriptorRepoSpy) ListResourceDescriptorVersions() []DescriptorVersion[*skeldesc.Resource] {
	return nil
}

func (*descriptorRepoSpy) ListServiceDescriptorVersions() []DescriptorVersion[*skeldesc.Service] {
	return nil
}

func (*descriptorRepoSpy) ListTaskDescriptorVersions() []DescriptorVersion[*skeldesc.Task] {
	return nil
}

func (*descriptorRepoSpy) ListWebDescriptorVersions() []DescriptorVersion[*skeldesc.Web] {
	return nil
}

func (*descriptorRepoSpy) ListAppConfigDescriptors() []*skeldesc.Config {
	return nil
}

func (*descriptorRepoSpy) ListActorDescriptors() []*skeldesc.Actor {
	return nil
}

func (*descriptorRepoSpy) ListEnumDescriptors() []*skeldesc.Enum {
	return nil
}

func (*descriptorRepoSpy) ListServiceDescriptors() []*skeldesc.Service {
	return nil
}

func (s *descriptorRepoSpy) ListWebDescriptors() []*skeldesc.Web {
	return s.webs
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

func newRegistryCoreForTest(repo RegistryRepo, descriptorRepo DescriptorRepo) *RegistryCore {
	return &RegistryCore{
		RegistryRepo:   repo,
		DescriptorRepo: descriptorRepo,
	}
}

func TestRegistryCoreRegister(t *testing.T) {
	repo := &registryRepoSpy{}
	descriptorRepo := &descriptorRepoSpy{}
	core := newRegistryCoreForTest(repo, descriptorRepo)

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
	assert.Equal(t, daemon.LinkIdentity, repo.rpcRegistrations[0].ServerIdentity)
	assert.Len(t, repo.webRegistrations, 1)
	assert.Equal(t, "http://127.0.0.1:23001/web/proxy/in/instance-1/default@demo.app", repo.webRegistrations[0].Endpoint)
	assert.Equal(t, daemon.LinkIdentity, repo.webRegistrations[0].ServerIdentity)
	assert.Empty(t, descriptorRepo.domainDescriptors)
}

func TestRegistryCoreRegisterKeepsProvidedProxyEndpoints(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})

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

func TestRegistryCoreRegisterSavesDomainDescriptors(t *testing.T) {
	repo := &registryRepoSpy{}
	descriptorRepo := &descriptorRepoSpy{}
	core := newRegistryCoreForTest(repo, descriptorRepo)
	domainDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "pkg-hash-1", Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}

	core.Register(AppRegistration{
		InstanceId:        "instance-1",
		Name:              "demo.app",
		DomainDescriptors: []*skeldesc.Domain{domainDescriptor},
	})

	assert.Empty(t, descriptorRepo.released)
	assert.Equal(t, []string{"demo.app:instance-1"}, descriptorRepo.saved)
	assert.Equal(t, []*skeldesc.Domain{domainDescriptor}, descriptorRepo.domainDescriptors)
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
	descriptorRepo := &descriptorRepoSpy{}
	core := newRegistryCoreForTest(repo, descriptorRepo)

	core.Unregister("demo.app", "instance-1")

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
		"RemoveRpcServiceRegistration:svc.alpha:demo.app:instance-1",
		"RemoveRpcServiceRegistration:svc.beta:demo.app:instance-1",
		"RemoveWebRegistration:default@demo.app:demo.app:instance-1",
		"RemoveAppStatus:demo.app:instance-1",
	}, repo.calls)
	assert.Equal(t, []string{"demo.app:instance-1"}, descriptorRepo.released)
}

func TestRegistryCoreUnregisterWithoutStatus(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})

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
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})

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
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})

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
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})

	registered := core.Heartbeat(AppHeartbeat{Name: "demo.app", InstanceId: "instance-1"})

	assert.Equal(t, []string{
		"GetAppStatus:demo.app:instance-1",
	}, repo.calls)
	assert.False(t, registered)
}

func TestRegistryPropagatesApiBoundaryFromDescriptor(t *testing.T) {
	repo := &registryRepoSpy{}
	core := newRegistryCoreForTest(repo, &descriptorRepoSpy{})
	core.Register(AppRegistration{
		InstanceId: "instance-1", Name: "demo.app", Version: "1.0.0",
		DomainDescriptors: []*skeldesc.Domain{{Name: "demo", Services: []*skeldesc.Service{{SkelName: "demo.ApiService", Api: true, AuthMode: skeldesc.AuthModeRequired}, {SkelName: "demo.BackendService", Pub: true, AuthMode: skeldesc.AuthModeRequired}, {SkelName: "demo.LegacyService", AuthMode: skeldesc.AuthModeRequired}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"}}},
		ServiceHandlers: []ServiceHandlerRegistration{
			{ServiceSkelName: "demo.ApiService"},
			{ServiceSkelName: "demo.BackendService"},
			{ServiceSkelName: "demo.LegacyService"},
		},
	})
	assert.Len(t, repo.rpcRegistrations, 3)
	assert.True(t, repo.rpcRegistrations[0].Api)
	assert.False(t, repo.rpcRegistrations[1].Api)
	assert.False(t, repo.rpcRegistrations[2].Api)
}

func (r *descriptorRepoSpy) GetWebDescriptor(skelName string) *skeldesc.Web {
	for _, descriptor := range r.webs {
		if descriptor.SkelName == skelName {
			return descriptor
		}
	}
	return nil
}

func (r *descriptorRepoSpy) ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data) {
	return r.ListAppConfigDescriptors(), r.ListEnumDescriptors(), nil
}
