package admin

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type queueRepoStub struct {
	statuses []core.MessageQueueStatus
	err      error
	ctx      context.Context
}

func (r *queueRepoStub) List(ctx context.Context) ([]core.MessageQueueStatus, error) {
	r.ctx = ctx
	return r.statuses, r.err
}

func TestMessageQueueServicePreservesCountsAndRequestContext(t *testing.T) {
	ctx := meta.NewContext(t.Context(), nil, nil, nil)
	repo := &queueRepoStub{statuses: []core.MessageQueueStatus{{Kind: "task", Stream: "VINE_TASKS", Exists: true, Messages: math.MaxUint64, Bytes: math.MaxUint64, Subjects: []core.MessageQueueSubject{{Subject: "task.demo", Messages: math.MaxUint64}}, Consumers: []core.MessageQueueConsumer{{Name: "demo", FilterSubjects: []string{"task.demo"}, Pending: math.MaxUint64, AckPending: 2, Redelivered: 1, Waiting: 3}}}}}
	service := &MessageQueueStatusApiServiceServerImpl{Context: ctx, MessageQueueRepo: repo}
	items := service.List()
	require.Same(t, ctx, repo.ctx)
	require.Equal(t, "18446744073709551615", items[0].Messages)
	require.Equal(t, items[0].Messages, items[0].Bytes)
	require.Equal(t, items[0].Messages, items[0].Subjects[0].Messages)
	require.Equal(t, items[0].Messages, items[0].Consumers[0].Pending)
	require.Equal(t, 2, items[0].Consumers[0].AckPending)
	require.Equal(t, 1, items[0].Consumers[0].Redelivered)
	require.Equal(t, 3, items[0].Consumers[0].Waiting)
}

func TestMessageQueueServiceDoesNotReturnConnectionSecrets(t *testing.T) {
	service := &MessageQueueStatusApiServiceServerImpl{Context: meta.NewContext(t.Context(), nil, nil, nil), MessageQueueRepo: &queueRepoStub{err: errors.New("nats://user:secret@host")}}
	require.PanicsWithError(t, "Unable to read message queue status type=SYSTEM code=SERVICE_UNAVAILABLE", func() { service.List() })
}
