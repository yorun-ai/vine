package skeled

import "go.yorun.ai/vine/internal/core/skel"

// EventOn Link triggers event processing information to the App
type EventOn struct {
	// Metadata Event meta information
	Metadata EventOnMeta `json:"metadata"`
	// EventSkelName Event Skel name
	EventSkelName string `json:"eventSkelName"`
	// EventJson Event JSON
	EventJson string `json:"eventJson"`
}

// EventOnMeta Link triggers event processing meta-information to the App
type EventOnMeta struct {
	// TraceId Trace ID of the current calling link
	TraceId string `json:"traceId"`
	// TraceSpan Trace Span of the current calling link
	TraceSpan string `json:"traceSpan"`
	// AppName The application name that sent the event
	AppName string `json:"appName"`
	// AppVersion The application version that sent the event
	AppVersion string `json:"appVersion"`
	// AppInstanceId The application instance ID that sent the event
	AppInstanceId skel.UUID `json:"appInstanceId"`
	// EmittedAt The time the event was sent
	EmittedAt skel.Timestamp `json:"emittedAt"`
}

// TaskRun Link triggers task execution information to the App
type TaskRun struct {
	// Metadata Task trigger meta information
	Metadata TaskRunMeta `json:"metadata"`
	// TaskSkelName Task Skel name
	TaskSkelName string `json:"taskSkelName"`
	// TriggerSkelName Task trigger Skel name
	TriggerSkelName string `json:"triggerSkelName"`
	// ArgumentsJson Task parameters JSON
	ArgumentsJson string `json:"argumentsJson"`
}

// TaskRunMeta Link triggers meta-information of task execution to App
type TaskRunMeta struct {
	// TraceId Trace ID of the current calling link
	TraceId string `json:"traceId"`
	// TraceSpan Trace Span of the current calling link
	TraceSpan string `json:"traceSpan"`
	// AppName Application name that initiated the task
	AppName string `json:"appName"`
	// AppVersion The application version that initiated the task
	AppVersion string `json:"appVersion"`
	// AppInstanceId The application instance ID that initiated the task
	AppInstanceId skel.UUID `json:"appInstanceId"`
	// LaunchedAt Task launch time
	LaunchedAt skel.Timestamp `json:"launchedAt"`
}
