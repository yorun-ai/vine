package event

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerEmitsNATSMessage(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	emission := skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "span-1",
			AppName:       "demo.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111")),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	}
	ch := make(chan jetstream.Msg, 1)
	consumeCtx := manager.NATSClient.Consume(eventStreamConfig(), eventspec.NATSSubject(emission.EventSkelName), eventspec.NATSConsumerName(emission.EventSkelName, "test.app"), func(msg jetstream.Msg) {
		ch <- msg
	})
	defer consumeCtx.Stop()
	manager.EmitEvent(emission)

	select {
	case msg := <-ch:
		got := vcode.MustUnmarshalJson[*eventspec.NATSMessage](msg.Data())
		assert.Equal(t, emission.Metadata.TraceId, got.Metadata.TraceId)
		assert.Equal(t, emission.Metadata.TraceSpan, got.Metadata.TraceSpan)
		assert.Equal(t, emission.Metadata.AppName, got.Metadata.AppName)
		assert.Equal(t, emission.Metadata.AppVersion, got.Metadata.AppVersion)
		assert.Equal(t, emission.Metadata.AppInstanceId, got.Metadata.AppInstanceId)
		assert.Equal(t, emission.EventSkelName, got.EventSkelName)
		assert.Equal(t, emission.EventJson, got.EventJson)
		assert.NotZero(t, got.Metadata.EmittedAt)
	case <-time.After(2 * time.Second):
		t.Fatal("event subject timeout")
	}
}
