package repo

import (
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func newTestPortalInstanceRepo(t *testing.T, inproc bool) (*PortalInstanceRepo, *watchserver.Server) {
	t.Helper()

	_, watchServer := newTestRegistryRepo(t, inproc)
	return &PortalInstanceRepo{
		WatchServer: watchServer,
		InprocFlag:  &internalapp.InternalInprocFlag{Enabled: inproc},
	}, watchServer
}

func TestPortalInstanceRepoSavesAndListsInstances(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})

	instance, ok := repo.GetById(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	require.True(t, ok)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), instance.InstanceId)
	assert.Equal(t, "1.2.3", instance.Version)
	assert.Equal(t, now.Add(hubPortalRegistryLeaseTTL), instance.ExpiresAt)

	instances := repo.List()
	require.Len(t, instances, 1)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), instances[0].InstanceId)
}

func TestPortalInstanceRepoHeartbeatKeepsLease(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})

	now = now.Add(20 * time.Second)
	assert.True(t, repo.Keep(uuid.MustParse("00000000-0000-0000-0000-000000000001")))

	instance, ok := repo.GetById(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	require.True(t, ok)
	assert.Equal(t, now.Add(hubPortalRegistryLeaseTTL), instance.ExpiresAt)
	assert.Equal(t, []string{watched.FormatPortalInstanceKey(uuid.MustParse("00000000-0000-0000-0000-000000000001"))}, testServer.Scan(watched.FormatPortalInstancePattern()))

	// An unknown instance cannot be kept alive.
	assert.False(t, repo.Keep(uuid.MustParse("00000000-0000-0000-0000-0000000000d7")))
}

func TestPortalInstanceRepoSweepsExpiredInstances(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})

	now = now.Add(hubPortalRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.ListExpiredLeases())

	now = now.Add(2 * time.Second)
	assert.Equal(t, []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001")}, repo.ListExpiredLeases())

	repo.Remove(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	_, ok := repo.GetById(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	assert.False(t, ok)
	assert.Empty(t, repo.List())
	assert.Empty(t, repo.ListExpiredLeases())
}

func TestPortalInstanceRepoRemovalDropsLease(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	setTimeNowForTest(t, func() time.Time { return time.Now().UTC().Round(0) })

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})
	repo.Remove(uuid.MustParse("00000000-0000-0000-0000-000000000001"))

	// A removed instance must not be reported as expired later, otherwise the
	// sweeper would try to remove it twice.
	now := time.Now().UTC().Round(0).Add(2 * hubPortalRegistryLeaseTTL)
	setTimeNowForTest(t, func() time.Time { return now })
	assert.Empty(t, repo.ListExpiredLeases())
	assert.Empty(t, testServer.Scan(watched.FormatPortalInstancePattern()))
}

func TestPortalInstanceRepoInprocModeSkipsLeases(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, true)

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})

	instance, ok := repo.GetById(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	require.True(t, ok)
	// Standalone Portal registers without a heartbeat and without a lease, so
	// the sweeper has nothing to collect.
	assert.True(t, instance.ExpiresAt.IsZero())
	assert.True(t, repo.Keep(uuid.MustParse("00000000-0000-0000-0000-000000000001")))
	assert.Empty(t, repo.ListExpiredLeases())
}

func TestPortalInstanceRepoRecordsJSONShape(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Version: "1.2.3"})

	value, ok := testServer.Get(watched.FormatPortalInstanceKey(uuid.MustParse("00000000-0000-0000-0000-000000000001")))
	require.True(t, ok)
	record := vcode.MustUnmarshalJsonS[map[string]any](value)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", record["instanceId"])
	assert.Equal(t, "1.2.3", record["version"])
	assert.NotEmpty(t, record["expiresAt"])
}

func TestPortalInstanceSurvivesStorageTTLAndCleansByLease(t *testing.T) {
	repo, server := newTestPortalInstanceRepo(t, false)
	now := time.Now().Add(-4 * time.Minute)
	setTimeNowForTest(t, func() time.Time { return now })
	repo.Save(&core.PortalInstance{InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5")})
	now = time.Now()
	_, ok := repo.GetById(uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))
	require.True(t, ok)
	require.Equal(t, -1, server.TTL(watched.FormatPortalInstanceKey(uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))))
	require.Empty(t, repo.List())
	require.Equal(t, []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-0000000000d5")}, repo.ListExpiredLeases())
	require.Equal(t, []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-0000000000d5")}, repo.ListExpiredLeases())
	target := &core.PortalInstanceCore{PortalInstanceRepo: repo}
	require.Equal(t, 1, target.SweepExpired())
	require.Empty(t, repo.ListExpiredLeases())
}
