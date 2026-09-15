package lock

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

func TestApplicationAndUniversalLockScopes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := new(testClient)
		ctx := context.Background()
		appA := NewLocker(backend, ctx, "a")
		appB := NewLocker(backend, ctx, "b")
		universalA := NewUniversalLocker(backend, ctx)
		universalB := NewUniversalLocker(backend, ctx)

		first := appA.Lock("job")
		second := appB.Lock("job")
		shared := universalA.Lock("job")
		_, ok := appA.TryLock("job")
		require.False(t, ok)
		_, ok = universalB.TryLock("job")
		require.False(t, ok)
		require.Equal(t, []string{"app:a:job", "app:b:job", "universal:job", "app:a:job", "universal:job"}, backend.keys)
		shared.Unlock()
		next, ok := universalB.TryLock("job")
		require.True(t, ok)
		next.Unlock()
		first.Unlock()
		second.Unlock()
	})
}

func TestUniversalWithContextKeepsScopeAndOriginalContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := new(testClient)
		original := NewUniversalLocker(backend, context.Background())
		ctx, cancel := context.WithCancel(context.Background())
		bound := original.WithContext(ctx)
		lease := bound.Lock("job")
		require.Equal(t, "universal:job", backend.keys[0])
		lease.Unlock()
		cancel()
		err := recoveredError(func() { bound.TryLock("job") })
		require.Equal(t, ex.InvocationCancelled, err.Code())
		original.Lock("job").Unlock()
	})
}
