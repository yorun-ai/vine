package access

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

func TestAccessLoadsActorAndServiceDescriptors(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(watched.DescriptorActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{Api: true,
			SkelName: "demo.UserService",
			Hash:     "service-main",
			AuthMode: skeldesc.AuthModeRequired,
		}),
	})

	actor, ok := access.Desc.Actor("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	service, ok := access.Desc.Service("demo.UserService")
	require.True(t, ok)
	assert.Equal(t, "service-main", service.Hash)
	assert.Equal(t, skeldesc.AuthModeRequired, service.AuthMode)
}

func newTestAccess(t *testing.T, valuesByKey map[string]string) *Access {
	watchClient := watchtest.New(t, valuesByKey)
	endpointManager := &epmgr.Manager{
		Context: context.Background(),
		Watch:   watchClient,
	}
	endpointManager.DIInit()
	descriptors := &desc.Manager{Context: t.Context(), Watch: watchClient}
	descriptors.DIInit()
	t.Cleanup(descriptors.AfterAppStop)
	access := &Access{
		Context: context.Background(),
		Desc:    descriptors,
		Epmgr:   endpointManager,
	}
	access.DIInit()
	t.Cleanup(access.AfterAppStop)
	return access
}

func TestAdmissionChangesReconcileEndpointWatches(t *testing.T) {
	actor := new(skeldesc.Actor{SkelName: "demo.User", Auth: new(skeldesc.ActorAuth{Service: new(skeldesc.Service{SkelName: "demo.Auth"})}), Permission: new(skeldesc.ActorPermission{Service: new(skeldesc.Service{SkelName: "demo.Permission"})})})
	resource := new(skeldesc.Resource{SkelName: "demo.Document", CheckService: new(skeldesc.Service{SkelName: "demo.Check"})})
	target := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey(actor.SkelName):       vcode.MustMarshalJsonS(actor),
		watched.FormatDescriptorResourceKey(resource.SkelName): vcode.MustMarshalJsonS(resource),
	})
	assertConfigured := func(name string, expected bool) {
		t.Helper()
		_, configured := target.Epmgr.NextRpcEndpoint(name)
		require.Equal(t, expected, configured, name)
	}
	assertConfigured("demo.Auth", true)
	assertConfigured("demo.Permission", true)
	assertConfigured("demo.Check", true)
	target.mutex.Lock()
	original := target.serviceWatches["permission:demo.User"]
	target.mutex.Unlock()
	actor.Hash = "updated"
	actor.Auth.Service = new(skeldesc.Service{SkelName: "demo.NextAuth"})
	target.Desc.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorActorKey(actor.SkelName), Value: vcode.MustMarshalJsonS(actor)})
	require.Eventually(t, func() bool { _, configured := target.Epmgr.NextRpcEndpoint("demo.NextAuth"); return configured }, time.Second, time.Millisecond)
	assertConfigured("demo.Auth", false)
	target.mutex.Lock()
	retained := target.serviceWatches["permission:demo.User"]
	target.mutex.Unlock()
	require.Same(t, original, retained, "an unchanged endpoint watch must not be recreated")

	target.Desc.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorResourceKey(resource.SkelName)})
	require.Eventually(t, func() bool { _, configured := target.Epmgr.NextRpcEndpoint("demo.Check"); return !configured }, time.Second, time.Millisecond)
	target.AfterAppStop()
	assertConfigured("demo.NextAuth", false)
	assertConfigured("demo.Permission", false)
	// The descriptor owner stays live after access releases its observer.
	actor.Auth.Service = new(skeldesc.Service{SkelName: "demo.AfterStop"})
	target.Desc.Watch.(*watchtest.Client).Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorActorKey(actor.SkelName), Value: vcode.MustMarshalJsonS(actor)})
	require.Eventually(t, func() bool {
		value, ok := target.Desc.Actor(actor.SkelName)
		return ok && value.Auth.Service.SkelName == "demo.AfterStop"
	}, time.Second, time.Millisecond)
	assertConfigured("demo.AfterStop", false)
}
