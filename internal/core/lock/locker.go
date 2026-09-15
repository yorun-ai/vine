package lock

import (
	"context"
	"math/rand/v2"
	"time"
	"uuid"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/rpc/client"
)

// Locker acquires locks shared by instances of the same application.
type Locker struct {
	client skeled.LockServiceClient
	prefix string
	ctx    context.Context
}

type _Options struct {
	ttl time.Duration
}

// OptionFunc configures a lock acquisition.
type OptionFunc func(*_Options)

// WithTTL sets the lease duration. Both Lock and TryLock renew acquired leases automatically.
func WithTTL(ttl time.Duration) OptionFunc {
	ex.PanicNewIfNot(ttl >= time.Millisecond, ex.InvalidRequest, "lock TTL must be at least one millisecond")
	return func(o *_Options) {
		o.ttl = ttl.Truncate(time.Millisecond)
	}
}

// NewLocker creates an application locker bound to Link and the active context.
func NewLocker(client skeled.LockServiceClient, ctx context.Context, appName string) *Locker {
	return new(Locker{client: client, prefix: "app:" + appName, ctx: ctx})
}

// WithContext returns a copy bound to ctx for acquisition and lease lifetime.
func (l *Locker) WithContext(ctx context.Context) *Locker {
	return new(Locker{client: l.client, prefix: l.prefix, ctx: ctx})
}

// Lock waits for ownership. Cancellation, timeout and backend failures panic with
// framework errors. Acquisition is neither fair nor reentrant.
func (l *Locker) Lock(key string, options ...OptionFunc) *Lock {
	ttl := lockTTL(options)
	delay := 10 * time.Millisecond
	for {
		if lease, ok := l.tryLock(key, ttl); ok {
			return lease
		}
		// Equal jitter prevents contenders from retrying in step.
		wait := delay/2 + time.Duration(rand.Int64N(int64(delay/2)))
		timer := time.NewTimer(wait)
		select {
		case <-l.ctx.Done():
			timer.Stop()
			checkContext(l.ctx)
		case <-timer.C:
		}
		delay = min(delay*2, 250*time.Millisecond)
	}
}

// TryLock attempts acquisition once. Only contention returns false; backend
// failures, cancellation and timeout panic with framework errors.
func (l *Locker) TryLock(key string, options ...OptionFunc) (*Lock, bool) {
	return l.tryLock(key, lockTTL(options))
}

func lockTTL(options []OptionFunc) time.Duration {
	o := _Options{ttl: 30 * time.Second}
	for _, apply := range options {
		apply(&o)
	}
	return o.ttl
}

func (l *Locker) tryLock(key string, ttl time.Duration) (*Lock, bool) {
	checkContext(l.ctx)
	token := uuid.NewV7().String()
	key = l.prefix + ":" + key
	started := time.Now()
	ctx, cancel := context.WithTimeout(l.ctx, ttl)
	defer cancel()
	acquired := l.client.Acquire(key, token, int(ttl/time.Millisecond), client.WithContext(ctx))
	checkContext(ctx)
	if !acquired {
		return nil, false
	}
	deadline := started.Add(ttl)
	ex.PanicNewIfNot(time.Now().Before(deadline), ex.InvocationTimeout, "lock acquisition exceeded its lease duration")
	return newLock(l.client, l.ctx, key, token, ttl, deadline), true
}

func checkContext(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		code := ex.InvocationCancelled
		if err == context.DeadlineExceeded {
			code = ex.InvocationTimeout
		}
		ex.PanicNew(code, "lock context ended", ex.WithCause(err))
	}
}
