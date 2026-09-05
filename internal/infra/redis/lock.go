package redis

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"uuid"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vpre"
)

type _LockOption struct {
	timeout time.Duration
	refresh bool
}

type LockOptionFunc func(*_LockOption)

func WithTimeout(timeout time.Duration) LockOptionFunc {
	vpre.Check(timeout > 0, "redis lock timeout must be positive")
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
	lockerKeyPrefixSentinel   = "\x00"
	lockDefaultTimeout        = 30 * time.Second
	lockRefreshInterval       = 10 * time.Second
	lockRefreshRetryInterval  = 3 * time.Second
	lockRefreshMaxRetry       = 7
	lockRefreshCommandTimeout = 2 * time.Second
	lockLeaseSafetyMarginMax  = time.Second
	unlockScript              = `
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

var (
	errLockLeaseExpired  = errors.New("redis lock lease expired")
	errLockOwnershipLost = errors.New("redis lock ownership lost")
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

func (m *RedisManager) instantiateLocker(lockerType reflect.Type, ctx context.Context) any {
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

	option        *_LockOption
	token         string
	lockCtx       context.Context
	lockCancel    context.CancelCauseFunc
	leaseDeadline time.Time
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
	token := uuid.NewV7().String()
	startedAt := time.Now()
	ok, err := l.cmdable.SetNX(l.ctx, l.key, token, l.option.timeout).Result()
	vpre.CheckNilError(err, "acquire redis lock failed")
	if !ok {
		return false
	}

	l.token = token
	l.lockCtx, l.lockCancel = context.WithCancelCause(l.ctx)
	l.leaseDeadline = localLeaseDeadline(startedAt, l.option.timeout)
	return true
}

func (l *Lock) waitTimeout() {
	wait := time.Until(l.currentLeaseDeadline())
	if wait <= 0 {
		l.markBroken(errLockLeaseExpired)
		return
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-l.lockCtx.Done():
		return
	case <-timer.C:
		l.markBroken(errLockLeaseExpired)
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
			if !l.refresh() {
				return
			}
		}
	}
}

func (l *Lock) refresh() bool {
	err := l.refreshWithRetry()
	if err == nil {
		return true
	}
	if l.lockCtx.Err() != nil {
		return false
	}
	l.markBroken(err)
	return false
}

func (l *Lock) refreshWithRetry() error {
	var lastErr error
	for attempt := range lockRefreshMaxRetry {
		startedAt := time.Now()
		deadline := l.currentLeaseDeadline()
		if !startedAt.Before(deadline) {
			if lastErr != nil {
				return fmt.Errorf("refresh redis lock failed before lease deadline: %w", lastErr)
			}
			return errLockLeaseExpired
		}

		commandDeadline := deadline
		if timeoutDeadline := startedAt.Add(lockRefreshCommandTimeout); timeoutDeadline.Before(commandDeadline) {
			commandDeadline = timeoutDeadline
		}
		commandCtx, cancel := context.WithDeadline(l.lockCtx, commandDeadline)
		result, err := l.cmdable.Eval(commandCtx, refreshScript, []string{l.key}, l.token, l.option.timeout.Milliseconds()).Int64()
		cancel()
		if err == nil {
			if result != 1 {
				return errLockOwnershipLost
			}
			if !l.extendLease(startedAt) {
				return context.Cause(l.lockCtx)
			}
			return nil
		}
		lastErr = err

		if attempt == lockRefreshMaxRetry-1 {
			return fmt.Errorf("refresh redis lock failed after %d attempts: %w", lockRefreshMaxRetry, err)
		}

		if !time.Now().Add(lockRefreshRetryInterval).Before(deadline) {
			return fmt.Errorf("refresh redis lock failed before lease deadline: %w", err)
		}
		retryTimer := time.NewTimer(lockRefreshRetryInterval)
		select {
		case <-l.lockCtx.Done():
			retryTimer.Stop()
			return context.Cause(l.lockCtx)
		case <-retryTimer.C:
		}
	}

	return fmt.Errorf("refresh redis lock failed: %w", lastErr)
}

func (l *Lock) currentLeaseDeadline() time.Time {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.leaseDeadline
}

func (l *Lock) extendLease(startedAt time.Time) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.lockCtx.Err() != nil {
		return false
	}
	l.leaseDeadline = localLeaseDeadline(startedAt, l.option.timeout)
	return true
}

func localLeaseDeadline(startedAt time.Time, timeout time.Duration) time.Time {
	safetyMargin := min(timeout/10, lockLeaseSafetyMarginMax)
	return startedAt.Add(timeout - safetyMargin)
}

func (l *Lock) markBroken(cause error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.markBrokenLocked(cause)
}

func (l *Lock) markBrokenLocked(cause error) {
	if l.lockCtx.Err() != nil {
		return
	}
	l.broken = true
	l.lockCancel(cause)
}

func (l *Lock) Unlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.broken {
		vpre.Panic(fmt.Errorf("redis lock is broken: %w", context.Cause(l.lockCtx)))
	}
	vpre.CheckNil(l.lockCtx.Err(), "lock is released")
	if !l.unlockLocked() {
		vpre.Panic(context.Cause(l.lockCtx))
	}
}

// TryUnlock atomically checks the local lock state and attempts a token-checked
// Redis release. It returns false when the lock was not acquired, was already
// released or broken, or is no longer owned by this token. Redis command errors
// panic according to the infrastructure fail-fast policy.
func (l *Lock) TryUnlock() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.lockCtx == nil || l.broken || l.lockCtx.Err() != nil {
		return false
	}
	return l.unlockLocked()
}

func (l *Lock) unlockLocked() bool {
	result, err := l.cmdable.Eval(l.ctx, unlockScript, []string{l.key}, l.token).Int64()
	if err != nil {
		cause := fmt.Errorf("release redis lock failed: %w", err)
		l.markBrokenLocked(cause)
		vpre.Panic(cause)
	}
	if result != 1 {
		l.markBrokenLocked(errLockOwnershipLost)
		return false
	}
	l.lockCancel(nil)
	return true
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
