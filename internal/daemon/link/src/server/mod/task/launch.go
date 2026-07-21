package task

import (
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vcode"
)

func (m *Manager) LaunchTask(launch skeled.TaskLaunch) {
	msg := taskspec.NATSMessage{
		Metadata: taskspec.NATSMessageMeta{
			TraceId:       launch.Metadata.TraceId,
			TraceSpan:     launch.Metadata.TraceSpan,
			AppName:       launch.Metadata.AppName,
			AppVersion:    launch.Metadata.AppVersion,
			AppInstanceId: launch.Metadata.AppInstanceId,
			LaunchedAt:    skel.NewTimestampNow(),
		},
		TaskSkelName:    launch.TaskSkelName,
		TriggerSkelName: launch.TriggerSkelName,
		ArgumentsJson:   launch.ArgumentsJson,
	}
	m.NATSClient.Publish(taskStreamConfig(), taskspec.NATSSubject(launch.TaskSkelName), vcode.MustMarshalJson(msg))
}

func taskStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      taskspec.NATSStreamName,
		Subjects:  []string{taskspec.NATSSubject(">")},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.MemoryStorage,
	}
}
