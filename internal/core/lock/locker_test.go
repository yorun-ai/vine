package lock

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
)

func recoveredError(call func()) (err ex.Error) {
	defer func() { err = ex.RecoverExecution(recover()) }()
	call()
	return nil
}

func TestTryLockAttemptsOnceAndPrefixesKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		locker := NewLocker(client, context.Background(), "app")
		lease, ok := locker.TryLock("job")
		require.True(t, ok)
		other, ok := locker.TryLock("job")
		require.False(t, ok)
		require.Nil(t, other)
		require.Equal(t, []string{"app:app:job", "app:app:job"}, client.keys)
		require.NotEqual(t, client.attempts[0], client.attempts[1])
		require.Equal(t, 30*time.Second, client.ttl)
		lease.Unlock()
	})
}

func TestLockWaitsForRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := new(testClient)
		locker := NewLocker(backend, context.Background(), "app")
		first := locker.Lock("job")
		next := make(chan *Lock, 1)
		go func() { next <- locker.Lock("job") }()
		synctest.Wait()
		require.Empty(t, next)
		first.Unlock()
		time.Sleep(300 * time.Millisecond)
		second := <-next
		require.NotEqual(t, first.token, second.token)
		second.Unlock()
	})
}

func TestLockCancellationAndTimeout(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			backend := new(testClient)
			backend.acquire = func(context.Context, string, string, time.Duration) bool { return false }
			ctx, cancel := context.WithCancel(context.Background())
			if timeout {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			}
			defer cancel()
			locker := NewLocker(backend, ctx, "app")
			result := make(chan ex.Error, 1)
			go func() { result <- recoveredError(func() { locker.Lock("job") }) }()
			synctest.Wait()
			if timeout {
				time.Sleep(100 * time.Millisecond)
			} else {
				cancel()
			}
			err := <-result
			require.NotNil(t, err)
			if timeout {
				require.Equal(t, ex.InvocationTimeout, err.Code())
			} else {
				require.Equal(t, ex.InvocationCancelled, err.Code())
			}
			require.ErrorIs(t, err, ctx.Err())
		})
	}
}

func TestAcquisitionFailureIsNotRetried(t *testing.T) {
	for _, wait := range []bool{false, true} {
		client := new(testClient)
		cause := ex.New(ex.ServiceUnavailable, "offline")
		client.acquire = func(context.Context, string, string, time.Duration) bool { panic(cause) }
		locker := NewLocker(client, context.Background(), "app")
		err := recoveredError(func() {
			if wait {
				locker.Lock("job")
			} else {
				locker.TryLock("job")
			}
		})
		require.Equal(t, ex.ServiceUnavailable, err.Code())
		require.Len(t, client.attempts, 1)
	}
}

func TestSlowAcquisitionDoesNotGrantFreshTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		client.acquire = func(context.Context, string, string, time.Duration) bool {
			time.Sleep(time.Second)
			return true
		}
		err := recoveredError(func() { NewLocker(client, context.Background(), "app").TryLock("job", WithTTL(time.Second)) })
		require.NotNil(t, err)
		require.Equal(t, ex.InvocationTimeout, err.Code())
	})
}

func TestTTLValidationAndPrecision(t *testing.T) {
	for _, ttl := range []time.Duration{-time.Second, 0, time.Microsecond} {
		err := recoveredError(func() { WithTTL(ttl) })
		require.Equal(t, ex.InvalidRequest, err.Code())
	}
	synctest.Test(t, func(t *testing.T) {
		client := new(testClient)
		lease := NewLocker(client, context.Background(), "app").Lock("job", WithTTL(1500*time.Microsecond))
		require.Equal(t, time.Millisecond, client.ttl)
		lease.Unlock()
	})
}

type requestTransport func(*http.Request) (*http.Response, error)

func (f requestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestLinkCallsCarryContextAndBoundedDeadline(t *testing.T) {
	for _, operation := range []string{"acquire", "wait", "renew", "release"} {
		t.Run(operation, func(t *testing.T) {
			marker := new(int)
			ctx := context.WithValue(t.Context(), marker, "request")
			ttl := 30 * time.Millisecond
			bound := time.Now().Add(ttl)
			calls := 0
			rpc := rpcclient.New(rpcclient.Option{
				Context:        meta.NewContext(ctx, meta.InitialTrace(), nil, meta.NewAbsentActor()),
				ClientApp:      meta.MustNewAppWithRandomId("test.lock", "v1.0.0"),
				Logger:         logger.New("test.lock"),
				ServerEndpoint: "http://localhost:7079",
				Transport: requestTransport(func(request *http.Request) (*http.Response, error) {
					calls++
					assert.Equal(t, "request", request.Context().Value(marker))
					deadline, ok := request.Context().Deadline()
					assert.True(t, ok)
					if operation == "renew" || operation == "release" {
						assert.False(t, deadline.After(bound))
					} else if operation == "acquire" || operation == "wait" {
						assert.LessOrEqual(t, time.Until(deadline), time.Second)
					}
					return nil, context.Canceled
				}),
			})
			link := skeled.NewLockServiceClient(skeled.NewLockServiceClientER(rpc))
			if operation == "acquire" {
				require.NotNil(t, recoveredError(func() { NewLocker(link, ctx, "app").TryLock("job", WithTTL(time.Second)) }))
			} else if operation == "wait" {
				require.NotNil(t, recoveredError(func() { NewLocker(link, ctx, "app").Lock("job", WithTTL(time.Second)) }))
			} else {
				lease := newLock(link, ctx, "app:job", "token", ttl, bound)
				defer lease.cancel(context.Canceled)
				if operation == "renew" {
					<-lease.Context().Done()
					require.True(t, lease.IsBroken())
				} else {
					require.NotNil(t, recoveredError(func() { lease.TryUnlock() }))
				}
			}
			require.Equal(t, 1, calls)
		})
	}
}
