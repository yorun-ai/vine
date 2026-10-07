package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
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
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName: "demo.UserService",
			Hash:     "service-main",
			AuthMode: skeldesc.AuthModeRequired,
		}),
	})

	actor, ok := access.actorDescriptor("demo.UserActor")
	require.True(t, ok)
	assert.Equal(t, "actor-main", actor.Hash)

	service, ok := access.serviceDescriptor("demo.UserService")
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
	access := &Access{
		Context: context.Background(),
		Watch:   watchClient,
		Epmgr:   endpointManager,
	}
	access.DIInit()
	return access
}
