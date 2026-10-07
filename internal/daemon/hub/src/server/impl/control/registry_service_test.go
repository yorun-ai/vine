package control

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/util/vslice"
)

type _RegistryServicePortalSiteRepo struct {
	entries []*core.PortalSite
}

func (r *_RegistryServicePortalSiteRepo) List() []*core.PortalSite {
	return r.entries
}

func (*_RegistryServicePortalSiteRepo) GetById(int) (*core.PortalSite, bool) {
	return nil, false
}

func (*_RegistryServicePortalSiteRepo) GetByName(string) (*core.PortalSite, bool) {
	return nil, false
}

func (*_RegistryServicePortalSiteRepo) Save(*core.PortalSite) {
}

func (*_RegistryServicePortalSiteRepo) Remove(int) bool {
	return false
}

type _RegistryServiceDescriptorRepo struct {
	actorDescriptors   []*skeldesc.Actor
	serviceDescriptors []*skeldesc.Service
}

func (*_RegistryServiceDescriptorRepo) SaveDomainDescriptors(string, string, []*skeldesc.Domain) {
}

func (*_RegistryServiceDescriptorRepo) SaveDomainDescriptorsJSON(string, string, []skeltype.JSON) {
}

func (*_RegistryServiceDescriptorRepo) ReleaseDomainDescriptors(string, string) {}

func (r *_RegistryServiceDescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	return []core.DomainDescriptorView{{
		DomainVersion: core.DomainDescriptorVersion{
			Main: true,
			Descriptor: &skeldesc.Domain{
				Services: r.serviceDescriptors, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			},
		},
		Actors: vslice.Collect(func(yield func(core.DescriptorVersion[*skeldesc.Actor]) bool) {
			for _, actor := range r.actorDescriptors {
				if !yield(core.DescriptorVersion[*skeldesc.Actor]{
					Descriptor:     actor,
					SkelName:       actor.SkelName,
					DescriptorHash: actor.Hash,
					Main:           true,
				}) {
					return
				}
			}
		}),
		Services: vslice.Collect(func(yield func(core.DescriptorVersion[*skeldesc.Service]) bool) {
			for _, service := range r.serviceDescriptors {
				if !yield(core.DescriptorVersion[*skeldesc.Service]{
					Descriptor:     service,
					SkelName:       service.SkelName,
					DescriptorHash: service.Hash,
					Main:           true,
				}) {
					return
				}
			}
		}),
	}}
}

func (*_RegistryServiceDescriptorRepo) ListVineHubDescriptorViews() []core.DomainDescriptorView {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListActorDescriptorVersions() []core.DescriptorVersion[*skeldesc.Actor] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListConfigDescriptorVersions() []core.DescriptorVersion[*skeldesc.Config] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListDataDescriptorVersions() []core.DescriptorVersion[*skeldesc.Data] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListEnumDescriptorVersions() []core.DescriptorVersion[*skeldesc.Enum] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListEventDescriptorVersions() []core.DescriptorVersion[*skeldesc.Event] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListResourceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Resource] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListServiceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Service] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListTaskDescriptorVersions() []core.DescriptorVersion[*skeldesc.Task] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListWebDescriptorVersions() []core.DescriptorVersion[*skeldesc.Web] {
	return nil
}

func (*_RegistryServiceDescriptorRepo) ListAppConfigDescriptors() []*skeldesc.Config {
	return nil
}

func (r *_RegistryServiceDescriptorRepo) ListActorDescriptors() []*skeldesc.Actor {
	return r.actorDescriptors
}

func (*_RegistryServiceDescriptorRepo) ListEnumDescriptors() []*skeldesc.Enum {
	return nil
}

func (r *_RegistryServiceDescriptorRepo) ListServiceDescriptors() []*skeldesc.Service {
	return r.serviceDescriptors
}

func (*_RegistryServiceDescriptorRepo) ListWebDescriptors() []*skeldesc.Web {
	return nil
}

func testRegistrySyncer(watchServer *watchserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{WatchServer: watchServer}
	target.DIInit()
	return target
}

