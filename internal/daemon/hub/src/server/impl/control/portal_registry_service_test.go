package control

import (
	"testing"
	"time"
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

func (r *_PortalInstanceRepoSpy) SavePortalInstance(instance *core.PortalInstance) {
	r.saved = append(r.saved, instance)
}

func (r *_PortalInstanceRepoSpy) ListPortalInstances() []*core.PortalInstance {
	return r.instances
}

func (r *_PortalInstanceRepoSpy) GetPortalInstance(instanceId string) (*core.PortalInstance, bool) {
	for _, instance := range r.instances {
		if instance.InstanceId == instanceId {
			return instance, true
		}
	}
	return nil, false
}

func (r *_PortalInstanceRepoSpy) KeepPortalInstance(instanceId string) bool {
	r.kept = append(r.kept, instanceId)
	return r.keepResult
}

func (r *_PortalInstanceRepoSpy) RemovePortalInstance(instanceId string) {
	r.removed = append(r.removed, instanceId)
}

func (r *_PortalInstanceRepoSpy) PopExpiredPortalLeases() []string {
	return nil
}

func TestPortalRegistryServiceRegistersAndUnregisters(t *testing.T) {
	repo := &_PortalInstanceRepoSpy{}
	service := &PortalRegistryServiceServerImpl{PortalInstanceRepo: repo}

	instanceId := skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	startedAt := time.Date(2026, 9, 15, 6, 30, 0, 0, time.UTC)
	service.Register(skeled.PortalRegistration{
		InstanceId: instanceId,
		Version:    "1.2.3",
		StartedAt:  skel.NewTimestamp(startedAt),
	})

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", repo.saved[0].InstanceId)
	assert.Equal(t, "1.2.3", repo.saved[0].Version)
	assert.Equal(t, startedAt, repo.saved[0].StartedAt)

	service.Unregister(instanceId)
	assert.Equal(t, []string{"11111111-1111-1111-1111-111111111111"}, repo.removed)
}

func TestPortalRegistryServiceHeartbeatReportsRegistration(t *testing.T) {
	instanceId := skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))

	unknownRepo := &_PortalInstanceRepoSpy{}
	unknownService := &PortalRegistryServiceServerImpl{PortalInstanceRepo: unknownRepo}
	assert.False(t, unknownService.Heartbeat(skeled.PortalStatus{InstanceId: instanceId}))
	assert.Equal(t, []string{"11111111-1111-1111-1111-111111111111"}, unknownRepo.kept)

	knownRepo := &_PortalInstanceRepoSpy{keepResult: true}
	knownService := &PortalRegistryServiceServerImpl{PortalInstanceRepo: knownRepo}
	assert.True(t, knownService.Heartbeat(skeled.PortalStatus{InstanceId: instanceId}))
}
