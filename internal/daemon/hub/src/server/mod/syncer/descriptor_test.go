package syncer

import (
	"testing"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	descriptorrepo "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/descriptor"
	"go.yorun.ai/vine/util/vcode"
)

func TestVersionedServicesPublishAndUnregisterIndependently(t *testing.T) {
	repo := new(descriptorrepo.DescriptorRepo)
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	syncer := testSyncer(server)
	for _, version := range []string{"", "V1", "V2", "V10"} {
		name := "OrderApiService" + version
		repo.SaveDomainDescriptors("demo.app", "instance"+version, []*skeldesc.Domain{{
			Name: "demo", Hash: "domain" + version,
			Generated: new(skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"}),
			Services: []*skeldesc.Service{{
				Name: name, SkelName: "demo." + name, Hash: "service" + version,
				Api: true, AuthMode: skeldesc.AuthModeRequired,
			}},
		}})
	}
	syncer.SyncDescriptors(repo.ListDomainDescriptorViews())
	for _, version := range []string{"", "V1", "V2", "V10"} {
		name := "demo.OrderApiService" + version
		value, ok := server.Get(watched.FormatDescriptorServiceKey(name))
		require.True(t, ok, name)
		require.Equal(t, name, vcode.MustUnmarshalJsonS[*skeldesc.Service](value).SkelName)
	}
	repo.ReleaseDomainDescriptors("demo.app", "instanceV1")
	syncer.SyncDescriptors(repo.ListDomainDescriptorViews())
	_, ok := server.Get(watched.FormatDescriptorServiceKey("demo.OrderApiServiceV1"))
	require.False(t, ok)
	for _, version := range []string{"", "V2", "V10"} {
		_, ok := server.Get(watched.FormatDescriptorServiceKey("demo.OrderApiService" + version))
		require.True(t, ok, "removing V1 must not remove %q", version)
	}
}

func TestAdditionalDescriptorTypesPublicationLifecycle(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		build func(hash string, main bool) (core.DomainDescriptorView, any)
	}{
		{
			name: "domain", key: watched.FormatDescriptorDomainKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Domain{Name: "demo.Contract", Hash: hash, Description: hash})
				return core.DomainDescriptorView{DomainVersion: core.DomainDescriptorVersion{Descriptor: value, Main: main}}, value
			},
		},
		{
			name: "data", key: watched.FormatDescriptorDataKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Data{SkelName: "demo.Contract", Hash: hash, Description: hash, Members: []*skeldesc.Member{{Name: "id", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt})}}})
				return core.DomainDescriptorView{Data: []core.DescriptorVersion[*skeldesc.Data]{{Descriptor: value, Main: main}}}, value
			},
		},
		{
			name: "enum", key: watched.FormatDescriptorEnumKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Enum{SkelName: "demo.Contract", Hash: hash, Description: hash, Items: []*skeldesc.EnumItem{{Name: "ACTIVE"}}})
				return core.DomainDescriptorView{Enums: []core.DescriptorVersion[*skeldesc.Enum]{{Descriptor: value, Main: main}}}, value
			},
		},
		{
			name: "config", key: watched.FormatDescriptorConfigKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Config{SkelName: "demo.Contract", Hash: hash, Description: hash, Lifecycle: skeldesc.ConfigLifecycleInstant, Members: []*skeldesc.Member{{Name: "flag", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBoolean})}}})
				return core.DomainDescriptorView{Configs: []core.DescriptorVersion[*skeldesc.Config]{{Descriptor: value, Main: main}}}, value
			},
		},
		{
			name: "event", key: watched.FormatDescriptorEventKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Event{SkelName: "demo.Contract", Hash: hash, Description: hash, Members: []*skeldesc.Member{{Name: "id", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt})}}})
				return core.DomainDescriptorView{Events: []core.DescriptorVersion[*skeldesc.Event]{{Descriptor: value, Main: main}}}, value
			},
		},
		{
			name: "task", key: watched.FormatDescriptorTaskKey("demo.Contract"),
			build: func(hash string, main bool) (core.DomainDescriptorView, any) {
				value := new(skeldesc.Task{SkelName: "demo.Contract", Hash: hash, Description: hash, Triggers: []*skeldesc.TaskTrigger{{Name: "Manual", SkelName: "manual", Arguments: []*skeldesc.Member{{Name: "id", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt})}}}}})
				return core.DomainDescriptorView{Tasks: []core.DescriptorVersion[*skeldesc.Task]{{Descriptor: value, Main: main}}}, value
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := watchserver.NewServerForTest()
			t.Cleanup(server.AfterAppStop)
			target := testSyncer(server)
			current, expected := test.build("current", true)
			other, _ := test.build("old", false)
			views := []core.DomainDescriptorView{current, other}
			target.SyncDescriptors(views)
			value, ok := server.Get(test.key)
			require.True(t, ok)
			require.JSONEq(t, vcode.MustMarshalJsonS(expected), value)
			revision := testWatchRevision(t, server)
			target.SyncDescriptors(views)
			require.Equal(t, revision, testWatchRevision(t, server), "unchanged descriptors must not publish another revision")

			updated, expected := test.build("updated", true)
			target.SyncDescriptors([]core.DomainDescriptorView{updated, other})
			value, ok = server.Get(test.key)
			require.True(t, ok)
			require.JSONEq(t, vcode.MustMarshalJsonS(expected), value)
			require.Equal(t, revision+1, testWatchRevision(t, server))

			fallback, expected := test.build("old", true)
			target.SyncDescriptors([]core.DomainDescriptorView{fallback})
			value, ok = server.Get(test.key)
			require.True(t, ok)
			require.JSONEq(t, vcode.MustMarshalJsonS(expected), value, "unregistering the selected version must publish its replacement")

			target.SyncDescriptors(nil)
			_, ok = server.Get(test.key)
			require.False(t, ok, "unregistered descriptors must be removed")
			target.SyncDescriptors([]core.DomainDescriptorView{other})
			_, ok = server.Get(test.key)
			require.False(t, ok, "non-selected versions must not be published")

			target.WriteDescriptors([]core.DomainDescriptorView{current, other})
			target.SyncDescriptors(nil)
			value, ok = server.Get(test.key)
			require.True(t, ok, "WriteDescriptors must not join the application diff/delete lifecycle")
			_, expected = test.build("current", true)
			require.JSONEq(t, vcode.MustMarshalJsonS(expected), value)
		})
	}
}

