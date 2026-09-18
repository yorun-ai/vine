package nats

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
)

func TestMessagesPullsOnlyOnDemand(t *testing.T) {
	server := newTestNATSServer(t)
	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	subject := formatTestQueueSubject("bounded")
	messages := client.Messages(t.Context(), queueStreamConfigForTest(), subject, "task.bounded")
	t.Cleanup(messages.Stop)
	for range 20 {
		client.Publish(queueStreamConfigForTest(), subject, []byte("work"))
	}
	consumer, err := client.jetStream.Consumer(t.Context(), testQueueStreamName, "task_bounded")
	require.NoError(t, err)
	for count := uint64(1); count <= 3; count++ {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		msg, err := messages.Next(ctx)
		cancel()
		require.NoError(t, err)
		require.NoError(t, msg.Ack())
		info, err := consumer.Info(t.Context())
		require.NoError(t, err)
		require.Equal(t, count, info.Delivered.Consumer, "iterator must not prefetch the backlog")
	}
}

func TestMessagesSurvivesRecoveryAndConnectionReplacement(t *testing.T) {
	first, second := newTestNATSServer(t), newTestNATSServer(t)
	client := new(_Client)
	old := connectTestNATS(t, "nats://"+first.Addr().String())
	client.setConn(old)
	subject := formatTestQueueSubject("recovery")
	messages := client.Messages(t.Context(), queueStreamConfigForTest(), subject, "recovery")
	t.Cleanup(messages.Stop)
	client.Publish(queueStreamConfigForTest(), subject, []byte("before"))
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	msg, err := messages.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, "before", string(msg.Data()))
	require.NoError(t, msg.Ack())

	// A Next already waiting on the old iterator must follow its replacement.
	result := make(chan jetstream.Msg, 1)
	errors := make(chan error, 1)
	go func() { msg, err := messages.Next(ctx); result <- msg; errors <- err }()
	client.recoverJetStream(ctx)
	client.setConn(connectTestNATS(t, "nats://"+second.Addr().String()))
	old.Close()
	client.recoverJetStream(ctx)
	client.Publish(queueStreamConfigForTest(), subject, []byte("after"))
	select {
	case msg := <-result:
		require.NoError(t, <-errors)
		require.Equal(t, "after", string(msg.Data()))
		require.NoError(t, msg.Ack())
	case <-ctx.Done():
		t.Fatal("message iterator did not recover")
	}
}

func TestMessagesCancelStopAndRecovery(t *testing.T) {
	server := newTestNATSServer(t)
	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	subject := formatTestQueueSubject("stop")
	messages := client.Messages(t.Context(), queueStreamConfigForTest(), subject, "stop")
	t.Cleanup(messages.Stop)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := messages.Next(ctx)
	require.ErrorIs(t, err, context.Canceled)
	ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := messages.Next(ctx); done <- err }()
	messages.Stop()
	messages.Stop()
	select {
	case err := <-done:
		require.ErrorIs(t, err, jetstream.ErrMsgIteratorClosed)
	case <-ctx.Done():
		t.Fatal("Stop did not interrupt Next")
	}
	client.recoverJetStream(ctx)
	client.mutex.Lock()
	count := len(client.messages)
	client.mutex.Unlock()
	require.Zero(t, count)
	_, err = messages.Next(ctx)
	require.ErrorIs(t, err, jetstream.ErrMsgIteratorClosed)
}
