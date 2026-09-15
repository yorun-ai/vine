package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func newTestPortalInstanceRepo(t *testing.T, inproc bool) (*WatchPortalInstanceRepo, *watchserver.Server) {
	t.Helper()

	_, watchServer := newTestRegistryRepo(t, inproc)
	return &WatchPortalInstanceRepo{
		WatchServer: watchServer,
		InprocFlag:  &internalapp.InternalInprocFlag{Enabled: inproc},
	}, watchServer
}

func TestPortalInstanceRepoSavesAndListsInstances(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})

	instance, ok := repo.GetPortalInstance("instance-1")
	require.True(t, ok)
	assert.Equal(t, "instance-1", instance.InstanceId)
	assert.Equal(t, "1.2.3", instance.Version)
	assert.Equal(t, now.Add(hubPortalRegistryLeaseTTL), instance.ExpiresAt)

	instances := repo.ListPortalInstances()
	require.Len(t, instances, 1)
	assert.Equal(t, "instance-1", instances[0].InstanceId)
}

func TestPortalInstanceRepoHeartbeatKeepsLease(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})

	now = now.Add(20 * time.Second)
	assert.True(t, repo.KeepPortalInstance("instance-1"))

	instance, ok := repo.GetPortalInstance("instance-1")
	require.True(t, ok)
	assert.Equal(t, now.Add(hubPortalRegistryLeaseTTL), instance.ExpiresAt)
	assert.Equal(t, []string{watched.FormatPortalInstanceKey("instance-1")}, testServer.Scan(watched.FormatPortalInstancePattern()))

	// An unknown instance cannot be kept alive.
	assert.False(t, repo.KeepPortalInstance("instance-unknown"))
}

func TestPortalInstanceRepoSweepsExpiredInstances(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})

	now = now.Add(hubPortalRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.PopExpiredPortalLeases())

	now = now.Add(2 * time.Second)
	assert.Equal(t, []string{"instance-1"}, repo.PopExpiredPortalLeases())

	repo.RemovePortalInstance("instance-1")
	_, ok := repo.GetPortalInstance("instance-1")
	assert.False(t, ok)
	assert.Empty(t, repo.ListPortalInstances())
	assert.Empty(t, repo.PopExpiredPortalLeases())
}

func TestPortalInstanceRepoRemovalDropsLease(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	setTimeNowForTest(t, func() time.Time { return time.Now().UTC().Round(0) })

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})
	repo.RemovePortalInstance("instance-1")

	// A removed instance must not be reported as expired later, otherwise the
	// sweeper would try to remove it twice.
	now := time.Now().UTC().Round(0).Add(2 * hubPortalRegistryLeaseTTL)
	setTimeNowForTest(t, func() time.Time { return now })
	assert.Empty(t, repo.PopExpiredPortalLeases())
	assert.Empty(t, testServer.Scan(watched.FormatPortalInstancePattern()))
}

func TestPortalInstanceRepoInprocModeSkipsLeases(t *testing.T) {
	repo, _ := newTestPortalInstanceRepo(t, true)

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})

	instance, ok := repo.GetPortalInstance("instance-1")
	require.True(t, ok)
	// Standalone Portal registers without a heartbeat and without a lease, so
	// the sweeper has nothing to collect.
	assert.True(t, instance.ExpiresAt.IsZero())
	assert.True(t, repo.KeepPortalInstance("instance-1"))
	assert.Empty(t, repo.PopExpiredPortalLeases())
}

func TestPortalInstanceRepoRecordsJSONShape(t *testing.T) {
	repo, testServer := newTestPortalInstanceRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	repo.SavePortalInstance(&core.PortalInstance{InstanceId: "instance-1", Version: "1.2.3"})

	value, ok := testServer.Get(watched.FormatPortalInstanceKey("instance-1"))
	require.True(t, ok)
	record := vcode.MustUnmarshalJsonS[map[string]any](value)
	assert.Equal(t, "instance-1", record["instanceId"])
	assert.Equal(t, "1.2.3", record["version"])
	assert.NotEmpty(t, record["expiresAt"])
}
