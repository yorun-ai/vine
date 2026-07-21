package embedded

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreZSetPopRangeByScoreReturnsSortedMembers(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.ZAdd("registry:leases", 30, "app-c:instance-3")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")
	store.ZAdd("registry:leases", 10, "app-a:instance-1")
	store.ZAdd("registry:leases", 50, "app-d:instance-4")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 0)

	assert.Equal(t, []string{
		"app-a:instance-1",
		"app-b:instance-2",
		"app-c:instance-3",
	}, members)
}

func TestStoreZSetUpdatesScoreAndLimits(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.ZAdd("registry:leases", 30, "app-a:instance-1")
	store.ZAdd("registry:leases", 5, "app-a:instance-1")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 1)

	assert.Equal(t, []string{"app-a:instance-1"}, members)
}

func TestStoreZSetRemovesMembers(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.ZAdd("registry:leases", 10, "app-a:instance-1")

	assert.True(t, store.ZRem("registry:leases", "app-a:instance-1"))
	assert.False(t, store.ZRem("registry:leases", "app-a:instance-1"))
	assert.Empty(t, store.ZPopRangeByScore("registry:leases", 0, 30, 0))
}

func TestStoreZSetPopsRangeByScore(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.ZAdd("registry:leases", 30, "app-c:instance-3")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")
	store.ZAdd("registry:leases", 10, "app-a:instance-1")
	store.ZAdd("registry:leases", 50, "app-d:instance-4")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 2)

	assert.Equal(t, []string{"app-a:instance-1", "app-b:instance-2"}, members)
	assert.Equal(t, []string{"app-c:instance-3", "app-d:instance-4"}, store.ZPopRangeByScore("registry:leases", 0, 100, 0))
}
