package control

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
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

type _RegistryServiceSchemaRepo struct {
	actorSchemas   []*skel.ActorSchema
	serviceSchemas []*skel.ServiceSchema
}

func (*_RegistryServiceSchemaRepo) SaveDomainSchemas(string, string, []*skel.DomainSchema) {
}

func (*_RegistryServiceSchemaRepo) SaveDomainSchemasJSON(string, string, []skel.JSON) {
}

func (*_RegistryServiceSchemaRepo) ReleaseDomainSchemas(string, string) {}

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

func (*_RegistryServiceSchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	return nil
}

func (*_RegistryServiceSchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	return nil
}

func (r *_RegistryServiceSchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	return r.actorSchemas
}

func (*_RegistryServiceSchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	return nil
}

func (r *_RegistryServiceSchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	return r.serviceSchemas
}

func (*_RegistryServiceSchemaRepo) ListWebSchemas() []*skel.WebSchema {
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
				BuiltIn:       true,
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
	schemaRepo := &_RegistryServiceSchemaRepo{
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
	}
	service := &RegistryServiceServerImpl{
		PortalSiteCore: &core.PortalSiteCore{PortalSiteRepo: siteRepo, SchemaRepo: schemaRepo},
		SchemaRepo:     schemaRepo,
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

func TestRegistryServiceRefreshesSchemas(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()

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
		Syncer: testRegistrySyncer(watchServer),
	}

	service.refreshSchemas()

	value, ok := watchServer.Get(watched.FormatSchemaActorKey("demo.UserActor"))
	require.True(t, ok)
	actor := new(watched.SchemaActor)
	require.NoError(t, json.Unmarshal([]byte(value), actor))
	assert.Equal(t, "demo.UserActor", actor.SkelName)
	assert.Equal(t, "actor-main", actor.Hash)
	require.NotNil(t, actor.AuthService)
	assert.Equal(t, "demo.UserAuthService", actor.AuthService.SkelName)

	value, ok = watchServer.Get(watched.FormatSchemaServiceKey("demo.UserService"))
	require.True(t, ok)
	serviceSchema := new(watched.SchemaService)
	require.NoError(t, json.Unmarshal([]byte(value), serviceSchema))
	assert.Equal(t, "demo.UserService", serviceSchema.SkelName)
	assert.Equal(t, "service-main", serviceSchema.Hash)
	assert.Equal(t, skel.AuthModeAuth, serviceSchema.Methods[0].AuthMode)
}

func (r *_RegistryServiceSchemaRepo) GetWebSchema(skelName string) *skel.WebSchema {
	for _, schema := range r.ListWebSchemas() {
		if schema.SkelName == skelName {
			return schema
		}
	}
	return nil
}
