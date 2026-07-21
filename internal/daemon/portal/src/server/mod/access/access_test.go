package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerLoadsActorAndServiceSchemas(t *testing.T) {
	manager := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName: "demo.UserService",
			Hash:     "service-main",
			AuthMode: skel.AuthModeAuth,
		}),
	})

	actor, ok := manager.actorSchema("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	service, ok := manager.serviceSchema("demo.UserService")
	require.True(t, ok)
	assert.Equal(t, "service-main", service.Hash)
	assert.Equal(t, skel.AuthModeAuth, service.AuthMode)
}

func testManager(valuesByKey map[string]string) *Access {
	redisClient := hubredis.NewTestClient(valuesByKey)
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   redisClient,
	}
	epmgrManager.DIInit()
	manager := &Access{
		Context: context.Background(),
		Redis:   redisClient,
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}
