package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerHandlesActorSchemaEvents(t *testing.T) {
	manager := testManager(map[string]string{})

	key := watched.FormatSchemaActorKey("demo.UserActor")
	manager.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
	})
	actor, ok := manager.actorSchema("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	manager.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-next",
		}),
	})
	actor, ok = manager.actorSchema("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-next", actor.Hash)

	manager.handleActorEvent(hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	_, ok = manager.actorSchema("demo.UserActor")
	assert.False(t, ok)
}

func TestManagerHandlesServiceSchemaEvents(t *testing.T) {
	manager := testManager(map[string]string{})

	key := watched.FormatSchemaServiceKey("demo.UserService")
	manager.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.SchemaService{
			SkelName: "demo.UserService",
			Hash:     "service-main",
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeAuth},
			},
		}),
	})
	service, ok := manager.serviceSchema("demo.UserService")
	require.True(t, ok)
	assert.Equal(t, "service-main", service.Hash)
	require.Len(t, service.Methods, 1)
	assert.Equal(t, skel.AuthModeAuth, service.Methods[0].AuthMode)

	manager.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	_, ok = manager.serviceSchema("demo.UserService")
	assert.False(t, ok)
}
