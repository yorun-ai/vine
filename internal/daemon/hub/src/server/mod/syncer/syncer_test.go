package syncer

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func testSyncer(redisServer *redisserver.Server) *Syncer {
	target := &Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}

func TestSyncerSyncSchemasWritesMainActorAndServiceSchemas(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	target.SyncSchemas([]core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{
			{
				Schema: &skel.ActorSchema{
					SkelName: "demo.user.UserActor",
					Hash:     "actor-main",
					AuthCredential: &skel.DataSchema{
						SkelName: "demo.user.UserCredential",
					},
					AuthInfo: &skel.DataSchema{
						SkelName: "demo.user.UserInfo",
					},
					AuthService: &skel.ServiceSchema{
						SkelName: "demo.user.UserAuthService",
						Hash:     "auth-service-main",
					},
				},
				SkelName:   "demo.user.UserActor",
				SchemaHash: "actor-main",
				Main:       true,
			},
			{
				Schema: &skel.ActorSchema{
					SkelName:    "demo.user.OldActor",
					AuthService: &skel.ServiceSchema{SkelName: "demo.user.OldAuthService"},
				},
				SkelName:   "demo.user.OldActor",
				SchemaHash: "actor-old",
				Main:       false,
			},
			{
				Schema: &skel.ActorSchema{
					SkelName: "demo.user.NoAuthActor",
					Hash:     "actor-no-auth",
				},
				SkelName:   "demo.user.NoAuthActor",
				SchemaHash: "actor-no-auth",
				Main:       true,
			},
		},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{
			{
				Schema: &skel.ServiceSchema{
					SkelName: "demo.user.UserService",
					Hash:     "service-main",
					AuthMode: skel.AuthModeAuth,
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.user.UserActor", Via: skel.ActorViaClient},
					},
					Methods: []*skel.MethodSchema{
						{SkelName: "Get", AuthMode: skel.AuthModeNoAuth},
						{SkelName: "Update", AuthMode: skel.AuthModeAuth},
					},
				},
				SkelName:   "demo.user.UserService",
				SchemaHash: "service-main",
				Main:       true,
			},
			{
				Schema: &skel.ServiceSchema{
					SkelName: "demo.user.OldService",
					Hash:     "service-old",
				},
				SkelName:   "demo.user.OldService",
				SchemaHash: "service-old",
				Main:       false,
			},
		},
	}})

	value, ok := redisServer.Get(redised.FormatSchemaActorKey("demo.user.UserActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
		"name": "",
		"skelName": "demo.user.UserActor",
		"hash": "actor-main",
		"authEnabled": false,
		"permEnabled": false,
		"vias": null,
		"authCredential": {
			"name": "",
			"skelName": "demo.user.UserCredential",
			"hash": ""
		},
		"authInfo": {
			"name": "",
			"skelName": "demo.user.UserInfo",
			"hash": ""
		},
		"authService": {
			"name": "",
			"skelName": "demo.user.UserAuthService",
			"hash": "auth-service-main",
			"pub": false,
			"authMode": "",
			"methods": null
		}
	}`, value)
	value, ok = redisServer.Get(redised.FormatSchemaActorKey("demo.user.NoAuthActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
		"name": "",
		"skelName": "demo.user.NoAuthActor",
		"hash": "actor-no-auth",
		"authEnabled": false,
		"permEnabled": false,
		"vias": null
	}`, value)
	_, ok = redisServer.Get(redised.FormatSchemaActorKey("demo.user.OldActor"))
	assert.False(t, ok)

	value, ok = redisServer.Get(redised.FormatSchemaServiceKey("demo.user.UserService"))
	require.True(t, ok)
	assert.JSONEq(t, `{
		"name": "",
		"skelName": "demo.user.UserService",
		"hash": "service-main",
		"pub": false,
		"authMode": "auth",
		"audiences": [
			{"name": "", "skelName": "demo.user.UserActor", "via": "client"}
		],
		"methods": [
			{"name": "", "skelName": "Get", "hash": "", "authMode": "noauth"},
			{"name": "", "skelName": "Update", "hash": "", "authMode": "auth"}
		]
	}`, value)
	_, ok = redisServer.Get(redised.FormatSchemaServiceKey("demo.user.OldService"))
	assert.False(t, ok)
}

