package control

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type _PortalInstanceRepoSpy struct {
	saved      []*core.PortalInstance
	removed    []string
	kept       []string
	keepResult bool
	instances  []*core.PortalInstance
}

func (r *_PortalInstanceRepoSpy) Save(instance *core.PortalInstance) {
	r.saved = append(r.saved, instance)
}

func (r *_PortalInstanceRepoSpy) List() []*core.PortalInstance {
	return r.instances
}

func (r *_PortalInstanceRepoSpy) GetById(instanceId string) (*core.PortalInstance, bool) {
	for _, instance := range r.instances {
		if instance.InstanceId == instanceId {
			return instance, true
		}
	}
	return nil, false
}

func (r *_PortalInstanceRepoSpy) Keep(instanceId string) bool {
	r.kept = append(r.kept, instanceId)
	return r.keepResult
}

func (r *_PortalInstanceRepoSpy) Remove(instanceId string) {
	r.removed = append(r.removed, instanceId)
}

func (r *_PortalInstanceRepoSpy) PopExpiredLeases() []string {
	return nil
}

func TestPortalRegistryServiceRegistersAndUnregisters(t *testing.T) {
	repo := &_PortalInstanceRepoSpy{}
	service := &PortalRegistryServiceServerImpl{PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: repo}}

	instanceId := skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	service.Register(skeled.PortalRegistration{InstanceId: instanceId, Version: "1.2.3"})

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", repo.saved[0].InstanceId)
	assert.Equal(t, "1.2.3", repo.saved[0].Version)

	service.Unregister(instanceId)
	assert.Equal(t, []string{"11111111-1111-1111-1111-111111111111"}, repo.removed)
}

func TestPortalRegistryServiceHeartbeatReportsRegistration(t *testing.T) {
	instanceId := skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))

	unknownRepo := &_PortalInstanceRepoSpy{}
	unknownService := &PortalRegistryServiceServerImpl{PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: unknownRepo}}
	assert.False(t, unknownService.Heartbeat(skeled.PortalStatus{InstanceId: instanceId}))
	assert.Equal(t, []string{"11111111-1111-1111-1111-111111111111"}, unknownRepo.kept)

	knownRepo := &_PortalInstanceRepoSpy{keepResult: true}
	knownService := &PortalRegistryServiceServerImpl{PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: knownRepo}}
	assert.True(t, knownService.Heartbeat(skeled.PortalStatus{InstanceId: instanceId}))
}
