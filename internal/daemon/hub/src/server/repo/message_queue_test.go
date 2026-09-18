package repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	ns "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/core/mtls/mtlstest"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func newMessageQueueTestRepo(t *testing.T, mode string) *MessageQueueRepo {
	t.Helper()
	r := &MessageQueueRepo{InprocFlag: &app.InternalInprocFlag{Enabled: mode == "inproc"}, Flag: &flag.Flag{MQMode: flag.MQModeEmbedded}}
	if mode == "external" {
		server, err := ns.NewServer(&ns.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir(), Username: "operator", Password: "secret", NoSigs: true, NoLog: true})
		require.NoError(t, err)
		server.Start()
		t.Cleanup(func() { server.Shutdown(); server.WaitForShutdown() })
		require.True(t, server.ReadyForConnections(5*time.Second))
		r.Flag = &flag.Flag{MQMode: flag.MQModeNATS, MQNatsEndpoint: "nats://operator:secret@" + server.Addr().String()}
	} else {
		identity := mtls.DisabledIdentity()
		if mode == "tls" {
			identity = mtlstest.NewCA(t).Identity(t, daemon.HubIdentity.SPIFFEPath())
		}
		r.NATSServer = &natsserver.NATSServer{InprocFlag: r.InprocFlag, Flag: r.Flag, Identity: identity}
		r.NATSServer.DIInit()
		t.Cleanup(r.NATSServer.AfterAppStop)
	}
	return r
}

func queueTestJS(t *testing.T, r *MessageQueueRepo) jetstream.JetStream {
	t.Helper()
	conn, err := r.connect()
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	js, err := jetstream.New(conn)
	require.NoError(t, err)
	return js
}

func TestMessageQueueRepoSnapshots(t *testing.T) {
	for _, mode := range []string{"tcp", "inproc", "external", "tls"} {
		t.Run(mode, func(t *testing.T) {
			r := newMessageQueueTestRepo(t, mode)
			js := queueTestJS(t, r)
			ctx := t.Context()
			for _, cfg := range []jetstream.StreamConfig{
				{Name: taskspec.NATSStreamName, Subjects: []string{"task.>"}, Retention: jetstream.WorkQueuePolicy, Storage: jetstream.MemoryStorage},
				{Name: eventspec.NATSStreamName, Subjects: []string{"event.>"}, Retention: jetstream.InterestPolicy, Storage: jetstream.MemoryStorage},
			} {
				_, err := js.CreateOrUpdateStream(ctx, cfg)
				require.NoError(t, err)
			}
			task, err := js.CreateConsumer(ctx, taskspec.NATSStreamName, jetstream.ConsumerConfig{Durable: "task_demo", FilterSubject: "task.demo.Task", AckPolicy: jetstream.AckExplicitPolicy})
			require.NoError(t, err)
			for _, name := range []string{"event_app_b", "event_app_a"} {
				_, err = js.CreateConsumer(ctx, eventspec.NATSStreamName, jetstream.ConsumerConfig{Durable: name, FilterSubjects: []string{"event.demo.Other", "event.demo.Event"}, AckPolicy: jetstream.AckExplicitPolicy})
				require.NoError(t, err)
			}
			for _, subject := range []string{"task.demo.Task", "task.demo.Orphan", "event.demo.Event"} {
				_, err = js.Publish(ctx, subject, []byte("payload"))
				require.NoError(t, err)
			}
			items, err := r.List(ctx)
			require.NoError(t, err)
			require.Len(t, items, 2)
			require.Equal(t, uint64(2), items[0].Messages)
			require.Equal(t, []core.MessageQueueSubject{{Subject: "task.demo.Orphan", Messages: 1}, {Subject: "task.demo.Task", Messages: 1}}, items[0].Subjects)
			require.Len(t, items[0].Consumers, 1)
			require.Equal(t, uint64(1), items[0].Consumers[0].Pending)
			require.Zero(t, items[0].Consumers[0].AckPending)
			require.Equal(t, uint64(1), items[1].Messages)
			require.Len(t, items[1].Consumers, 2)
			require.Equal(t, "event_app_a", items[1].Consumers[0].Name)
			for _, consumer := range items[1].Consumers {
				require.Equal(t, uint64(1), consumer.Pending)
				require.Equal(t, []string{"event.demo.Event", "event.demo.Other"}, consumer.FilterSubjects)
			}
			again, err := r.List(ctx)
			require.NoError(t, err)
			require.Equal(t, items, again, "reading status must not consume or acknowledge")
			msg, err := task.Next(jetstream.FetchMaxWait(time.Second))
			require.NoError(t, err)
			items, err = r.List(ctx)
			require.NoError(t, err)
			require.Zero(t, items[0].Consumers[0].Pending)
			require.Equal(t, 1, items[0].Consumers[0].AckPending)
			require.NoError(t, msg.DoubleAck(ctx))
			items, err = r.List(ctx)
			require.NoError(t, err)
			require.Equal(t, uint64(1), items[0].Messages, "only the orphan task remains")
			require.Zero(t, items[0].Consumers[0].AckPending)
		})
	}
}

func TestMessageQueueRepoMissingStreamsAndCancellation(t *testing.T) {
	r := newMessageQueueTestRepo(t, "external")
	items, err := r.List(t.Context())
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, item := range items {
		require.False(t, item.Exists)
		require.Empty(t, item.Subjects)
		require.Empty(t, item.Consumers)
	}
	js := queueTestJS(t, r)
	_, err = js.Stream(t.Context(), taskspec.NATSStreamName)
	require.ErrorIs(t, err, jetstream.ErrStreamNotFound)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = r.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
	r.Flag.MQNatsEndpoint = "nats://operator:wrong@" + js.Conn().ConnectedAddr()
	_, err = r.List(t.Context())
	require.ErrorIs(t, err, gonats.ErrAuthorization)
}

func TestMessageQueueRepoListsEveryConsumerPage(t *testing.T) {
	r := newMessageQueueTestRepo(t, "external")
	js := queueTestJS(t, r)
	_, err := js.CreateStream(t.Context(), jetstream.StreamConfig{Name: eventspec.NATSStreamName, Subjects: []string{"event.>"}, Storage: jetstream.MemoryStorage})
	require.NoError(t, err)
	for i := range 260 {
		_, err = js.CreateConsumer(t.Context(), eventspec.NATSStreamName, jetstream.ConsumerConfig{Durable: fmt.Sprintf("reader_%03d", i), AckPolicy: jetstream.AckExplicitPolicy})
		require.NoError(t, err)
	}
	items, err := r.List(t.Context())
	require.NoError(t, err)
	require.Len(t, items[1].Consumers, 260)
	require.Equal(t, []string{">"}, items[1].Consumers[0].FilterSubjects)
}
