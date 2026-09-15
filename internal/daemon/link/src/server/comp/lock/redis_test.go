package lock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

type redisHook struct {
	process func(context.Context, goredis.Cmder) error
}

func (h redisHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unexpected network connection")
	}
}

func (h redisHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return h.process
}

func (h redisHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return next
}

func TestRedisRoutesDirectlyAndPreservesRequestContext(t *testing.T) {
	locker, hub := newTestLocker(t, "redis", "rediss://user:secret@redis.example:6380/2", false)
	options := locker.impl.(*_RedisLocker).client.Options()
	assert.Equal(t, "redis.example:6380", options.Addr)
	assert.Equal(t, "user", options.Username)
	assert.Equal(t, "secret", options.Password)
	assert.Equal(t, 2, options.DB)
	assert.NotNil(t, options.TLSConfig)
	assert.True(t, options.ContextTimeoutEnabled)
	ctx := lockContext(context.Background())
	var commands [][]any
	locker.impl.(*_RedisLocker).client.AddHook(redisHook{process: func(got context.Context, cmd goredis.Cmder) error {
		assert.Same(t, ctx, got)
		commands = append(commands, cmd.Args())
		cmd.(*goredis.Cmd).SetVal(int64(1))
		return nil
	}})
	assert.True(t, locker.Acquire(ctx, "app:key", "token", 1234))
	assert.True(t, locker.Renew(ctx, "app:key", "token", 2345))
	assert.True(t, locker.Release(ctx, "app:key", "token"))
	assert.Equal(t, [][]any{
		{"eval", acquireScript, 1, "vine:core:lock:app:key", "token", 1234},
		{"eval", renewScript, 1, "vine:core:lock:app:key", "token", 2345},
		{"eval", releaseScript, 1, "vine:core:lock:app:key", "token"},
	}, commands)
	assert.Empty(t, hub.calls)
}

func TestRedisFailuresDoNotFallBackToHub(t *testing.T) {
	for _, failure := range []bool{false, true} {
		locker, hub := newTestLocker(t, "redis", "redis://localhost:6379", false)
		locker.impl.(*_RedisLocker).client.AddHook(redisHook{process: func(ctx context.Context, cmd goredis.Cmder) error {
			if failure {
				return errors.New("Redis unavailable")
			}
			cmd.(*goredis.Cmd).SetVal(int64(0))
			return nil
		}})
		call := func() bool { return locker.Acquire(lockContext(context.Background()), "key", "token", 1000) }
		if failure {
			require.Panics(t, func() { call() })
		} else {
			assert.False(t, call())
		}
		assert.Empty(t, hub.calls)
	}
}

func TestRedisRejectsInvalidLeaseBeforeSending(t *testing.T) {
	locker, _ := newTestLocker(t, "redis", "redis://localhost", false)
	ctx := lockContext(context.Background())
	assert.Panics(t, func() { locker.Acquire(ctx, "", "token", 1000) })
	assert.Panics(t, func() { locker.Acquire(ctx, "key", "", 1000) })
	assert.Panics(t, func() { locker.Acquire(ctx, "key", "token", 0) })
	assert.Panics(t, func() { locker.Renew(ctx, "key", "token", -1) })
}

func TestRedisOperationErrorsPreserveCodeAndCause(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		code  ex.Code
	}{
		{"connection refused", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, ex.ServiceUnavailable},
		{"Redis rejection", errors.New("NOPERM operation denied"), ex.ServiceUnavailable},
		{"cancelled", fmt.Errorf("request failed: %w", context.Canceled), ex.InvocationCancelled},
		{"deadline exceeded", fmt.Errorf("request failed: %w", context.DeadlineExceeded), ex.InvocationTimeout},
		{"socket timeout", &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}, ex.InvocationTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			locker, hub := newTestLocker(t, "redis", "redis://localhost", false)
			locker.impl.(*_RedisLocker).client.AddHook(redisHook{process: func(context.Context, goredis.Cmder) error {
				return tc.cause
			}})
			ctx := lockContext(context.Background())
			for _, operation := range []struct {
				name string
				call func()
			}{
				{"acquire", func() { locker.Acquire(ctx, "key", "token", 1000) }},
				{"renew", func() { locker.Renew(ctx, "key", "token", 1000) }},
				{"release", func() { locker.Release(ctx, "key", "token") }},
			} {
				t.Run(operation.name, func(t *testing.T) {
					var recovered ex.Error
					func() {
						defer func() { recovered = ex.RecoverExecution(recover()) }()
						operation.call()
					}()
					require.NotNil(t, recovered)
					assert.Equal(t, tc.code, recovered.Code())
					assert.ErrorIs(t, recovered, tc.cause)
				})
			}
			assert.Empty(t, hub.calls)
		})
	}
}
