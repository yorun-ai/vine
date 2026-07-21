package task

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerLaunchesNATSMessage(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ch := make(chan jetstream.Msg, 1)
	launch := skeled.TaskLaunch{
		Metadata: skeled.TaskLaunchMeta{
			TraceId:       "trace-1",
			TraceSpan:     "span-1",
			AppName:       "demo.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("11111111-1111-1111-1111-111111111111")),
		},
		TaskSkelName:    "demo.user.SyncUserTask",
		TriggerSkelName: "demo.user.SyncUserTaskManualTrigger",
		ArgumentsJson:   `{"userId":"u1"}`,
	}
	consumeCtx := manager.NATSClient.Consume(taskStreamConfig(), taskspec.NATSSubject(launch.TaskSkelName), taskspec.NATSConsumerName(launch.TaskSkelName), func(msg jetstream.Msg) {
		ch <- msg
	})
	defer consumeCtx.Stop()
	manager.LaunchTask(launch)

	select {
	case msg := <-ch:
		got := vcode.MustUnmarshalJson[*taskspec.NATSMessage](msg.Data())
		assert.Equal(t, launch.Metadata.TraceId, got.Metadata.TraceId)
		assert.Equal(t, launch.Metadata.TraceSpan, got.Metadata.TraceSpan)
		assert.Equal(t, launch.Metadata.AppName, got.Metadata.AppName)
		assert.Equal(t, launch.Metadata.AppVersion, got.Metadata.AppVersion)
		assert.Equal(t, launch.Metadata.AppInstanceId, got.Metadata.AppInstanceId)
		assert.Equal(t, launch.TaskSkelName, got.TaskSkelName)
		assert.Equal(t, launch.TriggerSkelName, got.TriggerSkelName)
		assert.Equal(t, launch.ArgumentsJson, got.ArgumentsJson)
		assert.NotZero(t, got.Metadata.LaunchedAt)
	case <-time.After(2 * time.Second):
		t.Fatal("task subject timeout")
	}
}
