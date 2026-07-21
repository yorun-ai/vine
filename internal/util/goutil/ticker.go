package goutil

import (
	"context"
	"time"
)

type SafeTicker struct {
	ctx       context.Context
	interval  time.Duration
	recoverFn func(any)
}

func Tick(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

func NewSafeTicker(ctx context.Context, interval time.Duration, recoverFn func(any)) *SafeTicker {
	if recoverFn == nil {
		recoverFn = rescueRecovered
	}
	return &SafeTicker{
		ctx:       ctx,
		interval:  interval,
		recoverFn: recoverFn,
	}
}

func (t *SafeTicker) Go(fn any, args ...any) {
	GoSafely(func() {
		Tick(t.ctx, t.interval, func() {
			RunWithRecover(t.recoverFn, fn, args...)
		})
	})
}

func GoTickSafely(ctx context.Context, interval time.Duration, fn any, args ...any) {
	NewSafeTicker(ctx, interval, nil).Go(fn, args...)
}
