package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
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
