package lock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisabledAndMissingModeRejectOperations(t *testing.T) {
	for _, mode := range []string{"disable", ""} {
		t.Run(mode, func(t *testing.T) {
			locker, hub := newTestLocker(t, mode, "", false)
			ctx := lockContext(context.Background())
			assert.Panics(t, func() { locker.Acquire(ctx, "key", "token", 1000) })
			assert.Panics(t, func() { locker.Renew(ctx, "key", "token", 1000) })
			assert.Panics(t, func() { locker.Release(ctx, "key", "token") })
			assert.Empty(t, hub.calls)
			assert.Nil(t, locker.impl)
		})
	}
}
