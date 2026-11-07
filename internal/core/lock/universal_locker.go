package lock

import (
	"context"

	"go.yorun.ai/vine/internal/core/link/skeled"
)

// UniversalLocker acquires locks shared across applications using the same lock backend.
type UniversalLocker struct {
	locker *Locker
}

// NewUniversalLocker creates a cross-application locker bound to Link and the active context.
func NewUniversalLocker(client skeled.LockServiceClient, ctx context.Context) *UniversalLocker {
	return new(UniversalLocker{locker: new(Locker{client: client, prefix: "universal", ctx: ctx})})
}

// WithContext returns a copy bound to ctx for acquisition and lease lifetime.
func (l *UniversalLocker) WithContext(ctx context.Context) *UniversalLocker {
	return new(UniversalLocker{locker: l.locker.WithContext(ctx)})
}

// Lock waits for ownership using the same lease behavior as Locker.Lock.
func (l *UniversalLocker) Lock(key string, options ...OptionFunc) *Lock {
	return l.locker.Lock(key, options...)
}

// TryLock attempts acquisition once using the same lease behavior as Locker.TryLock.
func (l *UniversalLocker) TryLock(key string, options ...OptionFunc) (*Lock, bool) {
	return l.locker.TryLock(key, options...)
}
