package nats

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Allow two retries within the same AckWait window after a failed send.
var renewalOffsets = [...]time.Duration{
	MessageAckWait - 10*time.Second,
	MessageAckWait - 5*time.Second,
	MessageAckWait - 2*time.Second,
}

// KeepMessageAlive renews an admitted delivery while its handler runs. Call
// the returned stop function before settling the message. It waits for any
// in-flight InProgress call and can also be deferred for panic cleanup.
func KeepMessageAlive(msg jetstream.Msg) func() {
	var mutex sync.Mutex
	var timer *time.Timer
	windowStart := time.Now()
	attempt := 0

	// A short handler can stop the timer before it starts a goroutine.
	mutex.Lock()
	timer = time.AfterFunc(renewalOffsets[0], func() {
		mutex.Lock()
		defer mutex.Unlock()
		now := time.Now()
		if timer == nil || now.Sub(windowStart) >= MessageAckWait {
			return
		}
		if err := msg.InProgress(); err != nil {
			natsLogger.Warn("message acknowledgement renewal failed", "subject", msg.Subject(), "error", err)
			attempt++
			if attempt == len(renewalOffsets) {
				return
			}
		} else {
			windowStart = now
			attempt = 0
		}
		timer.Reset(time.Until(windowStart.Add(renewalOffsets[attempt])))
	})
	mutex.Unlock()

	return func() {
		mutex.Lock()
		defer mutex.Unlock()
		if timer != nil {
			timer.Stop()
			timer = nil
		}
	}
}
