package nats

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
)

type dispatchTestMessages struct {
	next    func(context.Context) (jetstream.Msg, error)
	stopped bool
}

func (m *dispatchTestMessages) Next(ctx context.Context) (jetstream.Msg, error) { return m.next(ctx) }
func (m *dispatchTestMessages) Stop()                                           { m.stopped = true }

func TestDispatcherReleasesSlotBeforeRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		firstCtx, cancelFirst := context.WithCancel(t.Context())
		defer cancelFirst()
		reserves, releases := 0, 0
		messages := new(dispatchTestMessages)
		messages.next = func(ctx context.Context) (jetstream.Msg, error) {
			switch reserves {
			case 1:
				cancelFirst()
				return nil, ctx.Err()
			case 2:
				return nil, errors.New("temporary read failure")
			case 3:
				return nil, jetstream.ErrMsgIteratorClosed
			default:
				t.Fatalf("unexpected pull %d", reserves)
				return nil, nil
			}
		}
		dispatch := MessageDispatcher[int]{
			Reserve: func() (*int, context.Context) {
				require.Equal(t, reserves, releases, "previous slot must be free before the next pull")
				reserves++
				if reserves == 1 {
					return new(reserves), firstCtx
				}
				return new(reserves), t.Context()
			},
			Release: func(target *int) {
				releases++
				require.Equal(t, releases, *target)
			},
		}
		start := time.Now()
		dispatch.Run(t.Context(), messages)
		require.Equal(t, 3, releases)
		require.True(t, messages.stopped)
		require.Equal(t, 100*time.Millisecond, time.Since(start), "only the transient error should back off")
	})
}
