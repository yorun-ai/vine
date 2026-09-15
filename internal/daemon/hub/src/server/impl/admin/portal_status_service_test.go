package admin

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/skel"
	controlskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	controlimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
)

type _PortalStatusServicePortalInstanceRepo struct {
	core.PortalInstanceRepo

	instances []*core.PortalInstance
}

func (r *_PortalStatusServicePortalInstanceRepo) ListPortalInstances() []*core.PortalInstance {
	return r.instances
}

func TestPortalStatusServiceListsRegisteredPortals(t *testing.T) {
	service := &PortalStatusApiServiceServerImpl{
		PortalInstanceRepo: &_PortalStatusServicePortalInstanceRepo{
			instances: []*core.PortalInstance{
				{InstanceId: "22222222-2222-2222-2222-222222222222", Version: "1.2.3"},
				{InstanceId: "11111111-1111-1111-1111-111111111111", Version: "1.2.3"},
			},
		},
	}

	instances := service.List()

	require.Len(t, instances, 2)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", instances[0].InstanceId)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", instances[1].InstanceId)
	assert.Equal(t, "1.2.3", instances[0].Version)
}

// TestPortalStatusServiceListsWhatPortalRegistered runs the registration and the
// Dashboard listing against one real watch store, so a change in the recorded
// fields cannot leave the two sides disagreeing.
func TestPortalStatusServiceListsWhatPortalRegistered(t *testing.T) {
	watchServer := &watchserver.Server{
		Context:    t.Context(),
		Option:     &flag.Flag{},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
	}
	watchServer.DIInit()
	t.Cleanup(watchServer.AfterAppStop)

	instanceRepo := &repo.WatchPortalInstanceRepo{
		WatchServer: watchServer,
		InprocFlag:  &internalapp.InternalInprocFlag{},
	}
	registry := &controlimpl.PortalRegistryServiceServerImpl{PortalInstanceRepo: instanceRepo}
	service := &PortalStatusApiServiceServerImpl{PortalInstanceRepo: instanceRepo}

	assert.Empty(t, service.List())

	instanceId := skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	registry.Register(controlskeled.PortalRegistration{
		InstanceId: instanceId,
		Version:    "1.2.3",
	})

	instances := service.List()
	require.Len(t, instances, 1)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", instances[0].InstanceId)
	assert.Equal(t, "1.2.3", instances[0].Version)

	registry.Unregister(instanceId)
	assert.Empty(t, service.List())
}
