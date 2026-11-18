package nats

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// MessageDispatcher pulls from one subscription using slots supplied by its caller.
type MessageDispatcher[T any] struct {
	// Reserve waits for capacity. A nil target ends the loop. The read context
	// belongs to the target, so a draining target can cancel its outstanding pull.
	Reserve func() (*T, context.Context)
	// Admit rechecks the target after the pull. On rejection it settles the
	// message according to the subscription's policy.
	Admit   func(*T, jetstream.Msg) bool
	Handle  func(*T, jetstream.Msg)
	Release func(*T)
}

// Run stops the iterator when dispatch ends. Each admitted message holds its
// slot until Handle returns, including its final acknowledgement.
func (d MessageDispatcher[T]) Run(ctx context.Context, messages MessagesContext) {
	defer messages.Stop()
	for ctx.Err() == nil {
		target, readCtx := d.Reserve()
		if target == nil {
			return
		}
		msg, err := messages.Next(readCtx)
		if err != nil {
			d.Release(target)
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-readCtx.Done():
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}
		if !d.Admit(target, msg) {
			d.Release(target)
			continue
		}
		go func() {
			defer d.Release(target)
			d.Handle(target, msg)
		}()
	}
}
