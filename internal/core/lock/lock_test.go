package lock

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

func TestBothAcquisitionMethodsRenewAutomatically(t *testing.T) {
	for _, wait := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			client := new(testClient)
			locker := NewLocker(client, context.Background(), "app")
			var lease *Lock
			if wait {
				lease = locker.Lock("job", WithTTL(time.Second))
			} else {
				var ok bool
				lease, ok = locker.TryLock("job", WithTTL(time.Second))
				require.True(t, ok)
			}
			time.Sleep(3 * time.Second)
			synctest.Wait()
			require.False(t, lease.IsBroken())
			require.NoError(t, lease.Context().Err())
			require.Greater(t, client.renewals, 1)
			lease.Unlock()
			require.False(t, lease.IsBroken())
			require.ErrorIs(t, lease.Context().Err(), context.Canceled)
		})
	}
}

func TestRenewalFailureBreaksLease(t *testing.T) {
	for _, backendFailure := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			client := new(testClient)
			client.renew = func(context.Context, string, string, time.Duration) bool {
				if backendFailure {
					ex.PanicNew(ex.ServiceUnavailable, "offline")
				}
				return false
			}
			lease := NewLocker(client, context.Background(), "app").Lock("job", WithTTL(time.Second))
			time.Sleep(400 * time.Millisecond)
			synctest.Wait()
			require.True(t, lease.IsBroken())
			cause, ok := context.Cause(lease.Context()).(ex.Error)
			require.True(t, ok)
			if backendFailure {
				require.Equal(t, ex.ServiceUnavailable, cause.Code())
			} else {
				require.Equal(t, ex.OperationFailed, cause.Code())
			}
			require.False(t, lease.TryUnlock())
			require.Zero(t, client.releases)
		})
	}
}

func TestBlockedRenewalCannotExtendExpiredLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		client.renew = func(ctx context.Context, _ string, _ string, _ time.Duration) bool {
			// Simulate a late successful response despite cancellation.
			time.Sleep(2 * time.Second)
			return true
		}
		lease := NewLocker(client, context.Background(), "app").Lock("job", WithTTL(time.Second))
		time.Sleep(1100 * time.Millisecond)
		require.True(t, lease.IsBroken())
		require.Error(t, lease.Context().Err())
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.True(t, lease.IsBroken())
	})
}

func TestParentCancellationStopsRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		ctx, cancel := context.WithCancel(context.Background())
		lease := NewLocker(client, context.Background(), "app").WithContext(ctx).Lock("job")
		cancel()
		synctest.Wait()
		require.True(t, lease.IsBroken())
		require.ErrorIs(t, context.Cause(lease.Context()), context.Canceled)
		require.Zero(t, client.renewals)
	})
}

func TestConcurrentReleaseHappensOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		lease := NewLocker(client, context.Background(), "app").Lock("job")
		var success atomic.Int32
		var group sync.WaitGroup
		for range 10 {
			group.Go(func() {
				if lease.TryUnlock() {
					success.Add(1)
				}
			})
		}
		group.Wait()
		require.Equal(t, int32(1), success.Load())
		require.Equal(t, 1, client.releases)
		require.False(t, lease.IsBroken())
	})
}

func TestReleaseFailureBreaksLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		client.release = func(context.Context, string, string) bool {
			ex.PanicNew(ex.ServiceUnavailable, "offline")
			return false
		}
		lease := NewLocker(client, context.Background(), "app").Lock("job")
		err := recoveredError(func() { lease.TryUnlock() })
		require.NotNil(t, err)
		assert.Equal(t, ex.ServiceUnavailable, err.Code())
		assert.True(t, lease.IsBroken())
		assert.False(t, lease.TryUnlock())
	})
}

func TestReleaseWaitsForOngoingRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := new(testClient)
		renewing := make(chan struct{})
		finish := make(chan struct{})
		backend.renew = func(context.Context, string, string, time.Duration) bool {
			close(renewing)
			<-finish
			return true
		}
		lease := NewLocker(backend, context.Background(), "app").Lock("job", WithTTL(time.Second))
		<-renewing
		result := make(chan bool, 1)
		go func() { result <- lease.TryUnlock() }()
		synctest.Wait()
		require.Empty(t, result)
		close(finish)
		require.True(t, <-result)
		require.False(t, lease.IsBroken())
		time.Sleep(2 * time.Second)
		require.Equal(t, 1, backend.renewals)
		require.Equal(t, 1, backend.releases)
	})
}

func TestLateReleaseDoesNotReviveExpiredLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := new(testClient)
		backend.release = func(context.Context, string, string) bool {
			time.Sleep(2 * time.Second)
			return true
		}
		lease := NewLocker(backend, context.Background(), "app").Lock("job", WithTTL(time.Second))
		result := make(chan bool, 1)
		go func() { result <- lease.TryUnlock() }()
		synctest.Wait()
		time.Sleep(1100 * time.Millisecond)
		require.False(t, <-result)
		require.True(t, lease.IsBroken())
		time.Sleep(time.Second)
		synctest.Wait()
		require.True(t, lease.IsBroken())
		require.False(t, lease.TryUnlock())
		require.Equal(t, 1, backend.releases)
	})
}