func TestRegistryServiceRefreshesPortalSiteRpcgwServices(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()

	siteRepo := &_RegistryServicePortalSiteRepo{
		entries: []*core.PortalSite{
			{
				Id:            1,
				Name:          "demo.UserActor-client-rpc",
				Type:          core.PortalSiteTypeRPCGW,
				ActorSkelName: "demo.UserActor",
				ActorVia:      "client",
				RpcgwServices: []string{"demo.UserService"},
				Enabled:       true,
			},
			{
				Id:            2,
				Name:          "vine.hub.admin.AdminActor-client-rpc",
				Type:          core.PortalSiteTypeRPCGW,
				ActorSkelName: "vine.hub.admin.AdminActor",
				ActorVia:      "client",
			},
			{
				Id:      3,
				Name:    "demo.Web-web",
				Type:    core.PortalSiteTypeWEBGW,
				WebName: "demo.Web",
				Enabled: true,
			},
		},
	}
	descriptorRepo := &_RegistryServiceDescriptorRepo{
		serviceDescriptors: []*skeldesc.Service{
			{
				SkelName: "demo.UserService",
				Audiences: []*skeldesc.ActorAudience{
					{SkelName: "demo.UserActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
			{
				SkelName: "demo.AdminService",
				Audiences: []*skeldesc.ActorAudience{
					{SkelName: "demo.AdminActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		},
	}
	service := &RegistryServiceServerImpl{
		PortalSiteCore: &core.PortalSiteCore{PortalSiteRepo: siteRepo, DescriptorRepo: descriptorRepo},
		DescriptorRepo: descriptorRepo,
		Syncer:         testRegistrySyncer(watchServer),
	}

	service.refreshPortalSiteRpcgwServices()

	value, ok := watchServer.Get(watched.FormatPortalSiteKey("demo.UserActor-client-rpc"))
	require.True(t, ok)
	site := new(watched.PortalSite)
	require.NoError(t, json.Unmarshal([]byte(value), site))
	require.NotNil(t, site.RpcgwConfig)
	assert.Equal(t, []watched.PortalRpcgwService{{SkelName: "demo.UserService"}}, site.RpcgwConfig.Services)

	_, ok = watchServer.Get(watched.FormatPortalSiteKey("vine.hub.admin.AdminActor-client-rpc"))
	assert.False(t, ok)

	value, ok = watchServer.Get(watched.FormatPortalSiteKey("demo.Web-web"))
	require.True(t, ok)
	webSite := new(watched.PortalSite)
	require.NoError(t, json.Unmarshal([]byte(value), webSite))
	assert.Nil(t, webSite.RpcgwConfig)
	assert.Equal(t, "demo.Web", webSite.WebgwConfig.WebName)
}

func TestRegistryServiceRefreshesDescriptors(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()

	service := &RegistryServiceServerImpl{
		DescriptorRepo: &_RegistryServiceDescriptorRepo{
			actorDescriptors: []*skeldesc.Actor{
				{
					SkelName: "demo.UserActor",
					Hash:     "actor-main", Auth: &skeldesc.ActorAuth{Service: &skeldesc.Service{
						SkelName: "demo.UserAuthService",
						Hash:     "auth-service-main", AuthMode: skeldesc.AuthModeRequired,
					}},
				},
			},
			serviceDescriptors: []*skeldesc.Service{
				{
					SkelName: "demo.UserService", Api: true,
					Hash: "service-main",
					Methods: []*skeldesc.Method{
						{SkelName: "Get", AuthMode: skeldesc.AuthModeRequired, Name: "Get", EffectiveAuthMode: skeldesc.AuthModeRequired},
					}, AuthMode: skeldesc.AuthModeRequired,
				},
			},
		},
		Syncer: testRegistrySyncer(watchServer),
	}

	service.refreshDescriptors()

	value, ok := watchServer.Get(watched.FormatDescriptorActorKey("demo.UserActor"))
	require.True(t, ok)
	actor := new(watched.DescriptorActor)
	require.NoError(t, json.Unmarshal([]byte(value), actor))
	assert.Equal(t, "demo.UserActor", actor.SkelName)
	assert.Equal(t, "actor-main", actor.Hash)
	require.NotNil(t, actor.Auth.Service)
	assert.Equal(t, "demo.UserAuthService", actor.Auth.Service.SkelName)

	value, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.UserService"))
	require.True(t, ok)
	serviceDescriptor := new(watched.DescriptorService)
	require.NoError(t, json.Unmarshal([]byte(value), serviceDescriptor))
	assert.Equal(t, "demo.UserService", serviceDescriptor.SkelName)
	assert.Equal(t, "service-main", serviceDescriptor.Hash)
	assert.Equal(t, skeldesc.AuthModeRequired, serviceDescriptor.Methods[0].AuthMode)
}

func (r *_RegistryServiceDescriptorRepo) GetWebDescriptor(skelName string) *skeldesc.Web {
	for _, descriptor := range r.ListWebDescriptors() {
		if descriptor.SkelName == skelName {
			return descriptor
		}
	}
	return nil
}

func (r *_RegistryServiceDescriptorRepo) ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data) {
	return r.ListAppConfigDescriptors(), r.ListEnumDescriptors(), nil
}
