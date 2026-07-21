package goutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitableGoSafelyAndWait(t *testing.T) {
	var waitable Waitable
	var mu sync.Mutex
	values := []int{}

	waitable.GoSafely(func(value int) {
		mu.Lock()
		values = append(values, value)
		mu.Unlock()
	}, 3)
	waitable.GoSafely(func() {
		panic("boom")
	})
	waitable.Wait()

	assert.Equal(t, []int{3}, values)
}

func TestWaitableGoTickSafely(t *testing.T) {
	var waitable Waitable
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	count := 0

	waitable.GoTickSafely(ctx, 10*time.Millisecond, func(step int) {
		mu.Lock()
		count += step
		if count >= 2 {
			cancel()
		}
		mu.Unlock()
	}, 1)

	waitable.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, count, 2)
}
