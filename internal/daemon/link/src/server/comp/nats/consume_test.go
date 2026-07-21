package nats

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestConsumeSanitizesConsumerName(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	consumeCtx := client.Consume(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), "consumer.alpha/demo:app", func(jetstream.Msg) {})
	defer consumeCtx.Stop()

	managedCtx := consumeCtx.(*_ConsumeContext)
	if managedCtx.consumerName != "consumer_alpha_demo_app" {
		t.Fatalf("unexpected managed consumer name: %s", managedCtx.consumerName)
	}

	stream, err := client.jetStream.Stream(context.Background(), testBroadcastStreamName)
	if err != nil {
		t.Fatalf("read stream failed: %v", err)
	}
	_, err = stream.Consumer(context.Background(), "consumer_alpha_demo_app")
	if err != nil {
		t.Fatalf("read sanitized consumer failed: %v", err)
	}
}

func TestConsumeStopIsIdempotent(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	consumeCtx := client.Consume(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), formatTestBroadcastConsumer("alpha.created", "demo-app"), func(jetstream.Msg) {})

	consumeCtx.Stop()
	consumeCtx.Stop()

	if len(client.consumers) != 0 {
		t.Fatalf("unexpected managed consumer count: %d", len(client.consumers))
	}
}

func TestConsumeRestartsManagedConsumersOnReconnect(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	firstCh := make(chan string, 1)
	consumeCtx := client.Consume(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), formatTestBroadcastConsumer("alpha.created", "demo-app"), func(msg jetstream.Msg) {
		firstCh <- string(msg.Data())
	})
	defer consumeCtx.Stop()

	if len(client.consumers) != 1 {
		t.Fatalf("unexpected managed consumer count: %d", len(client.consumers))
	}
	managedCtx := consumeCtx.(*_ConsumeContext)
	firstRawCtx := managedCtx.consumeContext

	client.onReconnect(context.Background(), client.conn)

	waitUntil(t, func() bool {
		managedCtx.mutex.Lock()
		restarted := managedCtx.consumeContext != firstRawCtx
		managedCtx.mutex.Unlock()

		client.mutex.Lock()
		streamCount := len(client.ensuredStream)
		client.mutex.Unlock()
		return restarted && streamCount == 1
	})

	client.Publish(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), []byte("ok"))

	assertPayload(t, firstCh, "ok")
}

func TestConsumeStoppedManagedConsumerIsRemoved(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	consumeCtx := client.Consume(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), formatTestBroadcastConsumer("alpha.created", "demo-app"), func(jetstream.Msg) {})
	managedCtx := consumeCtx.(*_ConsumeContext)

	consumeCtx.Stop()
	client.onReconnect(context.Background(), client.conn)

	waitUntil(t, func() bool {
		client.mutex.Lock()
		consumerCount := len(client.consumers)
		client.mutex.Unlock()

		managedCtx.mutex.Lock()
		consumeContext := managedCtx.consumeContext
		managedCtx.mutex.Unlock()
		return consumerCount == 0 && consumeContext == nil
	})
}

func TestSanitizeNATSResourceNamePreservesAllowedChars(t *testing.T) {
	got := sanitizeNATSResourceName("Alpha-123_beta")
	if got != "Alpha-123_beta" {
		t.Fatalf("unexpected sanitized value: %s", got)
	}
}

func TestSanitizeNATSResourceNameReplacesUnsupportedChars(t *testing.T) {
	got := sanitizeNATSResourceName("alpha.beta/gamma:delta")
	if got != "alpha_beta_gamma_delta" {
		t.Fatalf("unexpected sanitized value: %s", got)
	}
}

func TestSanitizeNATSResourceNameReturnsFallbackForEmptyResult(t *testing.T) {
	got := sanitizeNATSResourceName("")
	if got != "consumer" {
		t.Fatalf("unexpected sanitized value: %s", got)
	}
}

func TestSanitizeNATSResourceNameReplacesOnlyUnsupportedChars(t *testing.T) {
	got := sanitizeNATSResourceName("...")
	if got != "___" {
		t.Fatalf("unexpected sanitized value: %s", got)
	}
}

func waitUntil(t *testing.T, ready func() bool) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("condition timeout")
		case <-tick.C:
			if ready() {
				return
			}
		}
	}
}
