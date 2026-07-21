package debug

import (
	"context"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

const (
	debugNatsReadyTimeout = time.Second
)

type _DebugNATSPublisher struct {
	NATSServer *natsserver.NATSServer `inject:""`
	Flag       *hubflag.Flag          `inject:""`
}

func (p *_DebugNATSPublisher) publish(stream jetstream.StreamConfig, subject string, payload []byte) {
	conn := p.connect()
	defer conn.Close()

	js, err := jetstream.New(conn)
	vpre.CheckNilError(err, "create nats jetstream context failed")
	ctx, cancel := context.WithTimeout(context.Background(), debugNatsReadyTimeout)
	defer cancel()
	_, err = js.CreateOrUpdateStream(ctx, stream)
	vpre.CheckNilError(err, "create nats jetstream stream failed")
	_, err = js.Publish(ctx, subject, payload)
	vpre.CheckNilError(err, "publish nats jetstream message failed")
}

func (p *_DebugNATSPublisher) connect() *gonats.Conn {
	if hubnats.InprocServer() != nil {
		return hubnats.ConnectInproc()
	}
	endpoint := p.Flag.MQExternalNatsURL
	if endpoint == "" && p.NATSServer != nil {
		endpoint = p.NATSServer.Endpoint()
	}
	vpre.CheckNotEmpty(endpoint, "nats endpoint is empty")
	conn, err := gonats.Connect(endpoint)
	vpre.CheckNilError(err, "connect nats failed")
	return conn
}

func debugTaskStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      taskspec.NATSStreamName,
		Subjects:  []string{taskspec.NATSSubject(">")},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.MemoryStorage,
	}
}

func debugEventStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      eventspec.NATSStreamName,
		Subjects:  []string{eventspec.NATSSubject(">")},
		Retention: jetstream.InterestPolicy,
		Storage:   jetstream.MemoryStorage,
	}
}

func debugTaskSubject(taskSkelName string) string {
	return taskspec.NATSSubject(taskSkelName)
}

func debugEventSubject(eventSkelName string) string {
	return eventspec.NATSSubject(eventSkelName)
}
