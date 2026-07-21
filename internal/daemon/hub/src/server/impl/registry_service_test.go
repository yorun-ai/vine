package impl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/util/vslice"
)

type _RegistryServicePortalSiteRepo struct {
	entries []core.PortalSite
}

func (r *_RegistryServicePortalSiteRepo) ListEntries() []core.PortalSite {
	return r.entries
}

func (*_RegistryServicePortalSiteRepo) GetEntryById(int) (*core.PortalSite, bool) {
	return nil, false
}

func (*_RegistryServicePortalSiteRepo) GetEntryByName(string) (*core.PortalSite, bool) {
	return nil, false
}

func (*_RegistryServicePortalSiteRepo) SaveEntry(*core.PortalSite) {
}

func (*_RegistryServicePortalSiteRepo) RemoveEntry(int) bool {
	return false
}

type _RegistryServiceSchemaRepo struct {
	_AppConfigServiceSchemaRepo

	actorSchemas   []*skel.ActorSchema
	serviceSchemas []*skel.ServiceSchema
}

func (r *_RegistryServiceSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	return []core.DomainSchemaView{{
		DomainVersion: core.DomainSchemaVersion{
			Main: true,
			Schema: &skel.DomainSchema{
				Services: r.serviceSchemas,
			},
		},
		Actors: vslice.Collect(func(yield func(core.SchemaVersion[*skel.ActorSchema]) bool) {
			for _, actor := range r.actorSchemas {
				if !yield(core.SchemaVersion[*skel.ActorSchema]{
					Schema:     actor,
					SkelName:   actor.SkelName,
					SchemaHash: actor.Hash,
					Main:       true,
				}) {
					return
				}
			}
		}),
		Services: vslice.Collect(func(yield func(core.SchemaVersion[*skel.ServiceSchema]) bool) {
			for _, service := range r.serviceSchemas {
				if !yield(core.SchemaVersion[*skel.ServiceSchema]{
					Schema:     service,
					SkelName:   service.SkelName,
					SchemaHash: service.Hash,
					Main:       true,
				}) {
					return
				}
			}
		}),
	}}
}

func (r *_RegistryServiceSchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	return r.serviceSchemas
}

func testRegistrySyncer(redisServer *redisserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}

func TestRegistryServiceRefreshesPortalSiteRpcgwServices(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()

	service := &RegistryServiceServerImpl{
		PortalSiteRepo: &_RegistryServicePortalSiteRepo{
			entries: []core.PortalSite{
				{
					Id:            1,
					Name:          "demo.UserActor-client-rpc",
					Type:          core.PortalSiteTypeRPCGW,
					ActorSkelName: "demo.UserActor",
					ActorVia:      "client",
				},
				{
					Id:            2,
					Name:          "vine.hub.AdminActor-client-rpc",
					Type:          core.PortalSiteTypeRPCGW,
					ActorSkelName: "vine.hub.AdminActor",
					ActorVia:      "client",
					BuiltIn:       true,
				},
				{
					Id:      3,
					Name:    "demo.Web-web",
					Type:    core.PortalSiteTypeWEBGW,
					WebName: "demo.Web",
				},
			},
		},
		SchemaRepo: &_RegistryServiceSchemaRepo{
			serviceSchemas: []*skel.ServiceSchema{
				{
					SkelName: "demo.UserService",
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.UserActor"},
					},
				},
				{
					SkelName: "demo.AdminService",
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.AdminActor"},
					},
				},
			},
		},
		Syncer: testRegistrySyncer(redisServer),
	}

	service.refreshPortalSiteRpcgwServices()

	value, ok := redisServer.Get(redised.FormatPortalSiteKey("demo.UserActor-client-rpc"))
	require.True(t, ok)
	site := new(redised.PortalSite)
	require.NoError(t, json.Unmarshal([]byte(value), site))
	require.NotNil(t, site.RpcgwConfig)
	assert.Equal(t, []redised.PortalRpcgwService{{SkelName: "demo.UserService"}}, site.RpcgwConfig.Services)

	_, ok = redisServer.Get(redised.FormatPortalSiteKey("vine.hub.AdminActor-client-rpc"))
	assert.False(t, ok)

	value, ok = redisServer.Get(redised.FormatPortalSiteKey("demo.Web-web"))
	require.True(t, ok)
	webSite := new(redised.PortalSite)
	require.NoError(t, json.Unmarshal([]byte(value), webSite))
	assert.Nil(t, webSite.RpcgwConfig)
	assert.Equal(t, "demo.Web", webSite.WebgwConfig.WebName)
}

func TestRegistryServiceRefreshesSchemas(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()

	service := &RegistryServiceServerImpl{
		SchemaRepo: &_RegistryServiceSchemaRepo{
			actorSchemas: []*skel.ActorSchema{
				{
					SkelName: "demo.UserActor",
					Hash:     "actor-main",
					AuthService: &skel.ServiceSchema{
						SkelName: "demo.UserAuthService",
						Hash:     "auth-service-main",
					},
				},
			},
			serviceSchemas: []*skel.ServiceSchema{
				{
					SkelName: "demo.UserService",
					Hash:     "service-main",
					Methods: []*skel.MethodSchema{
						{SkelName: "Get", AuthMode: skel.AuthModeAuth},
					},
				},
			},
		},
		Syncer: testRegistrySyncer(redisServer),
	}

	service.refreshSchemas()

	value, ok := redisServer.Get(redised.FormatSchemaActorKey("demo.UserActor"))
	require.True(t, ok)
	actor := new(redised.SchemaActor)
	require.NoError(t, json.Unmarshal([]byte(value), actor))
	assert.Equal(t, "demo.UserActor", actor.SkelName)
	assert.Equal(t, "actor-main", actor.Hash)
	require.NotNil(t, actor.AuthService)
	assert.Equal(t, "demo.UserAuthService", actor.AuthService.SkelName)

	value, ok = redisServer.Get(redised.FormatSchemaServiceKey("demo.UserService"))
	require.True(t, ok)
	serviceSchema := new(redised.SchemaService)
	require.NoError(t, json.Unmarshal([]byte(value), serviceSchema))
	assert.Equal(t, "demo.UserService", serviceSchema.SkelName)
	assert.Equal(t, "service-main", serviceSchema.Hash)
	assert.Equal(t, skel.AuthModeAuth, serviceSchema.Methods[0].AuthMode)
}
