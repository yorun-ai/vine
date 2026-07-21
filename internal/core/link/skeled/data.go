package skeled

import (
	rpc "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
)

// AppRegistration Application information registered by App to Link
type AppRegistration struct {
	// ConsoleEndpoint Console access endpoint of the current application
	ConsoleEndpoint string `json:"consoleEndpoint"`
	// ServiceEndpoint The Rpc service access endpoint of the current application
	ServiceEndpoint string `json:"serviceEndpoint"`
	// WebEndpointPrefix Web access endpoint prefix of the current application
	WebEndpointPrefix string `json:"webEndpointPrefix"`
	// EventEndpoint The event processing access endpoint of the current application
	EventEndpoint string `json:"eventEndpoint"`
	// TaskEndpoint The task execution endpoint of the current application
	TaskEndpoint string `json:"taskEndpoint"`
	// ServiceHandlers List of Rpc service processing capabilities provided by the current application
	ServiceHandlers []ServiceHandlerRegistration `json:"serviceHandlers"`
	// WebHandlers List of web processing capabilities provided by the current application
	WebHandlers []WebHandlerRegistration `json:"webHandlers"`
	// EventListeners List of event listening capabilities provided by the current application
	EventListeners []EventListenerRegistration `json:"eventListeners"`
	// TaskRunners List of task execution capabilities provided by the current application
	TaskRunners []TaskRunnerRegistration `json:"taskRunners"`
	// DomainSchemas List of all DomainSchemas registered by the current application
	DomainSchemas []skel.JSON `json:"domainSchemas"`
}

func (v *AppRegistration) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.ServiceHandlers, rpc.JoinPath(path, "ServiceHandlers")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.WebHandlers, rpc.JoinPath(path, "WebHandlers")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.EventListeners, rpc.JoinPath(path, "EventListeners")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.TaskRunners, rpc.JoinPath(path, "TaskRunners")); err != nil {
		return err
	}
	for i0 := range v.TaskRunners {
		if err := (&v.TaskRunners[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "TaskRunners"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.DomainSchemas, rpc.JoinPath(path, "DomainSchemas")); err != nil {
		return err
	}
	return nil
}

// BootInfo Key information obtained from Link when the App starts
type BootInfo struct {
	// RpcProxyEndpointPath The endpoint path that should be accessed when the current application initiates Rpc
	RpcProxyEndpointPath string `json:"rpcProxyEndpointPath"`
	// SkipDomainSchemas Whether to skip DomainSchema when app is registered
	SkipDomainSchemas bool `json:"skipDomainSchemas"`
}

// EventEmission Event dispatch information sent from App to Link
type EventEmission struct {
	// Metadata Event meta information
	Metadata EventEmissionMeta `json:"metadata"`
	// EventSkelName Event Skel name
	EventSkelName string `json:"eventSkelName"`
	// EventJson Event JSON
	EventJson string `json:"eventJson"`
}

// EventEmissionMeta Meta information carried when App initiates event sending to Link
type EventEmissionMeta struct {
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
}

// EventListenerRegistration Event listening capability registration information provided by the application
type EventListenerRegistration struct {
	// EventSkelName Event Skel name
	EventSkelName string `json:"eventSkelName"`
	// SchemaHash Event schema hash
	SchemaHash string `json:"schemaHash"`
	// TimeoutMs Execution timeout, in milliseconds
	TimeoutMs int `json:"timeoutMs"`
	// Concurrency Maximum concurrency
	Concurrency int `json:"concurrency"`
	// NoRetry Whether to disallow retrying after failure
	NoRetry bool `json:"noRetry"`
}

// ServiceHandlerRegistration Rpc service processing capability registration information provided by the application
type ServiceHandlerRegistration struct {
	// ServiceSkelName Service Skel name
	ServiceSkelName string `json:"serviceSkelName"`
	// SchemaHash Service schema hash
	SchemaHash string `json:"schemaHash"`
}

// TaskLaunch Task trigger information initiated by App to Link
type TaskLaunch struct {
	// Metadata Task trigger meta information
	Metadata TaskLaunchMeta `json:"metadata"`
	// TaskSkelName Task Skel name
	TaskSkelName string `json:"taskSkelName"`
	// TriggerSkelName Task trigger Skel name
	TriggerSkelName string `json:"triggerSkelName"`
	// ArgumentsJson Task parameters JSON
	ArgumentsJson string `json:"argumentsJson"`
}

// TaskLaunchMeta The meta information carried when the App initiates a task to Link
type TaskLaunchMeta struct {
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
}

// TaskRunnerCronScheduler Task execution Cron schedule
type TaskRunnerCronScheduler struct {
	// TriggerSkelName Trigger Skel name
	TriggerSkelName string `json:"triggerSkelName"`
	// CronExpr Cron expression
	CronExpr string `json:"cronExpr"`
}

// TaskRunnerRegistration Task execution capability registration information provided by the application
type TaskRunnerRegistration struct {
	// TaskSkelName Task Skel name
	TaskSkelName string `json:"taskSkelName"`
	// SchemaHash Task schema hash
	SchemaHash string `json:"schemaHash"`
	// TimeoutMs Execution timeout, in milliseconds
	TimeoutMs int `json:"timeoutMs"`
	// Concurrency Maximum concurrency
	Concurrency int `json:"concurrency"`
	// NoRetry Whether to disallow retrying after failure
	NoRetry bool `json:"noRetry"`
	// CronSchedulers Cron schedule list
	CronSchedulers []TaskRunnerCronScheduler `json:"cronSchedulers"`
}

func (v *TaskRunnerRegistration) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.CronSchedulers, rpc.JoinPath(path, "CronSchedulers")); err != nil {
		return err
	}
	return nil
}

// WebHandlerRegistration Web processing capability registration information provided by the application
type WebHandlerRegistration struct {
	// WebSkelName Web Skel name
	WebSkelName string `json:"webSkelName"`
	// SchemaHash Web schema hash
	SchemaHash string `json:"schemaHash"`
}
