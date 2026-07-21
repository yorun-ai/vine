package redis

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/util/vpre"
)

type _LockOption struct {
	timeout time.Duration
	refresh bool
}

type LockOptionFunc func(*_LockOption)

func WithTimeout(timeout time.Duration) LockOptionFunc {
	return func(option *_LockOption) {
		option.timeout = timeout
		option.refresh = false
	}
}

func defaultLockOption() *_LockOption {
	return &_LockOption{
		timeout: lockDefaultTimeout,
		refresh: true,
	}
}

const (
	lockKeyPrefixGlobal = "vine:lock:"
	// lockerKeyPrefixSentinel marks lockers that still use the base
	// Locker.KeyPrefix implementation, so we can derive a type-based prefix.
	lockerKeyPrefixSentinel  = "\x00"
	lockDefaultTimeout       = 30 * time.Second
	lockRefreshInterval      = 10 * time.Second
	lockRefreshRetryInterval = 3 * time.Second
	lockRefreshMaxRetry      = 7
	unlockScript             = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`
	refreshScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0
`
)

type _LockerSpec interface {
	KeyPrefix() string
	configure(ctx context.Context, cmdable goredis.Cmdable, keyPrefix string)
}

type Locker struct {
	ctx       context.Context
	cmdable   goredis.Cmdable
	keyPrefix string
}

func (l *Locker) KeyPrefix() string {
	// By default, each locker type gets a unique key prefix derived from its full
	// type name. If multiple locker types need to operate on the same Redis lock
	// namespace, they must override KeyPrefix() to return the same value.
	return lockerKeyPrefixSentinel
}

func (r *Redis) NewLocker(ctx context.Context, keyPrefix string) *Locker {
	vpre.CheckNotNil(ctx, "redis lock context is nil")
	vpre.CheckNotEmpty(keyPrefix, "redis lock key prefix is empty")
	return &Locker{
		ctx:       ctx,
		cmdable:   r.Cmdable,
		keyPrefix: keyPrefix,
	}
}

func (r *Redis) NewLockerByType(lockerType reflect.Type, ctx context.Context) any {
	vpre.CheckNotNil(ctx, "redis lock context is nil")
	vpre.Check(lockerType.Kind() == reflect.Pointer, "redis locker type %s must be pointer", lockerType)

	lockerValue := reflect.New(lockerType.Elem())
	locker := lockerValue.Interface()
	lockerSpec, ok := locker.(_LockerSpec)
	vpre.Check(ok, "locker type %s must embed redis.Locker", lockerType)
	keyPrefix := lockerSpec.KeyPrefix()
	if keyPrefix == lockerKeyPrefixSentinel {
		keyPrefix = defaultLockerTypeKeyPrefix(lockerType)
	}
	vpre.CheckNotEmpty(keyPrefix, "redis lock key prefix is empty")
	lockerSpec.configure(ctx, r.Cmdable, keyPrefix)
	return locker
}

func (m *RedisMinder) instantiateLocker(lockerType reflect.Type, ctx context.Context) any {
	return m.component.(_RedisAccessor).embeddedRedis().NewLockerByType(lockerType, ctx)
}

func defaultLockerTypeKeyPrefix(lockerType reflect.Type) string {
	kind := lockerType.Elem()
	// Use the full type name as the fallback namespace so lockers without an
	// explicit KeyPrefix still get a stable, unique Redis key prefix.
	return strings.ReplaceAll(kind.PkgPath()+"."+kind.Name(), "/", "_")
}

func (l *Locker) configure(ctx context.Context, cmdable goredis.Cmdable, keyPrefix string) {
	l.ctx = ctx
	l.cmdable = cmdable
	l.keyPrefix = keyPrefix
}

type Lock struct {
	ctx     context.Context
	cmdable goredis.Cmdable
	key     string

	mutex  sync.Mutex
	broken bool

	option     *_LockOption
	token      string
	lockCtx    context.Context
	lockCancel context.CancelFunc
}

func (l *Locker) Lock(key string, options ...LockOptionFunc) (*Lock, bool) {
	option := defaultLockOption()
	for _, optionFunc := range options {
		optionFunc(option)
	}

	lock := &Lock{
		ctx:     l.ctx,
		cmdable: l.cmdable,
		key:     joinLockKey(l.keyPrefix, key),
		option:  option,
	}
	return lock, lock.lock()
}

// joinLockKey combines the global lock namespace with the locker prefix and
// resource key. For example:
//
//	joinLockKey("lock:user", "123") == "vine:lock:lock:user:123"
func joinLockKey(keyPrefix string, key string) string {
	return fmt.Sprintf("%s%s:%s", lockKeyPrefixGlobal, keyPrefix, key)
}

func (l *Lock) lock() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.doLock() {
		return false
	}

	if l.option.refresh {
		go l.refreshLoop()
		return true
	}

	go l.waitTimeout()
	return true
}

func (l *Lock) doLock() bool {
	token := uuid.Must(uuid.NewV7()).String()
	ok, err := l.cmdable.SetNX(l.ctx, l.key, token, l.option.timeout).Result()
	vpre.CheckNilError(err, "acquire redis lock failed")
	if !ok {
		return false
	}

	l.token = token
	l.lockCtx, l.lockCancel = context.WithCancel(l.ctx)
	return true
}

func (l *Lock) waitTimeout() {
	timer := time.NewTimer(l.option.timeout)
	defer timer.Stop()

	select {
	case <-l.lockCtx.Done():
		return
	case <-timer.C:
		l.markBroken()
	}
}

func (l *Lock) refreshLoop() {
	ticker := time.NewTicker(lockRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.lockCtx.Done():
			return
		case <-ticker.C:
			if !l.refreshWithRetry() {
				l.markBroken()
				return
			}
		}
	}
}

func (l *Lock) refreshWithRetry() bool {
	retryTicker := time.NewTicker(lockRefreshRetryInterval)
	defer retryTicker.Stop()

	for range lockRefreshMaxRetry {
		result, err := l.cmdable.Eval(l.lockCtx, refreshScript, []string{l.key}, l.token, l.option.timeout.Milliseconds()).Int64()
		if err == nil && result == 1 {
			return true
		}

		select {
		case <-l.lockCtx.Done():
			return false
		case <-retryTicker.C:
		}
	}

	return false
}

func (l *Lock) markBroken() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.broken = true
	l.lockCancel()
}

func (l *Lock) Unlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	vpre.Check(!l.broken, "redis lock is broken")
	vpre.CheckNil(l.lockCtx.Err(), "lock is released")

	err := l.cmdable.Eval(l.ctx, unlockScript, []string{l.key}, l.token).Err()
	if err != nil {
		logger.Error("release redis lock failed", "key", l.key, "error", err)
	}
	l.lockCancel()
}

func (l *Lock) Context() context.Context {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	vpre.CheckNotNil(l.lockCtx, "redis lock context is nil")
	return l.lockCtx
}

func (l *Lock) IsBroken() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.broken
}
