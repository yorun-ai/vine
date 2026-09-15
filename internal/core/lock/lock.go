package lock

import (
	"context"
	"sync/atomic"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/rpc/client"
)

// Lock represents an automatically renewed lease. Work must stop when Context ends.
type Lock struct {
	client   skeled.LockServiceClient
	key      string
	token    string
	ttl      time.Duration
	ctx      context.Context
	cancel   context.CancelCauseFunc
	unlock   chan chan _Result
	released atomic.Bool
}

type _Result struct {
	ok      bool
	failure ex.Error
}

func newLock(client skeled.LockServiceClient, parent context.Context, key, token string, ttl time.Duration, deadline time.Time) *Lock {
	ctx, cancel := context.WithCancelCause(parent)
	l := new(Lock{client: client, key: key, token: token, ttl: ttl, ctx: ctx, cancel: cancel, unlock: make(chan chan _Result)})
	go l.run(deadline)
	return l
}

// Context is cancelled on release, cancellation, expiry or renewal failure.
func (l *Lock) Context() context.Context {
	return l.ctx
}

// IsBroken reports loss of the lease; a successful explicit release is not broken.
func (l *Lock) IsBroken() bool {
	return l.ctx.Err() != nil && !l.released.Load()
}

// run is the sole owner of the lease deadline and serializes all mutations.
func (l *Lock) run(deadline time.Time) {
	timer := time.NewTimer(min(l.ttl/3, time.Until(deadline)))
	defer timer.Stop()
	for {
		var reply chan _Result
		select {
		case <-l.ctx.Done():
			return
		case reply = <-l.unlock:
		case <-timer.C:
		}

		started := time.Now()
		result := l.invoke(deadline, reply != nil)
		if result.failure != nil {
			l.cancel(result.failure)
		} else if !result.ok {
			l.cancel(ex.New(ex.OperationFailed, "lock lease expired or ownership lost"))
		} else if reply != nil {
			l.released.Store(true)
			l.cancel(context.Canceled)
		}
		if reply != nil {
			reply <- result
			return
		}
		if !result.ok || result.failure != nil {
			return
		}
		// Count the new lease from request start, not response arrival.
		deadline = started.Add(l.ttl)
		timer.Reset(min(l.ttl/3, time.Until(deadline)))
	}
}

func (l *Lock) invoke(deadline time.Time, release bool) _Result {
	if l.ctx.Err() != nil || !time.Now().Before(deadline) {
		return _Result{}
	}
	ctx, cancel := context.WithDeadline(l.ctx, deadline)
	defer cancel()
	results := make(chan _Result, 1)
	// Keep expiry responsive even if a transport returns late. Only run mutates
	// lease state; the buffered channel lets late calls finish without blocking.
	go func() {
		result := _Result{}
		defer func() {
			result.failure = ex.RecoverExecution(recover())
			results <- result
		}()
		if release {
			result.ok = l.client.Release(l.key, l.token, client.WithContext(ctx))
		} else {
			result.ok = l.client.Renew(l.key, l.token, int(l.ttl/time.Millisecond), client.WithContext(ctx))
		}
	}()
	select {
	case <-ctx.Done():
		return _Result{}
	case result := <-results:
		if ctx.Err() != nil || !time.Now().Before(deadline) {
			result.ok = false
		}
		return result
	}
}

// TryUnlock releases a live lease once. Lost ownership or an earlier release
// returns false. Backend failures panic and mark the lease broken.
func (l *Lock) TryUnlock() bool {
	reply := make(chan _Result, 1)
	select {
	case <-l.ctx.Done():
		return false
	case l.unlock <- reply:
	}
	result := <-reply
	if result.failure != nil {
		ex.PanicIfError(result.failure)
	}
	return result.ok
}

// Unlock releases the lease and panics if ownership has been lost or it was already released.
func (l *Lock) Unlock() {
	ex.PanicNewIfNot(l.TryUnlock(), ex.OperationFailed, "lock ownership lost or already released")
}
