package redis

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/di"
)

type _TestLockerRedis struct {
	Redis
}

func (*_TestLockerRedis) InitOption(option *Option) {
	option.Endpoint = "redis://127.0.0.1:6379"
}

func (*_TestLockerRedis) InitLockers(add TypeAdder) {
	add(reflect.TypeFor[*_TestUserLocker]())
}

type _TestUserLocker struct {
	Locker
}

func (*_TestUserLocker) KeyPrefix() string {
	return "lock:user"
}

type _TestLockerConsumer struct {
	Locker *_TestUserLocker `inject:""`
}

type _TestDefaultLocker struct {
	Locker
}

type _TestEmptyPrefixLocker struct {
	Locker
}

func (*_TestEmptyPrefixLocker) KeyPrefix() string {
	return ""
}

type _TestLockCmdable struct {
	goredis.Cmdable

	mutex      sync.Mutex
	setNXCalls []_TestSetNXCall
	evalCalls  []_TestEvalCall
	setNXFunc  func(ctx context.Context, key string, value any, expiration time.Duration) *goredis.BoolCmd
	evalFunc   func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd
}

type _TestSetNXCall struct {
	ctx        context.Context
	key        string
	value      any
	expiration time.Duration
}

type _TestEvalCall struct {
	ctx    context.Context
	script string
	keys   []string
	args   []any
}

func (c *_TestLockCmdable) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *goredis.BoolCmd {
	c.mutex.Lock()
	c.setNXCalls = append(c.setNXCalls, _TestSetNXCall{
		ctx:        ctx,
		key:        key,
		value:      value,
		expiration: expiration,
	})
	c.mutex.Unlock()
	return c.setNXFunc(ctx, key, value, expiration)
}

func (c *_TestLockCmdable) Eval(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
	c.mutex.Lock()
	c.evalCalls = append(c.evalCalls, _TestEvalCall{
		ctx:    ctx,
		script: script,
		keys:   append([]string(nil), keys...),
		args:   append([]any(nil), args...),
	})
	c.mutex.Unlock()
	return c.evalFunc(ctx, script, keys, args...)
}

func newTestLockCmdable() *_TestLockCmdable {
	return &_TestLockCmdable{
		setNXFunc: func(ctx context.Context, key string, value any, expiration time.Duration) *goredis.BoolCmd {
			return goredis.NewBoolResult(true, nil)
		},
		evalFunc: func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
			return goredis.NewCmdResult(int64(1), nil)
		},
	}
}

func TestRedisNewLockerReturnsHandle(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}
	ctx := context.Background()

	locker := redis.NewLocker(ctx, "lock:user")

	require.NotNil(t, locker)
	assert.Equal(t, ctx, locker.ctx)
	assert.Equal(t, "lock:user", locker.keyPrefix)
	assert.Equal(t, cmdable, locker.cmdable)
}

func TestRedisNewLockerByTypeReturnsTypedHandle(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}
	ctx := context.Background()

	locker := redis.NewLockerByType(reflect.TypeFor[*_TestUserLocker](), ctx).(*_TestUserLocker)

	require.NotNil(t, locker)
	assert.Equal(t, ctx, locker.ctx)
	assert.Equal(t, "lock:user", locker.keyPrefix)
	assert.Equal(t, cmdable, locker.cmdable)
}

func TestRedisNewLockerByTypeUsesDefaultTypePrefixWhenNotOverridden(t *testing.T) {
	cmdable := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = cmdable.Close()
	})
	redis := &Redis{Cmdable: cmdable}

	locker := redis.NewLockerByType(reflect.TypeFor[*_TestDefaultLocker](), context.Background()).(*_TestDefaultLocker)

	assert.Equal(t, "go.yorun.ai_vine_internal_infra_redis._TestDefaultLocker", locker.keyPrefix)
}