func TestSyncerWriteSchemasWritesMainActorAndServiceSchemas(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	target.WriteSchemas([]core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{{
			Schema: &skel.ActorSchema{
				SkelName: "vine.hub.AdminActor",
				Hash:     "admin-actor-main",
			},
			SkelName:   "vine.hub.AdminActor",
			SchemaHash: "admin-actor-main",
			Main:       true,
		}},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{{
			Schema: &skel.ServiceSchema{
				SkelName: "vine.hub.SkeletonService",
				Hash:     "skeleton-service-main",
				AuthMode: skel.AuthModeNoAuth,
			},
			SkelName:   "vine.hub.SkeletonService",
			SchemaHash: "skeleton-service-main",
			Main:       true,
		}},
	}})

	value, ok := redisServer.Get(redised.FormatSchemaActorKey("vine.hub.AdminActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
		"name": "",
		"skelName": "vine.hub.AdminActor",
		"hash": "admin-actor-main",
		"authEnabled": false,
		"permEnabled": false,
		"vias": null
	}`, value)

	value, ok = redisServer.Get(redised.FormatSchemaServiceKey("vine.hub.SkeletonService"))
	require.True(t, ok)
	assert.JSONEq(t, `{
		"name": "",
		"skelName": "vine.hub.SkeletonService",
		"hash": "skeleton-service-main",
		"pub": false,
		"authMode": "noauth",
		"methods": null
	}`, value)
}

func TestSyncerSyncSchemasDoesNotDeleteVineHubSchemas(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	target.WriteSchemas([]core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{{
			Schema:     &skel.ActorSchema{SkelName: "vine.hub.AdminActor", Hash: "admin-actor-main"},
			SkelName:   "vine.hub.AdminActor",
			SchemaHash: "admin-actor-main",
			Main:       true,
		}},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{{
			Schema:     &skel.ServiceSchema{SkelName: "vine.hub.SkeletonService", Hash: "skeleton-service-main"},
			SkelName:   "vine.hub.SkeletonService",
			SchemaHash: "skeleton-service-main",
			Main:       true,
		}},
	}})

	target.SyncSchemas([]core.DomainSchemaView{})

	_, ok := redisServer.Get(redised.FormatSchemaActorKey("vine.hub.AdminActor"))
	assert.True(t, ok)
	_, ok = redisServer.Get(redised.FormatSchemaServiceKey("vine.hub.SkeletonService"))
	assert.True(t, ok)
}

func TestSyncerSyncSchemasRemovesStaleSchemas(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	target.SyncSchemas([]core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{{
			Schema: &skel.ActorSchema{
				SkelName:    "demo.user.UserActor",
				AuthService: &skel.ServiceSchema{SkelName: "demo.user.UserAuthService"},
			},
			SkelName:   "demo.user.UserActor",
			SchemaHash: "actor-main",
			Main:       true,
		}},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{{
			Schema: &skel.ServiceSchema{
				SkelName: "demo.user.UserService",
			},
			SkelName:   "demo.user.UserService",
			SchemaHash: "service-main",
			Main:       true,
		}},
	}})
	_, ok := redisServer.Get(redised.FormatSchemaActorKey("demo.user.UserActor"))
	require.True(t, ok)
	_, ok = redisServer.Get(redised.FormatSchemaServiceKey("demo.user.UserService"))
	require.True(t, ok)

	target.SyncSchemas([]core.DomainSchemaView{})

	_, ok = redisServer.Get(redised.FormatSchemaActorKey("demo.user.UserActor"))
	assert.False(t, ok)
	_, ok = redisServer.Get(redised.FormatSchemaServiceKey("demo.user.UserService"))
	assert.False(t, ok)
}

func TestSyncerSyncSchemasOnlyWritesChangedHashes(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	view := []core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{{
			Schema: &skel.ActorSchema{
				SkelName: "demo.user.UserActor",
				Hash:     "actor-main",
			},
			SkelName:   "demo.user.UserActor",
			SchemaHash: "actor-main",
			Main:       true,
		}},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{{
			Schema: &skel.ServiceSchema{
				SkelName: "demo.user.UserService",
				Hash:     "service-main",
			},
			SkelName:   "demo.user.UserService",
			SchemaHash: "service-main",
			Main:       true,
		}},
	}}

	baseRevision := testRedisRevision(t, redisServer)
	target.SyncSchemas(view)
	firstRevision := testRedisRevision(t, redisServer)
	assert.Equal(t, baseRevision+1, firstRevision)

	target.SyncSchemas(view)
	secondRevision := testRedisRevision(t, redisServer)
	assert.Equal(t, firstRevision, secondRevision)

	view[0].Services[0].Schema.Hash = "service-next"
	view[0].Services[0].SchemaHash = "service-next"
	target.SyncSchemas(view)
	thirdRevision := testRedisRevision(t, redisServer)
	assert.Equal(t, secondRevision+1, thirdRevision)
}

func testRedisRevision(t *testing.T, redisServer *redisserver.Server) uint64 {
	t.Helper()
	value, ok := redisServer.Get(hubredis.RevisionKey)
	require.True(t, ok)
	revision, err := strconv.ParseUint(value, 10, 64)
	require.NoError(t, err)
	return revision
}
