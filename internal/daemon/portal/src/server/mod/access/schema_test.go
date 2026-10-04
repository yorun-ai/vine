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
	manager := testManager(t, map[string]string{})

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
	manager := testManager(t, map[string]string{})

	key := watched.FormatSchemaServiceKey("demo.UserService")
	manager.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindUpsert,
		Key:  key,
		Value: vcode.MustMarshalJsonS(watched.SchemaService{
			SkelName: "demo.UserService",
			Hash:     "service-main",
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeRequired},
			},
		}),
	})
	service, ok := manager.serviceSchema("demo.UserService")
	require.True(t, ok)
	assert.Equal(t, "service-main", service.Hash)
	require.Len(t, service.Methods, 1)
	assert.Equal(t, skel.AuthModeRequired, service.Methods[0].AuthMode)

	manager.handleServiceEvent(hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	_, ok = manager.serviceSchema("demo.UserService")
	assert.False(t, ok)
}

func TestManagerHandlesWebSchemaEvents(t *testing.T) {
	key := watched.FormatSchemaWebKey("demo.Web")
	manager := testManager(t, map[string]string{key: vcode.MustMarshalJsonS(watched.SchemaWeb{SkelName: "demo.Web", AuthMode: skel.AuthModeRequired})})
	schema, ok := manager.webSchema("demo.Web")
	require.True(t, ok)
	require.Equal(t, skel.AuthModeRequired, schema.AuthMode)
	manager.handleWebEvent(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key,
		Value: vcode.MustMarshalJsonS(watched.SchemaWeb{SkelName: "demo.Web", AuthMode: skel.AuthModeAnonymous}),
	})
	schema, ok = manager.webSchema("demo.Web")
	require.True(t, ok)
	require.Equal(t, skel.AuthModeAnonymous, schema.AuthMode)
	manager.handleWebEvent(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	_, ok = manager.webSchema("demo.Web")
	require.False(t, ok)
}