func TestRedisManagerBindProvidesLocker(t *testing.T) {
	original := newRedisClient
	t.Cleanup(func() {
		newRedisClient = original
	})

	newRedisClient = func(opt *Option) *goredis.Client {
		return goredis.NewClient(endpointOptions(opt.Endpoint))
	}

	component := new(_TestLockerRedis)
	manager := new(RedisManager)
	manager.InitComponent(component)
	t.Cleanup(manager.AfterAppStop)

	injector := di.NewInjector(func(b *di.Binder) {
		b.Bind(reflect.TypeFor[context.Context]()).ToInstance(context.Background())
		manager.Bind(b)
		b.Bind(reflect.TypeFor[*_TestLockerConsumer]()).In(di.TransientScope)
	})

	consumer := injector.Get(reflect.TypeFor[*_TestLockerConsumer]()).Interface().(*_TestLockerConsumer)
	require.NotNil(t, consumer.Locker)
	assert.Equal(t, "lock:user", consumer.Locker.keyPrefix)
	require.NotNil(t, consumer.Locker.ctx)
	require.NotNil(t, consumer.Locker.cmdable)
}

func TestInstantiateLockerUsesDefaultTypePrefixWhenNotOverridden(t *testing.T) {
	manager := &RedisManager{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	manager.component = &Redis{Cmdable: manager.client}
	t.Cleanup(func() {
		_ = manager.client.Close()
	})

	locker := manager.instantiateLocker(reflect.TypeFor[*_TestDefaultLocker](), context.Background()).(*_TestDefaultLocker)

	assert.Equal(t, "go.yorun.ai_vine_internal_infra_redis._TestDefaultLocker", locker.keyPrefix)
}

func TestInstantiateLockerRequiresNonEmptyOverriddenPrefix(t *testing.T) {
	manager := &RedisManager{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	manager.component = &Redis{Cmdable: manager.client}
	t.Cleanup(func() {
		_ = manager.client.Close()
	})

	assert.Panics(t, func() {
		manager.instantiateLocker(reflect.TypeFor[*_TestEmptyPrefixLocker](), context.Background())
	})
}

func TestLockerLockBuildsNamespacedKeyAndDefaultOption(t *testing.T) {
	cmdable := newTestLockCmdable()
	locker := &Locker{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "lock:user",
	}

	lock, ok := locker.Lock("123")

	require.True(t, ok)
	require.NotNil(t, lock)
	require.Len(t, cmdable.setNXCalls, 1)
	assert.Equal(t, "vine:lock:lock:user:123", cmdable.setNXCalls[0].key)
	assert.Equal(t, lockDefaultTimeout, cmdable.setNXCalls[0].expiration)
	assert.True(t, lock.option.refresh)
	require.NotNil(t, lock.Context())
	assert.NoError(t, lock.Context().Err())
}

func TestLockerLockReturnsFalseWhenContended(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.setNXFunc = func(ctx context.Context, key string, value any, expiration time.Duration) *goredis.BoolCmd {
		return goredis.NewBoolResult(false, nil)
	}
	locker := &Locker{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "lock:user",
	}

	lock, ok := locker.Lock("123")

	require.False(t, ok)
	require.NotNil(t, lock)
	assert.Nil(t, lock.lockCtx)
	assert.Empty(t, lock.token)
	assert.False(t, lock.IsBroken())
	assert.Panics(t, func() {
		lock.Context()
	})
}

func TestLockerLockWithTimeoutDisablesRefreshAndBreaksAfterTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmdable := newTestLockCmdable()
		locker := &Locker{
			ctx:       t.Context(),
			cmdable:   cmdable,
			keyPrefix: "lock:user",
		}

		lock, ok := locker.Lock("123", WithTimeout(20*time.Millisecond))

		require.True(t, ok)
		require.NotNil(t, lock)
		assert.False(t, lock.option.refresh)
		assert.False(t, lock.IsBroken())

		synctest.Sleep(time.Until(lock.currentLeaseDeadline()))

		assert.True(t, lock.IsBroken())
		assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
		assert.ErrorIs(t, context.Cause(lock.Context()), errLockLeaseExpired)
	})
}

func TestWithTimeoutRejectsNonPositiveTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		assert.Panics(t, func() {
			WithTimeout(timeout)
		})
	}
}

