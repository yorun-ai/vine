package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerHandlesActorSchemaEvents(t *testing.T) {
	manager := testManager(map[string]string{})

	key := redised.FormatSchemaActorKey("demo.UserActor")
	manager.handleActorEvent(hubredis.Event{
		Kind: hubredis.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
	})
	actor, ok := manager.actorSchema("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	manager.handleActorEvent(hubredis.Event{
		Kind: hubredis.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-next",
		}),
	})
	actor, ok = manager.actorSchema("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-next", actor.Hash)

	manager.handleActorEvent(hubredis.Event{
		Kind: hubredis.EventKindDelete,
		Key:  key,
	})
	_, ok = manager.actorSchema("demo.UserActor")
	assert.False(t, ok)
}

func TestManagerHandlesServiceSchemaEvents(t *testing.T) {
	manager := testManager(map[string]string{})

	key := redised.FormatSchemaServiceKey("demo.UserService")
	manager.handleServiceEvent(hubredis.Event{
		Kind: hubredis.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(redised.SchemaService{
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

	manager.handleServiceEvent(hubredis.Event{
		Kind: hubredis.EventKindDelete,
		Key:  key,
	})
	_, ok = manager.serviceSchema("demo.UserService")
	assert.False(t, ok)
}
