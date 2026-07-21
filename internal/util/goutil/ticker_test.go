package goutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan int, 3)

	go Tick(ctx, 10*time.Millisecond, func() {
		ticks <- 1
		if len(ticks) >= 2 {
			cancel()
		}
	})

	timeout := time.After(time.Second)
	count := 0
	for count < 2 {
		select {
		case <-ticks:
			count++
		case <-timeout:
			t.Fatal("Tick did not run twice before timeout")
		}
	}

	assert.GreaterOrEqual(t, count, 2)
}

func TestGoTickSafely(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan int, 4)
	panicOnce := true

	GoTickSafely(ctx, 10*time.Millisecond, func() {
		ticks <- 1
		if panicOnce {
			panicOnce = false
			panic("boom")
		}
		if len(ticks) >= 3 {
			cancel()
		}
	})

	timeout := time.After(time.Second)
	count := 0
	for count < 3 {
		select {
		case <-ticks:
			count++
		case <-timeout:
			t.Fatal("GoTickSafely did not continue after panic before timeout")
		}
	}

	assert.GreaterOrEqual(t, count, 3)
}

func TestSafeTickerUsesCustomRecoverHookAndContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan int, 4)
	recovered := make(chan any, 1)
	panicOnce := true

	safeTicker := NewSafeTicker(ctx, 10*time.Millisecond, func(r any) {
		recovered <- r
	})
	safeTicker.Go(func() {
		ticks <- 1
		if panicOnce {
			panicOnce = false
			panic("boom")
		}
		if len(ticks) >= 3 {
			cancel()
		}
	})

	select {
	case got := <-recovered:
		assert.Equal(t, "boom", got)
	case <-time.After(time.Second):
		t.Fatal("SafeTicker did not call custom recover hook before timeout")
	}

	timeout := time.After(time.Second)
	count := 0
	for count < 3 {
		select {
		case <-ticks:
			count++
		case <-timeout:
			t.Fatal("SafeTicker did not continue after panic before timeout")
		}
	}

	assert.GreaterOrEqual(t, count, 3)
}