func TestLockRefreshOwnershipMismatchBreaksWithCause(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(int64(0), nil)
	}
	lock := newHeldTestLock(cmdable, lockDefaultTimeout)

	assert.False(t, lock.refresh())

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
	assert.ErrorIs(t, context.Cause(lock.Context()), errLockOwnershipLost)
	require.Len(t, cmdable.evalCalls, 1)
	assert.Equal(t, refreshScript, cmdable.evalCalls[0].script)
}

func TestLockRefreshUsesBoundedDeadlineAndExtendsLease(t *testing.T) {
	cmdable := newTestLockCmdable()
	var commandDeadline time.Time
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		commandDeadline, _ = ctx.Deadline()
		return goredis.NewCmdResult(int64(1), nil)
	}
	lock := newHeldTestLock(cmdable, lockDefaultTimeout)
	originalDeadline := lock.leaseDeadline
	startedAt := time.Now()

	require.NoError(t, lock.refreshWithRetry())

	assert.WithinDuration(t, startedAt.Add(lockRefreshCommandTimeout), commandDeadline, 100*time.Millisecond)
	assert.True(t, lock.leaseDeadline.After(originalDeadline))
}

func TestLocalLeaseDeadlineKeepsSafetyMargin(t *testing.T) {
	startedAt := time.Unix(1_700_000_000, 0)

	assert.Equal(t, startedAt.Add(29*time.Second), localLeaseDeadline(startedAt, 30*time.Second))
	assert.Equal(t, startedAt.Add(4500*time.Millisecond), localLeaseDeadline(startedAt, 5*time.Second))
}

func TestLockRefreshStopsWhenRetryWouldExceedLease(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(nil, errors.New("refresh failed"))
	}
	lock := newHeldTestLock(cmdable, 100*time.Millisecond)
	startedAt := time.Now()

	assert.False(t, lock.refresh())

	assert.True(t, lock.IsBroken())
	cause := context.Cause(lock.Context())
	require.Error(t, cause)
	assert.ErrorContains(t, cause, "refresh redis lock failed before lease deadline")
	assert.ErrorContains(t, cause, "refresh failed")
	assert.Less(t, time.Since(startedAt), time.Second)
	require.Len(t, cmdable.evalCalls, 1)
}

func TestLockRefreshCommandCannotRunPastLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmdable := newTestLockCmdable()
		cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
			<-ctx.Done()
			return goredis.NewCmdResult(nil, ctx.Err())
		}
		lock := newHeldTestLock(cmdable, 30*time.Millisecond)
		startedAt := time.Now()

		err := lock.refreshWithRetry()

		require.Error(t, err)
		assert.Equal(t, 27*time.Millisecond, time.Since(startedAt))
		require.Len(t, cmdable.evalCalls, 1)
	})
}

func TestLockUnlockPanicsAndBreaksOnReleaseFailure(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(nil, errors.New("release failed"))
	}
	lock := &Lock{
		ctx:     context.Background(),
		cmdable: cmdable,
		key:     "vine:lock:lock:user:123",
		option:  defaultLockOption(),
		token:   "held-token",
	}
	lock.lockCtx, lock.lockCancel = context.WithCancelCause(lock.ctx)

	assert.PanicsWithError(t, "release redis lock failed: release failed", func() {
		lock.Unlock()
	})

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
	assert.EqualError(t, context.Cause(lock.Context()), "release redis lock failed: release failed")
	require.Len(t, cmdable.evalCalls, 1)
	assert.Equal(t, unlockScript, cmdable.evalCalls[0].script)
	assert.Equal(t, []string{"vine:lock:lock:user:123"}, cmdable.evalCalls[0].keys)
	assert.Equal(t, []any{"held-token"}, cmdable.evalCalls[0].args)
}

func TestLockUnlockPanicsAndBreaksWhenOwnershipIsLost(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(int64(0), nil)
	}
	lock := newHeldTestLock(cmdable, time.Second)

	assert.PanicsWithError(t, errLockOwnershipLost.Error(), func() {
		lock.Unlock()
	})

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, context.Cause(lock.Context()), errLockOwnershipLost)
}

