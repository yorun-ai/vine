package embedded

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCursorReturnsNextBatchAndCleansWhenDone(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.Set("config:feature-a", "a")
	store.Set("config:feature-b", "b")
	store.Set("config:feature-c", "c")

	keys, cursor, err := store.ScanKeys(0, "config:*", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"config:feature-a", "config:feature-b"}, keys)
	require.NotZero(t, cursor)

	keys, nextCursor, err := store.ScanKeys(cursor, "config:*", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"config:feature-c"}, keys)
	assert.Zero(t, nextCursor)
	assert.Empty(t, store.scans)
}

func TestScanCursorExpiresAndIsCleaned(t *testing.T) {
	now := time.Now()
	SetTimeNowForTest(t, func() time.Time {
		return now
	})
	store := NewStore("", false, "test-server-password")

	store.Set("config:feature-a", "a")
	store.Set("config:feature-b", "b")

	_, cursor, err := store.ScanKeys(0, "config:*", 1)
	require.NoError(t, err)
	require.NotZero(t, cursor)

	now = now.Add(scanCursorTTL + time.Second)
	_, _, err = store.ScanKeys(cursor, "config:*", 1)
	assert.EqualError(t, err, "invalid scan cursor")
	assert.Empty(t, store.scans)
}

func TestScanCursorRejectsDifferentMatchPattern(t *testing.T) {
	store := NewStore("", false, "test-server-password")

	store.Set("portal:cert:production", "secret")
	store.Set("rpc:demo.Service:endpoint:demo.app:instance-1", "endpoint-1")
	store.Set("rpc:demo.Service:endpoint:demo.app:instance-2", "endpoint-2")

	_, cursor, err := store.ScanKeys(0, "*", 1)
	require.NoError(t, err)
	require.NotZero(t, cursor)

	_, _, err = store.ScanKeys(cursor, "rpc:demo.Service:endpoint:*", 1000)
	assert.EqualError(t, err, "invalid scan cursor")
}
