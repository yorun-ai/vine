package repo

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/app"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type MessageQueueRepo struct {
	InprocFlag *app.InternalInprocFlag `inject:""`
	Flag       *flag.Flag              `inject:""`
	NATSServer *natsserver.NATSServer  `inject:""`
}

// List reads broker state, including retained subjects with no consumer and
// durable consumers whose applications are currently offline. It never creates
// streams, consumers or subscriptions for message delivery.
func (r *MessageQueueRepo) List(ctx context.Context) ([]core.MessageQueueStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := r.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	items := []core.MessageQueueStatus{
		{Kind: "task", Stream: taskspec.NATSStreamName},
		{Kind: "event", Stream: eventspec.NATSStreamName},
	}
	for i := range items {
		item := &items[i]
		item.Subjects = []core.MessageQueueSubject{}
		item.Consumers = []core.MessageQueueConsumer{}
		stream, err := js.Stream(ctx, item.Stream)
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		info, err := stream.Info(ctx, jetstream.WithSubjectFilter(">"))
		if err != nil {
			return nil, err
		}
		item.Exists = true
		item.Messages = info.State.Msgs
		item.Bytes = info.State.Bytes
		for subject, count := range info.State.Subjects {
			item.Subjects = append(item.Subjects, core.MessageQueueSubject{Subject: subject, Messages: count})
		}
		slices.SortFunc(item.Subjects, func(a core.MessageQueueSubject, b core.MessageQueueSubject) int {
			return cmp.Compare(a.Subject, b.Subject)
		})
		consumers := stream.ListConsumers(ctx)
		for info := range consumers.Info() {
			filters := slices.Clone(info.Config.FilterSubjects)
			if info.Config.FilterSubject != "" {
				filters = append(filters, info.Config.FilterSubject)
			}
			if len(filters) == 0 {
				filters = []string{">"}
			}
			slices.Sort(filters)
			item.Consumers = append(item.Consumers, core.MessageQueueConsumer{
				Name: info.Name, FilterSubjects: filters, Pending: info.NumPending,
				AckPending: info.NumAckPending, Redelivered: info.NumRedelivered, Waiting: info.NumWaiting,
			})
		}
		if err := consumers.Err(); err != nil {
			return nil, err
		}
		slices.SortFunc(item.Consumers, func(a core.MessageQueueConsumer, b core.MessageQueueConsumer) int { return cmp.Compare(a.Name, b.Name) })
	}
	return items, nil
}

func (r *MessageQueueRepo) connect() (*gonats.Conn, error) {
	if r.Flag.MQMode == flag.MQModeNATS {
		return gonats.Connect(r.Flag.MQNatsEndpoint, gonats.Timeout(time.Second), gonats.NoReconnect())
	}
	if r.InprocFlag.Enabled {
		return gonats.Connect("", gonats.InProcessServer(hubnats.InprocServer()), gonats.Timeout(time.Second), gonats.NoReconnect())
	}
	return r.NATSServer.ConnectAsHub()
}
