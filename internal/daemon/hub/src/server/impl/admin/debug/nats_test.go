package debug

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/app"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	hubserver "go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
)

func newDebugBroker(t *testing.T, mode string) (*hubserver.NATSServer, *hubflag.Flag, jetstream.JetStream) {
	t.Helper()
	previous := hubnats.InprocServer()
	t.Cleanup(func() { hubnats.SetInprocServer(previous) })
	hubnats.SetInprocServer(nil)
	flag := &hubflag.Flag{MQMode: hubflag.MQModeEmbedded}
	var component *hubserver.NATSServer
	var conn *gonats.Conn
	if mode == "external" {
		server, err := natsserver.NewServer(&natsserver.Options{Port: -1, NoSigs: true, NoLog: true, JetStream: true, StoreDir: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() { server.Shutdown(); server.WaitForShutdown() })
		go server.Start()
		require.True(t, server.ReadyForConnections(2*time.Second))
		flag.MQNatsEndpoint = server.ClientURL()
		flag.MQMode = hubflag.MQModeNATS
		conn, err = gonats.Connect(flag.MQNatsEndpoint)
		require.NoError(t, err)
	} else {
		component = &hubserver.NATSServer{InprocFlag: &app.InternalInprocFlag{Enabled: mode == "inproc"}, Flag: flag, Identity: new(mtls.Identity)}
		t.Cleanup(component.AfterAppStop)
		component.DIInit()
		if mode == "inproc" {
			conn = hubnats.ConnectInproc()
		} else {
			var err error
			conn, err = component.ConnectAsHub()
			require.NoError(t, err)
		}
	}
	t.Cleanup(conn.Close)
	js, err := jetstream.New(conn)
	require.NoError(t, err)
	if mode == "external" {
		for _, config := range []jetstream.StreamConfig{debugEventStreamConfig(), debugTaskStreamConfig()} {
			config.Storage = jetstream.MemoryStorage
			config.MaxMsgs = 123
			_, err := js.CreateStream(t.Context(), config)
			require.NoError(t, err)
		}
	}
	return component, flag, js
}

func TestDebugMessagesPublishThroughConfiguredBroker(t *testing.T) {
	for _, mode := range []string{"inproc", "embedded", "external"} {
		t.Run(mode, func(t *testing.T) {
			component, flag, js := newDebugBroker(t, mode)
			eventService, taskService := newMessageDebugServices()
			eventService.NATSServer, eventService.Flag = component, flag
			taskService.NATSServer, taskService.Flag = component, flag
			eventConsumer, err := js.CreateConsumer(t.Context(), eventspec.NATSStreamName, jetstream.ConsumerConfig{Durable: "debug_event", FilterSubject: eventspec.NATSSubject("demo.A"), AckPolicy: jetstream.AckExplicitPolicy})
			require.NoError(t, err)
			taskConsumer, err := js.CreateConsumer(t.Context(), taskspec.NATSStreamName, jetstream.ConsumerConfig{Durable: "debug_task", FilterSubject: taskspec.NATSSubject("demo.A"), AckPolicy: jetstream.AckExplicitPolicy})
			require.NoError(t, err)
			trace := meta.InitialTrace()
			traceId, spanId := trace.Id(), trace.Span()
			before := time.Now()
			eventService.EmitEvent(skeled.EventDebugEmitRequest{EventSkelName: "demo.A", DescriptorHash: "v1", EventJson: skeltype.JSON(`{"message":"event"}`), TraceId: &traceId, SpanId: &spanId})
			event, err := eventConsumer.Next(jetstream.FetchMaxWait(time.Second))
			require.NoError(t, err)
			assert.Equal(t, eventspec.NATSSubject("demo.A"), event.Subject())
			decodedEvent := vcode.MustUnmarshalJson[eventspec.NATSMessage](event.Data())
			assert.Equal(t, "demo.A", decodedEvent.EventSkelName)
			assert.JSONEq(t, `{"message":"event"}`, decodedEvent.EventJson)
			assert.Equal(t, traceId, decodedEvent.Metadata.TraceId)
			assert.Equal(t, spanId, decodedEvent.Metadata.TraceSpan)
			assert.Equal(t, "vine.hub", decodedEvent.Metadata.AppName)
			assert.Equal(t, "1.2.3", decodedEvent.Metadata.AppVersion)
			assert.Equal(t, eventService.CurrentApp.InstanceId(), decodedEvent.Metadata.AppInstanceId.String())
			assert.False(t, decodedEvent.Metadata.EmittedAt.Time.Before(before.Add(-time.Second)))
			require.NoError(t, event.Ack())
			// Empty descriptor hash selects any registered version and missing trace creates one.
			taskService.LaunchTask(skeled.TaskDebugLaunchRequest{TaskSkelName: "demo.A", TriggerSkelName: "A", ArgumentsJson: skeltype.JSON(`{"message":"task"}`)})
			task, err := taskConsumer.Next(jetstream.FetchMaxWait(time.Second))
			require.NoError(t, err)
			assert.Equal(t, taskspec.NATSSubject("demo.A"), task.Subject())
			decodedTask := vcode.MustUnmarshalJson[taskspec.NATSMessage](task.Data())
			assert.Equal(t, "demo.A", decodedTask.TaskSkelName)
			assert.Equal(t, "A", decodedTask.TriggerSkelName)
			assert.JSONEq(t, `{"message":"task"}`, decodedTask.ArgumentsJson)
			_, err = meta.NewTrace(decodedTask.Metadata.TraceId, decodedTask.Metadata.TraceSpan)
			assert.NoError(t, err)
			assert.Equal(t, "vine.hub", decodedTask.Metadata.AppName)
			assert.Equal(t, "1.2.3", decodedTask.Metadata.AppVersion)
			assert.Equal(t, taskService.CurrentApp.InstanceId(), decodedTask.Metadata.AppInstanceId.String())
			assert.False(t, decodedTask.Metadata.LaunchedAt.Time.Before(before.Add(-time.Second)))
			require.NoError(t, task.Ack())
			for _, name := range []string{eventspec.NATSStreamName, taskspec.NATSStreamName} {
				stream, err := js.Stream(t.Context(), name)
				require.NoError(t, err)
				info, err := stream.Info(t.Context())
				require.NoError(t, err)
				assert.Equal(t, jetstream.MemoryStorage, info.Config.Storage)
				if mode == "external" {
					assert.EqualValues(t, 123, info.Config.MaxMsgs)
				}
			}
		})
	}
}

func TestDebugPublisherRequiresExternallyProvisionedStream(t *testing.T) {
	component, flag, js := newDebugBroker(t, "external")
	require.NoError(t, js.DeleteStream(t.Context(), eventspec.NATSStreamName))
	publisher := &_DebugNATSPublisher{NATSServer: component, Flag: flag}
	require.Panics(t, func() { publisher.publish(debugEventStreamConfig(), debugEventSubject("demo.A"), []byte(`{}`)) })
	_, err := js.Stream(context.Background(), eventspec.NATSStreamName)
	assert.ErrorIs(t, err, jetstream.ErrStreamNotFound)
}
