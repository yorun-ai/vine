package goutil

import (
	"sync"
)

type RecoverableOnce struct {
	mu   sync.Mutex
	done bool
}

func (o *RecoverableOnce) Do(fn func()) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.done {
		return
	}

	completed := false
	defer func() {
		if completed {
			o.done = true
		}
	}()

	fn()
	completed = true
}
