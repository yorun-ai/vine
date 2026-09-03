package goutil

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		count := 0

		go Tick(ctx, 10*time.Millisecond, func() {
			count++
			if count == 2 {
				cancel()
			}
		})
		synctest.Sleep(20 * time.Millisecond)

		assert.Equal(t, 2, count)
	})
}

func TestGoTickSafely(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		count := 0
		panicOnce := true

		GoTickSafely(ctx, 10*time.Millisecond, func() {
			count++
			if panicOnce {
				panicOnce = false
				panic("boom")
			}
			if count == 3 {
				cancel()
			}
		})
		synctest.Sleep(30 * time.Millisecond)

		assert.Equal(t, 3, count)
	})
}

func TestSafeTickerUsesCustomRecoverHookAndContinues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		count := 0
		var recovered any
		panicOnce := true

		safeTicker := NewSafeTicker(ctx, 10*time.Millisecond, func(r any) {
			recovered = r
		})
		safeTicker.Go(func() {
			count++
			if panicOnce {
				panicOnce = false
				panic("boom")
			}
			if count == 3 {
				cancel()
			}
		})
		synctest.Sleep(30 * time.Millisecond)

		assert.Equal(t, "boom", recovered)
		assert.Equal(t, 3, count)
	})
}
