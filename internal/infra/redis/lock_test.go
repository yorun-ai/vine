package redis

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
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
	setNXFunc  func(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.BoolCmd
	evalFunc   func(ctx context.Context, script string, keys []string, args ...interface{}) *goredis.Cmd
}

type _TestSetNXCall struct {
	ctx        context.Context
	key        string
	value      interface{}
	expiration time.Duration
}

type _TestEvalCall struct {
	ctx    context.Context
	script string
	keys   []string
	args   []interface{}
}

func (c *_TestLockCmdable) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.BoolCmd {
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

func (c *_TestLockCmdable) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *goredis.Cmd {
	c.mutex.Lock()
	c.evalCalls = append(c.evalCalls, _TestEvalCall{
		ctx:    ctx,
		script: script,
		keys:   append([]string(nil), keys...),
		args:   append([]interface{}(nil), args...),
	})
	c.mutex.Unlock()
	return c.evalFunc(ctx, script, keys, args...)
}

func newTestLockCmdable() *_TestLockCmdable {
	return &_TestLockCmdable{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.BoolCmd {
			return goredis.NewBoolResult(true, nil)
		},
		evalFunc: func(ctx context.Context, script string, keys []string, args ...interface{}) *goredis.Cmd {
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

func TestRedisMinderBindProvidesLocker(t *testing.T) {
	original := newRedisClient
	t.Cleanup(func() {
		newRedisClient = original
	})

	newRedisClient = func(opt *Option) *goredis.Client {
		return goredis.NewClient(endpointOptions(opt.Endpoint))
	}

	component := new(_TestLockerRedis)
	minder := new(RedisMinder)
	minder.InitComponent(component)
	t.Cleanup(minder.AfterAppStop)

	injector := di.NewInjector(func(b *di.Binder) {
		b.Bind(reflect.TypeFor[context.Context]()).ToInstance(context.Background())
		minder.Bind(b)
		b.Bind(reflect.TypeFor[*_TestLockerConsumer]()).In(di.TransientScope)
	})

	consumer := injector.Get(reflect.TypeFor[*_TestLockerConsumer]()).Interface().(*_TestLockerConsumer)
	require.NotNil(t, consumer.Locker)
	assert.Equal(t, "lock:user", consumer.Locker.keyPrefix)
	require.NotNil(t, consumer.Locker.ctx)
	require.NotNil(t, consumer.Locker.cmdable)
}

func TestInstantiateLockerUsesDefaultTypePrefixWhenNotOverridden(t *testing.T) {
	minder := &RedisMinder{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	minder.component = &Redis{Cmdable: minder.client}
	t.Cleanup(func() {
		_ = minder.client.Close()
	})

	locker := minder.instantiateLocker(reflect.TypeFor[*_TestDefaultLocker](), context.Background()).(*_TestDefaultLocker)

	assert.Equal(t, "go.yorun.ai_vine_internal_infra_redis._TestDefaultLocker", locker.keyPrefix)
}

func TestInstantiateLockerRequiresNonEmptyOverriddenPrefix(t *testing.T) {
	minder := &RedisMinder{client: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})}
	minder.component = &Redis{Cmdable: minder.client}
	t.Cleanup(func() {
		_ = minder.client.Close()
	})

	assert.Panics(t, func() {
		minder.instantiateLocker(reflect.TypeFor[*_TestEmptyPrefixLocker](), context.Background())
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
	cmdable.setNXFunc = func(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.BoolCmd {
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
	cmdable := newTestLockCmdable()
	locker := &Locker{
		ctx:       context.Background(),
		cmdable:   cmdable,
		keyPrefix: "lock:user",
	}

	lock, ok := locker.Lock("123", WithTimeout(20*time.Millisecond))

	require.True(t, ok)
	require.NotNil(t, lock)
	assert.False(t, lock.option.refresh)
	assert.Eventually(t, func() bool {
		return lock.IsBroken()
	}, time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool {
		return errors.Is(lock.Context().Err(), context.Canceled)
	}, time.Second, 10*time.Millisecond)
}

func TestLockUnlockCancelsContextAndLogsReleaseFailure(t *testing.T) {
	cmdable := newTestLockCmdable()
	cmdable.evalFunc = func(ctx context.Context, script string, keys []string, args ...interface{}) *goredis.Cmd {
		return goredis.NewCmdResult(nil, errors.New("release failed"))
	}
	lock := &Lock{
		ctx:     context.Background(),
		cmdable: cmdable,
		key:     "vine:lock:lock:user:123",
		option:  defaultLockOption(),
		token:   "held-token",
	}
	lock.lockCtx, lock.lockCancel = context.WithCancel(lock.ctx)

	lock.Unlock()

	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
	require.Len(t, cmdable.evalCalls, 1)
	assert.Equal(t, unlockScript, cmdable.evalCalls[0].script)
	assert.Equal(t, []string{"vine:lock:lock:user:123"}, cmdable.evalCalls[0].keys)
	assert.Equal(t, []interface{}{"held-token"}, cmdable.evalCalls[0].args)
}

func TestLockMarkBrokenPreventsUnlock(t *testing.T) {
	lockCtx, lockCancel := context.WithCancel(context.Background())
	t.Cleanup(lockCancel)
	lock := &Lock{
		token:      "held-token",
		option:     &_LockOption{timeout: time.Second},
		lockCtx:    lockCtx,
		lockCancel: lockCancel,
	}

	lock.markBroken()

	assert.True(t, lock.IsBroken())
	assert.ErrorIs(t, lock.Context().Err(), context.Canceled)
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
