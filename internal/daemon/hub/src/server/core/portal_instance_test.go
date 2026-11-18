package core

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type portalLeaseRepo struct {
	PortalInstanceRepo
	candidates []string
	instances  map[string]*PortalInstance
	failRemove string
}

func (r *portalLeaseRepo) ListExpiredLeases() []string {
	return slices.Clone(r.candidates)
}

func (r *portalLeaseRepo) GetById(id string) (*PortalInstance, bool) {
	instance, ok := r.instances[id]
	return instance, ok
}

func (r *portalLeaseRepo) Remove(id string) {
	if id == r.failRemove {
		panic("portal removal failed")
	}
	delete(r.instances, id)
	r.candidates = slices.DeleteFunc(r.candidates, func(candidate string) bool { return candidate == id })
}

func TestPortalSweepRechecksRenewedCandidate(t *testing.T) {
	repo := &portalLeaseRepo{
		candidates: []string{"renewed"},
		instances: map[string]*PortalInstance{
			"renewed": {InstanceId: "renewed", ExpiresAt: time.Now().Add(time.Minute)},
		},
	}
	target := &PortalInstanceCore{PortalInstanceRepo: repo}
	require.Zero(t, target.SweepExpired())
	require.Contains(t, repo.instances, "renewed")
}

func TestPortalSweepRetriesFailureAndContinuesOtherInstances(t *testing.T) {
	repo := &portalLeaseRepo{
		candidates: []string{"failed", "healthy"},
		instances: map[string]*PortalInstance{
			"failed":  {InstanceId: "failed", ExpiresAt: time.Now().Add(-time.Minute)},
			"healthy": {InstanceId: "healthy", ExpiresAt: time.Now().Add(-time.Minute)},
		},
		failRemove: "failed",
	}
	target := &PortalInstanceCore{PortalInstanceRepo: repo}
	require.Equal(t, 1, target.SweepExpired())
	require.Equal(t, []string{"failed"}, repo.candidates)
	require.Contains(t, repo.instances, "failed")
	require.NotContains(t, repo.instances, "healthy")
	repo.failRemove = ""
	require.Equal(t, 1, target.SweepExpired())
	require.Empty(t, repo.candidates)
	require.Empty(t, repo.instances)
	require.Empty(t, target.locks.items)
}
