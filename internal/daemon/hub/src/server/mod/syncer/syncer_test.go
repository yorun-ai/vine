package syncer

import (
	"encoding/json/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func testSyncer(watchServer *watchserver.Server) *Syncer {
	target := &Syncer{WatchServer: watchServer}
	target.DIInit()
	return target
}

func TestSyncerSyncDescriptorsWritesMainActorAndServiceDescriptors(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	target.SyncDescriptors([]core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{
			{
				Descriptor: &skeldesc.Actor{
					SkelName: "demo.user.UserActor",
					Hash:     "actor-main", Auth: &skeldesc.ActorAuth{Credential: &skeldesc.Data{
						SkelName: "demo.user.UserCredential",
					}, Info: &skeldesc.Data{
						SkelName: "demo.user.UserInfo",
					}, Service: &skeldesc.Service{
						SkelName: "demo.user.UserAuthService",
						Hash:     "auth-service-main", AuthMode: skeldesc.AuthModeRequired,
					}},
				},
				SkelName:       "demo.user.UserActor",
				DescriptorHash: "actor-main",
				Main:           true,
			},
			{
				Descriptor: &skeldesc.Actor{
					SkelName: "demo.user.OldActor", Auth: &skeldesc.ActorAuth{Service: &skeldesc.Service{SkelName: "demo.user.OldAuthService", AuthMode: skeldesc.AuthModeRequired}},
				},
				SkelName:       "demo.user.OldActor",
				DescriptorHash: "actor-old",
				Main:           false,
			},
			{
				Descriptor: &skeldesc.Actor{
					SkelName: "demo.user.NoAuthActor",
					Hash:     "actor-no-auth",
				},
				SkelName:       "demo.user.NoAuthActor",
				DescriptorHash: "actor-no-auth",
				Main:           true,
			},
		},
		Services: []core.DescriptorVersion[*skeldesc.Service]{
			{
				Descriptor: &skeldesc.Service{
					SkelName: "demo.user.UserService", Api: true,
					Hash:     "service-main",
					AuthMode: skeldesc.AuthModeRequired,
					Audiences: []*skeldesc.ActorAudience{
						{SkelName: "demo.user.UserActor", Via: skeldesc.ActorViaClient},
					},
					Methods: []*skeldesc.Method{
						{SkelName: "Get", AuthMode: skeldesc.AuthModeOptional, Name: "Get", EffectiveAuthMode: skeldesc.AuthModeOptional},
						{SkelName: "Update", AuthMode: skeldesc.AuthModeRequired, Name: "Update", EffectiveAuthMode: skeldesc.AuthModeRequired},
					},
				},
				SkelName:       "demo.user.UserService",
				DescriptorHash: "service-main",
				Main:           true,
			},
			{
				Descriptor: &skeldesc.Service{
					SkelName: "demo.user.OldService",
					Hash:     "service-old", AuthMode: skeldesc.AuthModeRequired,
				},
				SkelName:       "demo.user.OldService",
				DescriptorHash: "service-old",
				Main:           false,
			},
		},
	}})

	value, ok := watchServer.Get(watched.FormatDescriptorActorKey("demo.user.UserActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
  "name": "",
  "skelName": "demo.user.UserActor",
  "hash": "actor-main",
  "vias": [],
  "auth": {
    "methodName": "",
    "credential": {
      "name": "",
      "skelName": "demo.user.UserCredential",
      "hash": ""
    },
    "info": {
      "name": "",
      "skelName": "demo.user.UserInfo",
      "hash": ""
    },
    "service": {
      "name": "",
      "skelName": "demo.user.UserAuthService",
      "hash": "auth-service-main",
      "pub": false,
      "authMode": "required",
      "methods": []
    }
  }
}`, value)
	value, ok = watchServer.Get(watched.FormatDescriptorActorKey("demo.user.NoAuthActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
  "name": "",
  "skelName": "demo.user.NoAuthActor",
  "hash": "actor-no-auth",
  "vias": []
}`, value)
	_, ok = watchServer.Get(watched.FormatDescriptorActorKey("demo.user.OldActor"))
	assert.False(t, ok)

	value, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.user.UserService"))
	require.True(t, ok)
	assert.JSONEq(t, `{
  "name": "",
  "skelName": "demo.user.UserService",
  "hash": "service-main",
  "pub": false,
  "authMode": "required",
  "audiences": [
    {
      "name": "",
      "skelName": "demo.user.UserActor",
      "via": "client"
    }
  ],
  "methods": [
    {
      "name": "Get",
      "skelName": "Get",
      "hash": "",
      "authMode": "optional",
      "effectiveAuthMode": "optional"
    },
    {
      "name": "Update",
      "skelName": "Update",
      "hash": "",
      "authMode": "required",
      "effectiveAuthMode": "required"
    }
  ],
  "api": true
}`, value)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.user.OldService"))
	assert.False(t, ok)
}

