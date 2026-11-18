package core

import (
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
)

type portalLeaseRepo struct {
	PortalInstanceRepo
	candidates []uuid.UUID
	instances  map[uuid.UUID]*PortalInstance
	failRemove uuid.UUID
}

func (r *portalLeaseRepo) ListExpiredLeases() []uuid.UUID {
	return slices.Clone(r.candidates)
}

func (r *portalLeaseRepo) GetById(id uuid.UUID) (*PortalInstance, bool) {
	instance, ok := r.instances[id]
	return instance, ok
}

func (r *portalLeaseRepo) Remove(id uuid.UUID) {
	if id == r.failRemove {
		panic("portal removal failed")
	}
	delete(r.instances, id)
	r.candidates = slices.DeleteFunc(r.candidates, func(candidate uuid.UUID) bool { return candidate == id })
}

func TestPortalSweepRechecksRenewedCandidate(t *testing.T) {
	repo := &portalLeaseRepo{
		candidates: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-0000000000d2")},
		instances: map[uuid.UUID]*PortalInstance{
			uuid.MustParse("00000000-0000-0000-0000-0000000000d2"): {InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d2"), ExpiresAt: time.Now().Add(time.Minute)},
		},
	}
	target := &PortalInstanceCore{PortalInstanceRepo: repo}
	require.Zero(t, target.SweepExpired())
	require.Contains(t, repo.instances, uuid.MustParse("00000000-0000-0000-0000-0000000000d2"))
}

func TestPortalSweepRetriesFailureAndContinuesOtherInstances(t *testing.T) {
	repo := &portalLeaseRepo{
		candidates: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-0000000000d3"), uuid.MustParse("00000000-0000-0000-0000-0000000000d4")},
		instances: map[uuid.UUID]*PortalInstance{
			uuid.MustParse("00000000-0000-0000-0000-0000000000d3"): {InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d3"), ExpiresAt: time.Now().Add(-time.Minute)},
			uuid.MustParse("00000000-0000-0000-0000-0000000000d4"): {InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d4"), ExpiresAt: time.Now().Add(-time.Minute)},
		},
		failRemove: uuid.MustParse("00000000-0000-0000-0000-0000000000d3"),
	}
	target := &PortalInstanceCore{PortalInstanceRepo: repo}
	require.Equal(t, 1, target.SweepExpired())
	require.Equal(t, []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-0000000000d3")}, repo.candidates)
	require.Contains(t, repo.instances, uuid.MustParse("00000000-0000-0000-0000-0000000000d3"))
	require.NotContains(t, repo.instances, uuid.MustParse("00000000-0000-0000-0000-0000000000d4"))
	repo.failRemove = uuid.UUID{}
	require.Equal(t, 1, target.SweepExpired())
	require.Empty(t, repo.candidates)
	require.Empty(t, repo.instances)
	require.Empty(t, target.locks.items)
}
