package core

import (
	"time"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon"
)

type ServiceHandlerRegistration struct {
	ServiceSkelName string
	DescriptorHash  string
	Endpoint        string
}

type WebHandlerRegistration struct {
	WebSkelName    string
	DescriptorHash string
	Endpoint       string
}

type EventListenerRegistration struct {
	EventSkelName  string
	DescriptorHash string
	TimeoutMs      int
	Concurrency    int
	NoRetry        bool
}

type TaskRunnerRegistration struct {
	TaskSkelName   string
	DescriptorHash string
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
	Name              string
	InstanceId        string
	Version           string
	Endpoint          string
	ServiceHandlers   []ServiceHandlerRegistration
	WebHandlers       []WebHandlerRegistration
	EventListeners    []EventListenerRegistration
	TaskRunners       []TaskRunnerRegistration
	DomainDescriptors []*skeldesc.Domain
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

type DomainDescriptorVersion struct {
	Descriptor         *skeldesc.Domain
	MainDescriptorHash string
	Main               bool
	MultiVersion       bool
}

type DescriptorVersion[T any] struct {
	Descriptor           T
	Domain               string
	SkelName             string
	DescriptorHash       string
	MainDescriptorHash   string
	Main                 bool
	MultiVersion         bool
	DomainDescriptorHash string
}

type DomainDescriptorView struct {
	DomainVersion DomainDescriptorVersion
	Actors        []DescriptorVersion[*skeldesc.Actor]
	Configs       []DescriptorVersion[*skeldesc.Config]
	Data          []DescriptorVersion[*skeldesc.Data]
	Enums         []DescriptorVersion[*skeldesc.Enum]
	Events        []DescriptorVersion[*skeldesc.Event]
	Resources     []DescriptorVersion[*skeldesc.Resource]
	Services      []DescriptorVersion[*skeldesc.Service]
	Tasks         []DescriptorVersion[*skeldesc.Task]
	Webs          []DescriptorVersion[*skeldesc.Web]
}

type RpcServiceRegistration struct {
	Endpoint       string
	ServerIdentity daemon.Identity
	ServiceName    string
	Api            bool
	AppName        string
	AppVersion     string
	AppInstanceId  string
}

type WebRegistration struct {
	Endpoint       string
	ServerIdentity daemon.Identity
	WebSkelName    string
	AppName        string
	AppVersion     string
	AppInstanceId  string
}

// RegistryRepo tracks several entity kinds at once - application instances, Rpc
// service registrations, Web registrations and their leases - so every method names
// the entity it addresses.
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

// DescriptorRepo stores the descriptors applications register and selects the versions
// the Hub serves.
type DescriptorRepo interface {
	SaveDomainDescriptors(ownerName string, ownerId string, descriptors []*skeldesc.Domain)
	ReleaseDomainDescriptors(ownerName string, ownerId string)

	ListDomainDescriptorViews() []DomainDescriptorView
	ListVineHubDescriptorViews() []DomainDescriptorView
	ListActorDescriptorVersions() []DescriptorVersion[*skeldesc.Actor]
	ListConfigDescriptorVersions() []DescriptorVersion[*skeldesc.Config]
	ListDataDescriptorVersions() []DescriptorVersion[*skeldesc.Data]
	ListEnumDescriptorVersions() []DescriptorVersion[*skeldesc.Enum]
	ListEventDescriptorVersions() []DescriptorVersion[*skeldesc.Event]
	ListResourceDescriptorVersions() []DescriptorVersion[*skeldesc.Resource]
	ListServiceDescriptorVersions() []DescriptorVersion[*skeldesc.Service]
	ListTaskDescriptorVersions() []DescriptorVersion[*skeldesc.Task]
	ListWebDescriptorVersions() []DescriptorVersion[*skeldesc.Web]

	ListActorDescriptors() []*skeldesc.Actor
	ListServiceDescriptors() []*skeldesc.Service
	ListWebDescriptors() []*skeldesc.Web
	// GetWebDescriptor returns the selected descriptor by fully qualified Skel name, or nil if absent.
	GetWebDescriptor(skelName string) *skeldesc.Web

	ListAppConfigDescriptors() []*skeldesc.Config
	ListEnumDescriptors() []*skeldesc.Enum
	ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data)
}
