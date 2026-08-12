package scheduler

import (
	"context"
	"fmt"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
)

const schedulerNatsReadyTimeout = time.Second

type _NATSTaskPublisher struct {
	NATSServer *natsserver.NATSServer `inject:""`
	Flag       *hubflag.Flag          `inject:""`
}

func (p *_NATSTaskPublisher) PublishTask(message taskspec.NATSMessage) error {
	conn, err := p.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	js, err := jetstream.New(conn)
	if err != nil {
		return fmt.Errorf("create nats jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), schedulerNatsReadyTimeout)
	defer cancel()
	_, err = js.Stream(ctx, taskspec.NATSStreamName)
	if err != nil {
		return fmt.Errorf("read task nats jetstream stream: %w", err)
	}
	payload, err := vcode.MarshalJson(message)
	if err != nil {
		return fmt.Errorf("marshal task nats jetstream message: %w", err)
	}
	if _, err = js.Publish(ctx, taskspec.NATSSubject(message.TaskSkelName), payload); err != nil {
		return fmt.Errorf("publish task nats jetstream message: %w", err)
	}
	return nil
}

func (p *_NATSTaskPublisher) connect() (*gonats.Conn, error) {
	if server := hubnats.InprocServer(); server != nil {
		conn, err := gonats.Connect("", gonats.InProcessServer(server), gonats.Timeout(schedulerNatsReadyTimeout))
		if err != nil {
			return nil, fmt.Errorf("connect inproc nats: %w", err)
		}
		return conn, nil
	}
	if p.Flag == nil {
		return nil, fmt.Errorf("scheduler nats flag is nil")
	}
	if p.Flag.MQExternalNatsURL == "" {
		if p.NATSServer == nil {
			return nil, fmt.Errorf("embedded nats server is nil")
		}
		conn, err := p.NATSServer.ConnectAsHub()
		if err != nil {
			return nil, fmt.Errorf("connect embedded nats as hub: %w", err)
		}
		return conn, nil
	}
	conn, err := gonats.Connect(p.Flag.MQExternalNatsURL, gonats.Timeout(schedulerNatsReadyTimeout))
	if err != nil {
		return nil, fmt.Errorf("connect external nats: %w", err)
	}
	return conn, nil
}
