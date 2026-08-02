package scheduler

import (
	"context"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

const schedulerNatsReadyTimeout = time.Second

type _NATSTaskPublisher struct {
	NATSServer *natsserver.NATSServer `inject:""`
	Flag       *hubflag.Flag          `inject:""`
}

func (p *_NATSTaskPublisher) PublishTask(message taskspec.NATSMessage) {
	conn := p.connect()
	defer conn.Close()

	js, err := jetstream.New(conn)
	vpre.CheckNilError(err, "create nats jetstream context failed")

	ctx, cancel := context.WithTimeout(context.Background(), schedulerNatsReadyTimeout)
	defer cancel()
	_, err = js.CreateOrUpdateStream(ctx, taskStreamConfig())
	vpre.CheckNilError(err, "create task nats jetstream stream failed")
	_, err = js.Publish(ctx, taskspec.NATSSubject(message.TaskSkelName), vcode.MustMarshalJson(message))
	vpre.CheckNilError(err, "publish task nats jetstream message failed")
}

func (p *_NATSTaskPublisher) connect() *gonats.Conn {
	if hubnats.InprocServer() != nil {
		return hubnats.ConnectInproc()
	}
	if p.Flag.MQExternalNatsURL == "" {
		vpre.CheckNotNil(p.NATSServer, "embedded nats server is nil")
		conn, err := p.NATSServer.ConnectAsHub()
		vpre.CheckNilError(err, "connect embedded nats as hub failed")
		return conn
	}
	conn, err := gonats.Connect(p.Flag.MQExternalNatsURL)
	vpre.CheckNilError(err, "connect external nats failed")
	return conn
}

func taskStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      taskspec.NATSStreamName,
		Subjects:  []string{taskspec.NATSSubject(">")},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.MemoryStorage,
	}
}