func TestSyncerWriteDescriptorsWritesMainActorAndServiceDescriptors(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	target.WriteDescriptors([]core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{{
			Descriptor: &skeldesc.Actor{
				SkelName: "vine.hub.admin.AdminActor",
				Hash:     "admin-actor-main",
			},
			SkelName:       "vine.hub.admin.AdminActor",
			DescriptorHash: "admin-actor-main",
			Main:           true,
		}},
		Services: []core.DescriptorVersion[*skeldesc.Service]{{
			Descriptor: &skeldesc.Service{
				SkelName: "vine.hub.admin.SkeletonApiService", Api: true,
				Hash:     "skeleton-service-main",
				AuthMode: skeldesc.AuthModeOptional,
			},
			SkelName:       "vine.hub.admin.SkeletonApiService",
			DescriptorHash: "skeleton-service-main",
			Main:           true,
		}},
	}})

	value, ok := watchServer.Get(watched.FormatDescriptorActorKey("vine.hub.admin.AdminActor"))
	require.True(t, ok)
	assert.JSONEq(t, `{
  "name": "",
  "skelName": "vine.hub.admin.AdminActor",
  "hash": "admin-actor-main",
  "vias": []
}`, value)

	value, ok = watchServer.Get(watched.FormatDescriptorServiceKey("vine.hub.admin.SkeletonApiService"))
	require.True(t, ok)
	assert.JSONEq(t, `{
  "name": "",
  "skelName": "vine.hub.admin.SkeletonApiService",
  "hash": "skeleton-service-main",
  "pub": false,
  "authMode": "optional",
  "methods": [],
  "api": true
}`, value)
}

func TestSyncerSyncDescriptorsDoesNotDeleteVineHubDescriptors(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	target.WriteDescriptors([]core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{{
			Descriptor:     &skeldesc.Actor{SkelName: "vine.hub.admin.AdminActor", Hash: "admin-actor-main"},
			SkelName:       "vine.hub.admin.AdminActor",
			DescriptorHash: "admin-actor-main",
			Main:           true,
		}},
		Services: []core.DescriptorVersion[*skeldesc.Service]{{
			Descriptor:     &skeldesc.Service{Api: true, SkelName: "vine.hub.admin.SkeletonApiService", Hash: "skeleton-service-main", AuthMode: skeldesc.AuthModeRequired},
			SkelName:       "vine.hub.admin.SkeletonApiService",
			DescriptorHash: "skeleton-service-main",
			Main:           true,
		}},
	}})

	target.SyncDescriptors([]core.DomainDescriptorView{})

	_, ok := watchServer.Get(watched.FormatDescriptorActorKey("vine.hub.admin.AdminActor"))
	assert.True(t, ok)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("vine.hub.admin.SkeletonApiService"))
	assert.True(t, ok)
}

func TestSyncerSyncDescriptorsRemovesStaleDescriptors(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	target.SyncDescriptors([]core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{{
			Descriptor: &skeldesc.Actor{
				SkelName: "demo.user.UserActor", Auth: &skeldesc.ActorAuth{Service: &skeldesc.Service{SkelName: "demo.user.UserAuthService", AuthMode: skeldesc.AuthModeRequired}},
			},
			SkelName:       "demo.user.UserActor",
			DescriptorHash: "actor-main",
			Main:           true,
		}},
		Services: []core.DescriptorVersion[*skeldesc.Service]{{
			Descriptor: &skeldesc.Service{Api: true,
				SkelName: "demo.user.UserService", AuthMode: skeldesc.AuthModeRequired,
			},
			SkelName:       "demo.user.UserService",
			DescriptorHash: "service-main",
			Main:           true,
		}},
	}})
	_, ok := watchServer.Get(watched.FormatDescriptorActorKey("demo.user.UserActor"))
	require.True(t, ok)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.user.UserService"))
	require.True(t, ok)

	target.SyncDescriptors([]core.DomainDescriptorView{})

	_, ok = watchServer.Get(watched.FormatDescriptorActorKey("demo.user.UserActor"))
	assert.False(t, ok)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.user.UserService"))
	assert.False(t, ok)
}

