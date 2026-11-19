package nats

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
)

type renewalTestMessage struct {
	jetstream.Msg
	renew func() error
}

func (m *renewalTestMessage) InProgress() error { return m.renew() }
func (*renewalTestMessage) Subject() string     { return "test.renewal" }

func TestMessageRenewalRetriesWithinAckWindow(t *testing.T) {
	const second = time.Second
	for _, tc := range []struct {
		name     string
		failures int
		want     []time.Duration
	}{
		{"first retry succeeds", 1, []time.Duration{20 * second, 25 * second, 45 * second}},
		{"second retry succeeds", 2, []time.Duration{20 * second, 25 * second, 28 * second, 48 * second}},
		{"all attempts fail", 3, []time.Duration{20 * second, 25 * second, 28 * second}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				var attempts []time.Duration
				msg := &renewalTestMessage{renew: func() error {
					attempts = append(attempts, time.Since(start))
					if len(attempts) <= tc.failures {
						return errors.New("send failed")
					}
					return nil
				}}
				stop := KeepMessageAlive(msg)
				time.Sleep(50 * time.Second)
				stop()
				time.Sleep(MessageAckWait)
				require.Equal(t, tc.want, attempts)
			})
		})
	}
}
