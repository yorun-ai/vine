package admin

import (
	"strconv"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type MessageQueueStatusApiServiceServerImpl struct {
	skeled.DefaultMessageQueueStatusApiServiceServer
	Context          meta.Context          `inject:""`
	MessageQueueRepo core.MessageQueueRepo `inject:""`
}

func (s *MessageQueueStatusApiServiceServerImpl) List() []skeled.MessageQueueStatusView {
	statuses, err := s.MessageQueueRepo.List(s.Context)
	if err != nil {
		// Connection errors can include endpoint credentials. Keep them out of
		// the browser response; a failed snapshot must not look like zero backlog.
		ex.PanicNew(ex.ServiceUnavailable, "Unable to read message queue status")
	}
	items := make([]skeled.MessageQueueStatusView, 0, len(statuses))
	for _, status := range statuses {
		item := skeled.MessageQueueStatusView{
			Kind: status.Kind, Stream: status.Stream, Exists: status.Exists,
			Messages: strconv.FormatUint(status.Messages, 10), Bytes: strconv.FormatUint(status.Bytes, 10),
			Subjects: []skeled.MessageQueueSubject{}, Consumers: []skeled.MessageQueueConsumer{},
		}
		for _, subject := range status.Subjects {
			item.Subjects = append(item.Subjects, skeled.MessageQueueSubject{Subject: subject.Subject, Messages: strconv.FormatUint(subject.Messages, 10)})
		}
		for _, consumer := range status.Consumers {
			item.Consumers = append(item.Consumers, skeled.MessageQueueConsumer{
				Name: consumer.Name, FilterSubjects: consumer.FilterSubjects, Pending: strconv.FormatUint(consumer.Pending, 10),
				AckPending: consumer.AckPending, Redelivered: consumer.Redelivered, Waiting: consumer.Waiting,
			})
		}
		items = append(items, item)
	}
	return items
}