func TestSyncerSyncDescriptorsOnlyWritesChangedHashes(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	view := []core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{{
			Descriptor: &skeldesc.Actor{
				SkelName: "demo.user.UserActor",
				Hash:     "actor-main",
			},
			SkelName:       "demo.user.UserActor",
			DescriptorHash: "actor-main",
			Main:           true,
		}},
		Services: []core.DescriptorVersion[*skeldesc.Service]{{
			Descriptor: &skeldesc.Service{Api: true,
				SkelName: "demo.user.UserService",
				Hash:     "service-main", AuthMode: skeldesc.AuthModeRequired,
			},
			SkelName:       "demo.user.UserService",
			DescriptorHash: "service-main",
			Main:           true,
		}},
	}}

	baseRevision := testWatchRevision(t, watchServer)
	target.SyncDescriptors(view)
	firstRevision := testWatchRevision(t, watchServer)
	assert.Equal(t, baseRevision+1, firstRevision)

	target.SyncDescriptors(view)
	secondRevision := testWatchRevision(t, watchServer)
	assert.Equal(t, firstRevision, secondRevision)

	view[0].Services[0].Descriptor.Hash = "service-next"
	view[0].Services[0].Descriptor.Deprecated = true
	view[0].Services[0].Descriptor.DeprecatedReason = "Use demo.user.NextService instead."
	view[0].Services[0].DescriptorHash = "service-next"
	target.SyncDescriptors(view)
	thirdRevision := testWatchRevision(t, watchServer)
	assert.Equal(t, secondRevision+1, thirdRevision)

	value, ok := watchServer.Get(watched.FormatDescriptorServiceKey("demo.user.UserService"))
	require.True(t, ok)
	var service watched.DescriptorService
	require.NoError(t, json.Unmarshal([]byte(value), &service))
	assert.True(t, service.Deprecated)
	assert.Equal(t, "Use demo.user.NextService instead.", service.DeprecatedReason)
}

func testWatchRevision(t *testing.T, watchServer *watchserver.Server) uint64 {
	t.Helper()
	value, ok := watchServer.Get(hubwatch.RevisionKey)
	require.True(t, ok)
	revision, err := strconv.ParseUint(value, 10, 64)
	require.NoError(t, err)
	return revision
}

func TestPortalDescriptorsExcludeBackendServices(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)
	view := []core.DomainDescriptorView{{Services: []core.DescriptorVersion[*skeldesc.Service]{
		{Main: true, Descriptor: &skeldesc.Service{SkelName: "demo.BackendService", Pub: true, Hash: "backend", AuthMode: skeldesc.AuthModeRequired}},
		{Main: true, Descriptor: &skeldesc.Service{SkelName: "demo.ApiService", Api: true, Hash: "api", AuthMode: skeldesc.AuthModeRequired}},
		{Main: true, Descriptor: &skeldesc.Service{SkelName: "demo.LegacyService", AuthMode: skeldesc.AuthModeRequired, Hash: "legacy"}},
	}}}
	target.SyncDescriptors(view)
	for _, name := range []string{"demo.ApiService"} {
		_, ok := watchServer.Get(watched.FormatDescriptorServiceKey(name))
		require.True(t, ok, name)
	}
	_, ok := watchServer.Get(watched.FormatDescriptorServiceKey("demo.LegacyService"))
	require.False(t, ok)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.BackendService"))
	require.False(t, ok)
	view[0].Services[1].Descriptor.Api = false
	view[0].Services[1].Descriptor.Pub = true
	target.SyncDescriptors(view)
	_, ok = watchServer.Get(watched.FormatDescriptorServiceKey("demo.ApiService"))
	require.False(t, ok, "API converted to backend must be removed from Portal")
}