func TestWebDescriptorsPublicationLifecycle(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	views := []core.DomainDescriptorView{{Webs: []core.DescriptorVersion[*skeldesc.Web]{
		{Descriptor: new(skeldesc.Web{SkelName: "demo.Web", Hash: "first", AuthMode: skeldesc.AuthModeRequired}), Main: true},
		{Descriptor: new(skeldesc.Web{SkelName: "demo.OldWeb", Hash: "old", AuthMode: skeldesc.AuthModeRequired}), Main: false},
	}}}
	target.SyncDescriptors(views)
	key := watched.FormatDescriptorWebKey("demo.Web")
	value, ok := server.Get(key)
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeRequired, vcode.MustUnmarshalJsonS[*skeldesc.Web](value).AuthMode)
	_, ok = server.Get(watched.FormatDescriptorWebKey("demo.OldWeb"))
	require.False(t, ok)
	revision := testWatchRevision(t, server)
	target.SyncDescriptors(views)
	require.Equal(t, revision, testWatchRevision(t, server))
	views[0].Webs[0].Descriptor.Hash = "second"
	views[0].Webs[0].Descriptor.AuthMode = skeldesc.AuthModeOff
	target.SyncDescriptors(views)
	value, ok = server.Get(key)
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeOff, vcode.MustUnmarshalJsonS[*skeldesc.Web](value).AuthMode)
	target.SyncDescriptors(nil)
	_, ok = server.Get(key)
	require.False(t, ok)
	target.WriteDescriptors(views)
	target.SyncDescriptors(nil)
	_, ok = server.Get(key)
	require.True(t, ok, "WriteDescriptors must not join the application diff/delete lifecycle")
}

func TestDomainDescriptorsRetainCompleteContractWithOnlyAPIServiceKeys(t *testing.T) {
	server := watchserver.NewServerForTest()
	t.Cleanup(server.AfterAppStop)
	target := testSyncer(server)
	api := new(skeldesc.Service{SkelName: "demo.API", Hash: "api", Api: true})
	internal := new(skeldesc.Service{SkelName: "demo.Internal", Hash: "internal"})
	domain := new(skeldesc.Domain{Name: "demo", Hash: "domain", Services: []*skeldesc.Service{api, internal}})
	view := core.DomainDescriptorView{
		DomainVersion: core.DomainDescriptorVersion{Descriptor: domain, Main: true},
		Services: []core.DescriptorVersion[*skeldesc.Service]{
			{Descriptor: api, Main: true},
			{Descriptor: internal, Main: true},
		},
	}
	for _, write := range []func([]core.DomainDescriptorView){target.SyncDescriptors, target.WriteDescriptors} {
		write([]core.DomainDescriptorView{view})
		value, ok := server.Get(watched.FormatDescriptorDomainKey("demo"))
		require.True(t, ok)
		require.JSONEq(t, vcode.MustMarshalJsonS(domain), value)
		_, ok = server.Get(watched.FormatDescriptorServiceKey(api.SkelName))
		require.True(t, ok)
		_, ok = server.Get(watched.FormatDescriptorServiceKey(internal.SkelName))
		require.False(t, ok)
	}
}
