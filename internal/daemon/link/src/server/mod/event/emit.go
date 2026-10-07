package event

import (
	"github.com/nats-io/nats.go/jetstream"
	skeltype "go.yorun.ai/skel/types"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/util/vcode"
)

func (m *Manager) EmitEvent(emission skeled.EventEmission) {
	msg := eventspec.NATSMessage{
		Metadata: eventspec.NATSMessageMeta{
			TraceId:       emission.Metadata.TraceId,
			TraceSpan:     emission.Metadata.TraceSpan,
			AppName:       emission.Metadata.AppName,
			AppVersion:    emission.Metadata.AppVersion,
			AppInstanceId: emission.Metadata.AppInstanceId,
			EmittedAt:     skeltype.NewTimestampNow(),
		},
		EventSkelName: emission.EventSkelName,
		EventJson:     emission.EventJson,
	}
	m.NATSClient.Publish(eventStreamConfig(), eventspec.NATSSubject(emission.EventSkelName), vcode.MustMarshalJson(msg))
}

func eventStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      eventspec.NATSStreamName,
		Subjects:  []string{eventspec.NATSSubject(">")},
		Retention: jetstream.InterestPolicy,
	}
}
