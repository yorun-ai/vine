package lockserver

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func TestDefaultEmbeddedLeaseOwnershipAndExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		flags := &flag.Flag{SeedHubData: "{}"}
		flags.Normalize(false)
		server := &Server{Flag: flags}
		server.DIInit()
		defer server.AfterAppStop()
		require.True(t, server.Acquire("key", "a", 1000))
		assert.False(t, server.Acquire("key", "b", 1000))
		assert.False(t, server.Renew("key", "b", 2000))
		assert.False(t, server.Release("key", "b"))
		time.Sleep(600 * time.Millisecond)
		assert.False(t, server.Acquire("key", "a", 1000))
		time.Sleep(500 * time.Millisecond)
		require.True(t, server.Acquire("key", "b", 1000))
		assert.False(t, server.Release("key", "a"))
		assert.True(t, server.Renew("key", "b", 2000))
		time.Sleep(1500 * time.Millisecond)
		assert.True(t, server.Release("key", "b"))
		assert.False(t, server.Release("key", "b"))
	})
}

func TestNonEmbeddedModesDoNotStartHubStore(t *testing.T) {
	for _, mode := range []string{"redis", "disable"} {
		t.Run(mode, func(t *testing.T) {
			server := &Server{Flag: &flag.Flag{LockMode: mode}}
			server.DIInit()
			t.Cleanup(server.AfterAppStop)
			assert.Nil(t, server.items)
			assert.Nil(t, server.stop)
			assert.Panics(t, func() { server.Acquire("key", "token", 1000) })
			assert.Panics(t, func() { server.Renew("key", "token", 1000) })
			assert.Panics(t, func() { server.Release("key", "token") })
		})
	}
}
