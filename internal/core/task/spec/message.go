package spec

import (
	"fmt"

	skeltype "go.yorun.ai/skel/types"
)

const (
	NATSStreamName         = "VINE_TASKS"
	NATSSubjectFormat      = "task.%s"
	NATSConsumerNameFormat = "taskconsumer_%s"
)

func NATSSubject(taskSkelName string) string {
	return fmt.Sprintf(NATSSubjectFormat, taskSkelName)
}

func NATSConsumerName(taskSkelName string) string {
	return fmt.Sprintf(NATSConsumerNameFormat, taskSkelName)
}

type NATSMessage struct {
	Metadata NATSMessageMeta `json:"metadata"`

	TaskSkelName    string `json:"taskSkelName"`
	TriggerSkelName string `json:"triggerSkelName"`
	ArgumentsJson   string `json:"argumentsJson"`
}

type NATSMessageMeta struct {
	TraceId   string `json:"traceId"`
	TraceSpan string `json:"traceSpan"`

	AppName       string        `json:"appName"`
	AppVersion    string        `json:"appVersion"`
	AppInstanceId skeltype.UUID `json:"appInstanceId"`

	LaunchedAt skeltype.Timestamp `json:"launchedAt"`
}
