package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerLoadsActorAndServiceSchemas(t *testing.T) {
	manager := testManager(map[string]string{
		watched.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(watched.SchemaActor{
			SkelName: "demo.UserActor",
			Hash:     "actor-main",
		}),
		watched.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.SchemaService{
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
	watchClient := hubwatch.NewTestClient(valuesByKey)
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Watch:   watchClient,
	}
	epmgrManager.DIInit()
	manager := &Access{
		Context: context.Background(),
		Watch:   watchClient,
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}
