package core

import (
	"time"

	"go.yorun.ai/vine/internal/core/skel"
)

type ServiceHandlerRegistration struct {
	ServiceSkelName string
	SchemaHash      string
	Endpoint        string
}

type WebHandlerRegistration struct {
	WebSkelName string
	SchemaHash  string
	Endpoint    string
}

type EventListenerRegistration struct {
	EventSkelName string
	SchemaHash    string
	TimeoutMs     int
	Concurrency   int
	NoRetry       bool
}

type TaskRunnerRegistration struct {
	TaskSkelName   string
	SchemaHash     string
	TimeoutMs      int
	Concurrency    int
	NoRetry        bool
	CronSchedulers []TaskRunnerCronScheduler
}

type TaskRunnerCronScheduler struct {
	TriggerSkelName string
	CronExpr        string
}

type AppRegistration struct {
	Name            string
	InstanceId      string
	Version         string
	Endpoint        string
	ServiceHandlers []ServiceHandlerRegistration
	WebHandlers     []WebHandlerRegistration
	EventListeners  []EventListenerRegistration
	TaskRunners     []TaskRunnerRegistration
	DomainSchemas   []skel.JSON
}

type AppHeartbeat struct {
	Name       string
	InstanceId string
}

type AppStatus struct {
	InstanceId      string
	Name            string
	Version         string
	Endpoint        string
	ExpiresAt       time.Time
	ServiceHandlers []ServiceHandlerRegistration
	WebHandlers     []WebHandlerRegistration
	EventListeners  []EventListenerRegistration
	TaskRunners     []TaskRunnerRegistration
}

type DomainSchemaVersion struct {
	Schema         *skel.DomainSchema
	MainSchemaHash string
	Main           bool
	MultiVersion   bool
}

type SchemaVersion[T any] struct {
	Schema           T
	Domain           string
	SkelName         string
	SchemaHash       string
	MainSchemaHash   string
	Main             bool
	MultiVersion     bool
	DomainSchemaHash string
}

type DomainSchemaView struct {
	DomainVersion DomainSchemaVersion
	Actors        []SchemaVersion[*skel.ActorSchema]
	Configs       []SchemaVersion[*skel.ConfigSchema]
	Data          []SchemaVersion[*skel.DataSchema]
	Enums         []SchemaVersion[*skel.EnumSchema]
	Events        []SchemaVersion[*skel.EventSchema]
	Resources     []SchemaVersion[*skel.ResourceSchema]
	Services      []SchemaVersion[*skel.ServiceSchema]
	Tasks         []SchemaVersion[*skel.TaskSchema]
	Webs          []SchemaVersion[*skel.WebSchema]
}

type RpcServiceRegistration struct {
	Endpoint      string
	ServiceName   string
	AppName       string
	AppVersion    string
	AppInstanceId string
}

type WebRegistration struct {
	Endpoint      string
	WebSkelName   string
	AppName       string
	AppVersion    string
	AppInstanceId string
}

type RegistryRepo interface {
	SaveAppStatus(status *AppStatus)
	ListAppStatuses() []*AppStatus
	GetAppStatus(appName string, instanceId string) (*AppStatus, bool)
	KeepAppStatus(appName string, instanceId string) bool
	RemoveAppStatus(appName string, instanceId string)
	PopExpiredAppLeases() []AppHeartbeat

	SaveRpcServiceRegistration(registration *RpcServiceRegistration)
	GetRpcServiceRegistration(serviceName string, appName string, instanceId string) (*RpcServiceRegistration, bool)
	KeepRpcServiceRegistration(serviceName string, appName string, appInstanceId string) bool
	RemoveRpcServiceRegistration(serviceName string, appName string, appInstanceId string)

	SaveWebRegistration(registration *WebRegistration)
	GetWebRegistration(name string, appName string, instanceId string) (*WebRegistration, bool)
	KeepWebRegistration(name string, appName string, appInstanceId string) bool
	RemoveWebRegistration(name string, appName string, appInstanceId string)
}

type SchemaRepo interface {
	SaveDomainSchemas(ownerName string, ownerId string, schemas []*skel.DomainSchema)
	SaveDomainSchemasJSON(ownerName string, ownerId string, schemas []skel.JSON)
	ReleaseDomainSchemas(ownerName string, ownerId string)

	ListDomainSchemaViews() []DomainSchemaView
	ListVineHubSchemaViews() []DomainSchemaView
	ListActorSchemaVersions() []SchemaVersion[*skel.ActorSchema]
	ListConfigSchemaVersions() []SchemaVersion[*skel.ConfigSchema]
	ListDataSchemaVersions() []SchemaVersion[*skel.DataSchema]
	ListEnumSchemaVersions() []SchemaVersion[*skel.EnumSchema]
	ListEventSchemaVersions() []SchemaVersion[*skel.EventSchema]
	ListResourceSchemaVersions() []SchemaVersion[*skel.ResourceSchema]
	ListServiceSchemaVersions() []SchemaVersion[*skel.ServiceSchema]
	ListTaskSchemaVersions() []SchemaVersion[*skel.TaskSchema]
	ListWebSchemaVersions() []SchemaVersion[*skel.WebSchema]

	ListActorSchemas() []*skel.ActorSchema
	ListServiceSchemas() []*skel.ServiceSchema
	ListWebSchemas() []*skel.WebSchema

	ListAppConfigSchemas() []*skel.ConfigSchema
	ListEnumSchemas() []*skel.EnumSchema
}
