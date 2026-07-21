package nats

import (
	"context"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _TestInfoServiceClient struct {
	info hubskeled.Info
}

const (
	testBroadcastStreamName  = "TEST_STREAM_BROADCAST"
	testBroadcastSubjectFmt  = "subject.%s"
	testBroadcastConsumerFmt = "consumer_%s_%s"
	testQueueStreamName      = "TEST_STREAM_QUEUE"
	testQueueSubjectFmt      = "job.%s"
	testQueueConsumerFmt     = "worker_%s"
)

func (c *_TestInfoServiceClient) GetInfo(_ ...rpcclient.InvokeOption) hubskeled.Info {
	return c.info
}

func TestClientInitOptionUsesHubInfoNATSPort(t *testing.T) {
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize(false)
	infoComponent := &hubinfo.HubInfo{
		Flag: flags,
		InfoServiceClient: &_TestInfoServiceClient{
			info: hubskeled.Info{
				NatsPort: 4222,
			},
		},
	}
	infoComponent.DIInit()

	client := &Client{
		Flag:    flags,
		HubInfo: infoComponent,
	}

	option := &_Option{}
	client.InitOption(option)

	assert.False(t, option.InprocMode)
	assert.Equal(t, "nats://127.0.0.1:4222", option.Endpoint)
}

func TestClientInitOptionUsesHubInfoMQEndpointWhenNATSPortIsEmpty(t *testing.T) {
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize(false)
	infoComponent := &hubinfo.HubInfo{
		Flag: flags,
		InfoServiceClient: &_TestInfoServiceClient{
			info: hubskeled.Info{
				MqEndpoint: "nats://10.0.0.8:4222",
			},
		},
	}
	infoComponent.DIInit()

	client := &Client{
		Flag:    flags,
		HubInfo: infoComponent,
	}

	option := &_Option{}
	client.InitOption(option)

	assert.False(t, option.InprocMode)
	assert.Equal(t, "nats://10.0.0.8:4222", option.Endpoint)
}

func TestClientInitOptionSkipsHubInfoLookupInInprocMode(t *testing.T) {
	flags := &flag.Flag{
		HubInprocMode: true,
		HubEndpoint:   "rpc+inproc://vine/hub",
	}
	flags.Normalize(false)
	client := &Client{
		Flag:    flags,
		HubInfo: &hubinfo.HubInfo{},
	}

	option := &_Option{}
	client.InitOption(option)

	assert.True(t, option.InprocMode)
	assert.Empty(t, option.Endpoint)
}

func TestClientPublishConsumeUsesBroadcastStream(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	ch := make(chan string, 1)
	consumeCtx := client.Consume(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), formatTestBroadcastConsumer("alpha.created", "demo-app"), func(msg jetstream.Msg) {
		ch <- string(msg.Data())
	})
	defer consumeCtx.Stop()

	client.Publish(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), []byte("ok"))

	assertPayload(t, ch, "ok")
}

func TestClientPublishConsumeUsesQueueStream(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	ch := make(chan string, 1)
	consumeCtx := client.Consume(queueStreamConfigForTest(), formatTestQueueSubject("sync-job"), formatTestQueueConsumer("sync-job"), func(msg jetstream.Msg) {
		ch <- string(msg.Data())
	})
	defer consumeCtx.Stop()

	client.Publish(queueStreamConfigForTest(), formatTestQueueSubject("sync-job"), []byte("ok"))

	assertPayload(t, ch, "ok")
}

func TestClientCreatesExpectedStreams(t *testing.T) {
	server := newTestNATSServer(t)

	client := new(_Client)
	client.setConn(connectTestNATS(t, "nats://"+server.Addr().String()))
	defer client.conn.Close()

	client.Publish(broadcastStreamConfigForTest(), formatTestBroadcastSubject("alpha.created"), []byte("event"))
	client.Publish(queueStreamConfigForTest(), formatTestQueueSubject("sync-job"), []byte("task"))

	conn := connectTestNATS(t, "nats://"+server.Addr().String())
	defer conn.Close()

	jsCtx, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("create jetstream context failed: %v", err)
	}

	eventStream, err := jsCtx.Stream(context.Background(), testBroadcastStreamName)
	if err != nil {
		t.Fatalf("read event stream failed: %v", err)
	}
	eventInfo, err := eventStream.Info(context.Background())
	if err != nil {
		t.Fatalf("read event stream info failed: %v", err)
	}
	if eventInfo.Config.Retention != jetstream.InterestPolicy {
		t.Fatalf("unexpected event retention: %v", eventInfo.Config.Retention)
	}

	taskStream, err := jsCtx.Stream(context.Background(), testQueueStreamName)
	if err != nil {
		t.Fatalf("read task stream failed: %v", err)
	}
	taskInfo, err := taskStream.Info(context.Background())
	if err != nil {
		t.Fatalf("read task stream info failed: %v", err)
	}
	if taskInfo.Config.Retention != jetstream.WorkQueuePolicy {
		t.Fatalf("unexpected task retention: %v", taskInfo.Config.Retention)
	}
}

func newTestNATSServer(t *testing.T) *natsserver.Server {
	t.Helper()

	server, err := natsserver.NewServer(&natsserver.Options{
		Port:      -1,
		NoSigs:    true,
		NoLog:     true,
		JetStream: true,
		StoreDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new nats server failed: %v", err)
	}
	go server.Start()
	if !server.ReadyForConnections(2 * time.Second) {
		t.Fatalf("nats server not ready")
	}
	t.Cleanup(server.Shutdown)
	return server
}

func connectTestNATS(t *testing.T, endpoint string) *gonats.Conn {
	t.Helper()

	conn, err := newNATSConnect(endpoint)
	if err != nil {
		t.Fatalf("connect nats failed: %v", err)
	}
	return conn
}

func assertPayload(t *testing.T, ch <-chan string, expected string) {
	t.Helper()

	select {
	case payload := <-ch:
		if payload != expected {
			t.Fatalf("unexpected payload: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("subscribe timeout")
	}
}

func broadcastStreamConfigForTest() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      testBroadcastStreamName,
		Subjects:  []string{formatTestBroadcastSubject(">")},
		Retention: jetstream.InterestPolicy,
		Storage:   jetstream.MemoryStorage,
	}
}

func queueStreamConfigForTest() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      testQueueStreamName,
		Subjects:  []string{formatTestQueueSubject(">")},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.MemoryStorage,
	}
}

func formatTestBroadcastSubject(subjectName string) string {
	return fmt.Sprintf(testBroadcastSubjectFmt, subjectName)
}

func formatTestBroadcastConsumer(subjectName string, consumerGroup string) string {
	return fmt.Sprintf(testBroadcastConsumerFmt, subjectName, consumerGroup)
}

func formatTestQueueSubject(subjectName string) string {
	return fmt.Sprintf(testQueueSubjectFmt, subjectName)
}

func formatTestQueueConsumer(subjectName string) string {
	return fmt.Sprintf(testQueueConsumerFmt, subjectName)
}
