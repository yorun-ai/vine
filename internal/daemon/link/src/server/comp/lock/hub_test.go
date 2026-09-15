package lock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmbeddedRoutesToHub(t *testing.T) {
	for _, inproc := range []bool{false, true} {
		t.Run(map[bool]string{false: "network", true: "inproc"}[inproc], func(t *testing.T) {
			locker, hub := newTestLocker(t, "embedded", "", inproc)
			ctx := lockContext(context.Background())
			assert.True(t, locker.Acquire(ctx, "key", "token", 1000))
			assert.True(t, locker.Renew(ctx, "key", "token", 1000))
			assert.True(t, locker.Release(ctx, "key", "token"))
			assert.Equal(t, []string{"acquire", "renew", "release"}, hub.calls)
			assert.IsType(t, new(_HubLocker), locker.impl)
		})
	}
}
