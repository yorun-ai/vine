package spec

import (
	"fmt"

	skeltype "go.yorun.ai/skel/types"
)

const (
	NATSStreamName         = "VINE_EVENTS"
	NATSSubjectFormat      = "event.%s"
	NATSConsumerNameFormat = "eventconsumer_%s_%s"
)

func NATSSubject(eventSkelName string) string {
	return fmt.Sprintf(NATSSubjectFormat, eventSkelName)
}

func NATSConsumerName(eventSkelName string, appName string) string {
	return fmt.Sprintf(NATSConsumerNameFormat, eventSkelName, appName)
}

type NATSMessage struct {
	Metadata NATSMessageMeta `json:"metadata"`

	EventSkelName string `json:"eventSkelName"`
	EventJson     string `json:"eventJson"`
}

type NATSMessageMeta struct {
	TraceId   string `json:"traceId"`
	TraceSpan string `json:"traceSpan"`

	AppName       string        `json:"appName"`
	AppVersion    string        `json:"appVersion"`
	AppInstanceId skeltype.UUID `json:"appInstanceId"`

	EmittedAt skeltype.Timestamp `json:"emittedAt"`
}