func TestLockUnlockCancelsContextAfterRelease(t *testing.T) {
	lock := newHeldTestLock(newTestLockCmdable(), time.Second)

	lock.Unlock()

	assert.False(t, lock.IsBroken())
	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
	assert.ErrorIs(t, context.Cause(lock.Context()), context.Canceled)
}

func TestLockTryUnlockReleasesHeldLock(t *testing.T) {
	lock := newHeldTestLock(newTestLockCmdable(), time.Second)

	assert.True(t, lock.TryUnlock())

	assert.False(t, lock.IsBroken())
	assert.ErrorIs(t, context.Cause(lock.Context()), context.Canceled)
	assert.False(t, lock.TryUnlock())
}

func TestLockTryUnlockReturnsFalseWhenUnavailable(t *testing.T) {
	assert.False(t, new(Lock).TryUnlock())

	lock := newHeldTestLock(newTestLockCmdable(), time.Second)
	lock.markBroken(errLockLeaseExpired)

	assert.False(t, lock.TryUnlock())
}

func TestLockTryUnlockReturnsFalseAndBreaksWhenOwnershipIsLost(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(int64(0), nil)
	}
	lock := newHeldTestLock(cmdable, time.Second)

	assert.False(t, lock.TryUnlock())

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, context.Cause(lock.Context()), errLockOwnershipLost)
}

func TestLockTryUnlockPanicsAndBreaksOnReleaseFailure(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		return goredis.NewCmdResult(nil, errors.New("release failed"))
	}
	lock := newHeldTestLock(cmdable, time.Second)

	assert.PanicsWithError(t, "release redis lock failed: release failed", func() {
		lock.TryUnlock()
	})

	assert.True(t, lock.IsBroken())
	assert.EqualError(t, context.Cause(lock.Context()), "release redis lock failed: release failed")
}

func TestLockTryUnlockIsAtomicWithMarkBroken(t *testing.T) {
	cmdable := newTestLockCmdable()
	evalStarted := make(chan struct{})
	evalContinue := make(chan struct{})
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...any) *goredis.Cmd {
		close(evalStarted)
		<-evalContinue
		return goredis.NewCmdResult(int64(1), nil)
	}
	lock := newHeldTestLock(cmdable, time.Second)
	tryUnlockResult := make(chan bool, 1)
	go func() {
		tryUnlockResult <- lock.TryUnlock()
	}()
	<-evalStarted

	markBrokenDone := make(chan struct{})
	go func() {
		lock.markBroken(errLockLeaseExpired)
		close(markBrokenDone)
	}()
	close(evalContinue)

	assert.True(t, <-tryUnlockResult)
	<-markBrokenDone
	assert.False(t, lock.IsBroken())
	assert.ErrorIs(t, context.Cause(lock.Context()), context.Canceled)
}

func TestLockMarkBrokenPreventsUnlock(t *testing.T) {
	lockCtx, lockCancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() {
		lockCancel(nil)
	})
	lock := &Lock{
		token:      "held-token",
		option:     &_LockOption{timeout: time.Second},
		lockCtx:    lockCtx,
		lockCancel: lockCancel,
	}

	lock.markBroken(errLockLeaseExpired)

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
	assert.ErrorIs(t, context.Cause(lock.Context()), errLockLeaseExpired)
	assert.Panics(t, func() {
		lock.Unlock()
	})
	assert.Panics(t, func() {
		lock.Unlock()
	})
}

func TestLockContextRequiresHeldLock(t *testing.T) {
	lock := new(Lock)

	assert.False(t, lock.IsBroken())
	assert.Panics(t, func() {
		lock.Context()
	})
}

func newHeldTestLock(cmdable goredis.Cmdable, timeout time.Duration) *Lock {
	ctx := context.Background()
	lock := &Lock{
		ctx:           ctx,
		cmdable:       cmdable,
		key:           "vine:lock:lock:user:123",
		option:        &_LockOption{timeout: timeout, refresh: true},
		token:         "held-token",
		leaseDeadline: localLeaseDeadline(time.Now(), timeout),
	}
	lock.lockCtx, lock.lockCancel = context.WithCancelCause(ctx)
	return lock
}
