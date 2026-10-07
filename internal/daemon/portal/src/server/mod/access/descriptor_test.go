package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestAccessHandlesActorDescriptorEvents(t *testing.T) {
	access := newTestAccess(t, map[string]string{})

	key := watched.FormatDescriptorActorKey("demo.UserActor")
	access.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.DescriptorActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
	})
	actor, ok := access.actorDescriptor("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	access.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.DescriptorActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-next",
		}),
	})
	actor, ok = access.actorDescriptor("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-next", actor.Hash)

	access.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	_, ok = access.actorDescriptor("demo.UserActor")
	assert.False(t, ok)
}

func TestAccessHandlesServiceDescriptorEvents(t *testing.T) {
	access := newTestAccess(t, map[string]string{})

	key := watched.FormatDescriptorServiceKey("demo.UserService")
	access.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName: "demo.UserService",
			Hash:     "service-main",
			Methods: []*skeldesc.Method{
				{SkelName: "Get", AuthMode: skeldesc.AuthModeRequired, Name: "Get", EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	})
	service, ok := access.serviceDescriptor("demo.UserService")
	require.True(t, ok)
	assert.Equal(t, "service-main", service.Hash)
	require.Len(t, service.Methods, 1)
	assert.Equal(t, skeldesc.AuthModeRequired, service.Methods[0].AuthMode)

	access.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	_, ok = access.serviceDescriptor("demo.UserService")
	assert.False(t, ok)
}

func TestAccessHandlesWebDescriptorEvents(t *testing.T) {
	key := watched.FormatDescriptorWebKey("demo.Web")
	access := newTestAccess(t, map[string]string{key: vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.Web", AuthMode: skeldesc.AuthModeRequired})})
	descriptor, ok := access.webDescriptor("demo.Web")
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeRequired, descriptor.AuthMode)
	access.handleWebEvent(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key,
		Value: vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.Web", AuthMode: skeldesc.AuthModeAnonymous}),
	})
	descriptor, ok = access.webDescriptor("demo.Web")
	require.True(t, ok)
	require.Equal(t, skeldesc.AuthModeAnonymous, descriptor.AuthMode)
	access.handleWebEvent(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	_, ok = access.webDescriptor("demo.Web")
	require.False(t, ok)
}
