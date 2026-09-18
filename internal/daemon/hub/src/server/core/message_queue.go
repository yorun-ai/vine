package core

import "context"

type MessageQueueStatus struct {
	Kind      string
	Stream    string
	Exists    bool
	Messages  uint64
	Bytes     uint64
	Subjects  []MessageQueueSubject
	Consumers []MessageQueueConsumer
}

type MessageQueueSubject struct {
	Subject  string
	Messages uint64
}

type MessageQueueConsumer struct {
	Name           string
	FilterSubjects []string
	Pending        uint64
	AckPending     int
	Redelivered    int
	Waiting        int
}

type MessageQueueRepo interface {
	List(ctx context.Context) ([]MessageQueueStatus, error)
}
