package skeled

import (
	rpc "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
)

// AppConfigCreation Configuration creation parameters
type AppConfigCreation struct {
	// SkelName Configuration Skel name
	SkelName string `json:"skelName"`
	// Value Configuration JSON
	Value string `json:"value"`
}

// AppConfigItem Configuration items
type AppConfigItem struct {
	// Id Configuration ID
	Id int `json:"id"`
	// Key Configuration key
	Key string `json:"key"`
	// Status Configuration status
	Status string `json:"status"`
	// Lifecycle Configuration lifecycle
	Lifecycle string `json:"lifecycle"`
	// Value Configuration JSON
	Value string `json:"value"`
	// Schema Configuration schema
	Schema *AppConfigSchema `json:"schema"`
}

func (v *AppConfigItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Schema != nil {
		if err := v.Schema.Validate(rpc.JoinPath(path, "Schema")); err != nil {
			return err
		}
	}
	return nil
}

// AppConfigSchema Configuration schema items
type AppConfigSchema struct {
	// SkelName Configuration Skel name
	SkelName string `json:"skelName"`
	// Name Configuration name
	Name string `json:"name"`
	// Description Configuration description
	Description *string `json:"description"`
	// Lifecycle Configuration lifecycle
	Lifecycle string `json:"lifecycle"`
	// Fields Configuration field list
	Fields []AppConfigSchemaField `json:"fields"`
}

func (v *AppConfigSchema) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	for i0 := range v.Fields {
		if err := (&v.Fields[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Fields"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// AppConfigSchemaEnumItem Configuration schema enumeration options
type AppConfigSchemaEnumItem struct {
	// Name Enum option name
	Name string `json:"name"`
	// Description Enumeration options description
	Description *string `json:"description"`
}

// AppConfigSchemaField Configuration schema fields
type AppConfigSchemaField struct {
	// Name Field name
	Name string `json:"name"`
	// Type Field type
	Type string `json:"type"`
	// Description Field description
	Description *string `json:"description"`
	// EnumItems Enumeration options list
	EnumItems []AppConfigSchemaEnumItem `json:"enumItems"`
}

func (v *AppConfigSchemaField) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.EnumItems, rpc.JoinPath(path, "EnumItems")); err != nil {
		return err
	}
	return nil
}

// AppConfigUpdate Configuration update parameters
type AppConfigUpdate struct {
	// Value Configuration JSON
	Value *string `json:"value"`
}

// AppRegistration Link application instance information registered with Hub
type AppRegistration struct {
	// Name Application name
	Name string `json:"name"`
	// InstanceId Application instance ID
	InstanceId skel.UUID `json:"instanceId"`
	// Version Application version
	Version string `json:"version"`
	// Endpoint Application access address (e.g. "http://10.1.2.3:23001")
	Endpoint string `json:"endpoint"`
	// ServiceHandlers List of Rpc service processing capabilities provided by the application
	ServiceHandlers []ServiceHandlerRegistration `json:"serviceHandlers"`
	// WebHandlers List of web processing capabilities provided by the application
	WebHandlers []WebHandlerRegistration `json:"webHandlers"`
	// EventListeners List of event listening capabilities provided by the application
	EventListeners []EventListenerRegistration `json:"eventListeners"`
	// TaskRunners List of task execution capabilities provided by the application
	TaskRunners []TaskRunnerRegistration `json:"taskRunners"`
	// DomainSchemas List of all DomainSchemas registered by the application
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

// AppStatus Application instance status information, used for heartbeat refresh
type AppStatus struct {
	// Name Application name
	Name string `json:"name"`
	// InstanceId Application instance ID
	InstanceId skel.UUID `json:"instanceId"`
}

// AppStatusView Application instance status view for Dashboard display
type AppStatusView struct {
	// Name Application name
	Name string `json:"name"`
	// InstanceId Application instance ID
	InstanceId string `json:"instanceId"`
	// Version Application version
	Version string `json:"version"`
	// Endpoint Application access address
	Endpoint string `json:"endpoint"`
	// ServiceHandlers List of Rpc service processing capabilities provided by the application
	ServiceHandlers []ServiceHandlerRegistration `json:"serviceHandlers"`
	// WebHandlers List of web processing capabilities provided by the application
	WebHandlers []WebHandlerRegistration `json:"webHandlers"`
	// EventListeners List of event listening capabilities provided by the application
	EventListeners []EventListenerRegistration `json:"eventListeners"`
	// TaskRunners List of task execution capabilities provided by the application
	TaskRunners []TaskRunnerRegistration `json:"taskRunners"`
}

func (v *AppStatusView) Validate(path string) error {
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
	return nil
}

// EventDebugDefaultEmitRequest Default Event Debug send request
type EventDebugDefaultEmitRequest struct {
	// TraceId Trace ID
	TraceId string `json:"traceId"`
	// SpanId Span ID
	SpanId string `json:"spanId"`
	// EventJson Default event JSON
	EventJson skel.JSON `json:"eventJson"`
}

// EventDebugEmitRequest Event Debug send request
type EventDebugEmitRequest struct {
	// EventSkelName Event Skel name
	EventSkelName string `json:"eventSkelName"`
	// SchemaHash Event schema hash
	SchemaHash string `json:"schemaHash"`
	// EventJson Event JSON
	EventJson skel.JSON `json:"eventJson"`
	// TraceId Trace ID
	TraceId *string `json:"traceId"`
	// SpanId Span ID
	SpanId *string `json:"spanId"`
}

// EventDebugEventItem Event called by Event Debug
type EventDebugEventItem struct {
	// Name Event name
	Name string `json:"name"`
	// EventSkelName Event Skel name
	EventSkelName string `json:"eventSkelName"`
	// SchemaHash Event schema hash
	SchemaHash string `json:"schemaHash"`
	// Description Event description
	Description *string `json:"description"`
	// Fields Field list
	Fields []SkeletonField `json:"fields"`
}

func (v *EventDebugEventItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	return nil
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

// Info Hub information
type Info struct {
	// ApiPort API service port
	ApiPort int `json:"apiPort"`
	// RedisPort Redis service port
	RedisPort int `json:"redisPort"`
	// NatsPort NATS service port
	NatsPort int `json:"natsPort"`
	// MqEndpoint Standalone MQ service address
	MqEndpoint string `json:"mqEndpoint"`
}

// PortalCert Portal site certificate
type PortalCert struct {
	// Id Certificate ID
	Id int `json:"id"`
	// Name Certificate name
	Name string `json:"name"`
	// Issuer Certificate issuer
	Issuer string `json:"issuer"`
	// Domains Certificate domain name
	Domains []string `json:"domains"`
	// PublicKeyBase64 Certificate Base64
	PublicKeyBase64 string `json:"publicKeyBase64"`
	// PrivateKeyConfigured Whether the private key has been configured
	PrivateKeyConfigured bool `json:"privateKeyConfigured"`
	// ValidFrom Validity start time
	ValidFrom skel.Timestamp `json:"validFrom"`
	// ValidTo Validity end time
	ValidTo skel.Timestamp `json:"validTo"`
}

func (v *PortalCert) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Domains, rpc.JoinPath(path, "Domains")); err != nil {
		return err
	}
	return nil
}

// PortalCertCreation Portal site certificate creation parameters
type PortalCertCreation struct {
	// Name Certificate name
	Name string `json:"name"`
	// PublicKeyBase64 Certificate Base64
	PublicKeyBase64 string `json:"publicKeyBase64"`
	// PrivateKeyBase64 Private key Base64
	PrivateKeyBase64 string `json:"privateKeyBase64"`
}

// PortalCertUpdate Portal site certificate update parameters
type PortalCertUpdate struct {
	// Name Certificate name
	Name *string `json:"name"`
	// PublicKeyBase64 Certificate Base64
	PublicKeyBase64 *string `json:"publicKeyBase64"`
	// PrivateKeyBase64 Private key Base64
	PrivateKeyBase64 *string `json:"privateKeyBase64"`
}

// PortalCors Portal site CORS configuration
type PortalCors struct {
	// Mode CORS mode: DISABLED/SAME_DOMAIN/STRICT
	Mode PortalCorsMode `json:"mode"`
	// AllowedOrigins List of origins allowed in strict mode
	AllowedOrigins []string `json:"allowedOrigins"`
}

func (v *PortalCors) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.AllowedOrigins, rpc.JoinPath(path, "AllowedOrigins")); err != nil {
		return err
	}
	return nil
}

// PortalDashboardAccess Hub Dashboard access entry
type PortalDashboardAccess struct {
	// Scheme Entry protocol
	Scheme string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host string `json:"host"`
	// Port Entry port
	Port int `json:"port"`
	// PathPrefix Match path prefix
	PathPrefix string `json:"pathPrefix"`
	// CanUpdate Whether to allow modification of Dashboard access entry
	CanUpdate bool `json:"canUpdate"`
}

// PortalEntry Portal access entry
type PortalEntry struct {
	// Name Entry name
	Name string `json:"name"`
	// Scheme Entry protocol
	Scheme string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host string `json:"host"`
	// Port Entry port
	Port int `json:"port"`
	// Rules Entry rule list
	Rules []PortalEntryRule `json:"rules"`
}

func (v *PortalEntry) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Rules, rpc.JoinPath(path, "Rules")); err != nil {
		return err
	}
	for i0 := range v.Rules {
		if err := (&v.Rules[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Rules"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// PortalEntryAccessUpdate Portal access entry configuration update parameters
type PortalEntryAccessUpdate struct {
	// Scheme Entry protocol
	Scheme string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host string `json:"host"`
	// Port Entry port
	Port int `json:"port"`
}

// PortalEntryRule Portal access entry rules
type PortalEntryRule struct {
	// Rule Entry rules
	Rule PortalRule `json:"rule"`
	// Site Target site
	Site *PortalSite `json:"site"`
}

func (v *PortalEntryRule) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Site != nil {
		if err := v.Site.Validate(rpc.JoinPath(path, "Site")); err != nil {
			return err
		}
	}
	return nil
}

// PortalRule Portal entry rules
type PortalRule struct {
	// Id Rule ID
	Id int `json:"id"`
	// Name Rule name
	Name string `json:"name"`
	// Scheme Matching protocol
	Scheme string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host string `json:"host"`
	// Port Match port, 0 means no restriction
	Port int `json:"port"`
	// PathPrefix Match path prefix, empty string means match all paths
	PathPrefix string `json:"pathPrefix"`
	// TargetType Target type
	TargetType string `json:"targetType"`
	// SiteName Site name
	SiteName string `json:"siteName"`
	// RedirectionPattern Redirect Pattern
	RedirectionPattern string `json:"redirectionPattern"`
}

// PortalRuleCreation Portal entry rule creation parameters
type PortalRuleCreation struct {
	// Name Rule name
	Name string `json:"name"`
	// Scheme Matching protocol
	Scheme string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host string `json:"host"`
	// Port Match port, 0 means no restriction
	Port int `json:"port"`
	// PathPrefix Match path prefix, empty string means match all paths
	PathPrefix string `json:"pathPrefix"`
	// TargetType Target type
	TargetType string `json:"targetType"`
	// SiteName Site name
	SiteName string `json:"siteName"`
	// RedirectionPattern Redirect Pattern
	RedirectionPattern string `json:"redirectionPattern"`
}

// PortalRuleUpdate Portal entry rule update parameters
type PortalRuleUpdate struct {
	// Name Rule name
	Name *string `json:"name"`
	// Scheme Matching protocol
	Scheme *string `json:"scheme"`
	// Host Match Host, empty string means no restriction
	Host *string `json:"host"`
	// Port Match port, 0 means no restriction
	Port *int `json:"port"`
	// PathPrefix Match path prefix, empty string means match all paths
	PathPrefix *string `json:"pathPrefix"`
	// TargetType Target type
	TargetType *string `json:"targetType"`
	// SiteName Site name
	SiteName *string `json:"siteName"`
	// RedirectionPattern Redirect Pattern
	RedirectionPattern *string `json:"redirectionPattern"`
}

// PortalSite Portal target site
type PortalSite struct {
	// Id Target site id
	Id int `json:"id"`
	// Name Target site name
	Name string `json:"name"`
	// Type Target site type
	Type PortalSiteType `json:"type"`
	// ActorSkelName Actor Skel name
	ActorSkelName string `json:"actorSkelName"`
	// ActorVia Actor access method
	ActorVia string `json:"actorVia"`
	// RpcgwServices Rpc gateway service Skel name list
	RpcgwServices []string `json:"rpcgwServices"`
	// Cors CORS configuration
	Cors *PortalCors `json:"cors"`
	// WebName Web name
	WebName string `json:"webName"`
}

func (v *PortalSite) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.RpcgwServices, rpc.JoinPath(path, "RpcgwServices")); err != nil {
		return err
	}
	if v.Cors != nil {
		if err := v.Cors.Validate(rpc.JoinPath(path, "Cors")); err != nil {
			return err
		}
	}
	return nil
}

// PortalSiteActorOption Portal target site Actor options
type PortalSiteActorOption struct {
	// Name Actor name
	Name string `json:"name"`
	// SkelName Actor Skel name
	SkelName string `json:"skelName"`
	// ActorVias Actor access method list
	ActorVias []string `json:"actorVias"`
}

func (v *PortalSiteActorOption) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.ActorVias, rpc.JoinPath(path, "ActorVias")); err != nil {
		return err
	}
	return nil
}

// PortalSiteCreation Portal target site creation parameters
type PortalSiteCreation struct {
	// Name Target site name
	Name string `json:"name"`
	// Type Target site type
	Type PortalSiteType `json:"type"`
	// ActorSkelName Actor Skel name
	ActorSkelName string `json:"actorSkelName"`
	// ActorVia Actor access method
	ActorVia string `json:"actorVia"`
	// Cors CORS configuration
	Cors *PortalCors `json:"cors"`
	// WebName Web name
	WebName string `json:"webName"`
}

func (v *PortalSiteCreation) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Cors != nil {
		if err := v.Cors.Validate(rpc.JoinPath(path, "Cors")); err != nil {
			return err
		}
	}
	return nil
}

// PortalSiteOptions Portal target site form options
type PortalSiteOptions struct {
	// Actors Actor options
	Actors []PortalSiteActorOption `json:"actors"`
	// Services Rpc service options
	Services []PortalSiteServiceOption `json:"services"`
	// Webs Web options
	Webs []PortalSiteWebOption `json:"webs"`
}

func (v *PortalSiteOptions) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Actors, rpc.JoinPath(path, "Actors")); err != nil {
		return err
	}
	for i0 := range v.Actors {
		if err := (&v.Actors[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Actors"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Services, rpc.JoinPath(path, "Services")); err != nil {
		return err
	}
	for i0 := range v.Services {
		if err := (&v.Services[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Services"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Webs, rpc.JoinPath(path, "Webs")); err != nil {
		return err
	}
	for i0 := range v.Webs {
		if err := (&v.Webs[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Webs"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// PortalSiteServiceOption Portal target site service options
type PortalSiteServiceOption struct {
	// Name Service name
	Name string `json:"name"`
	// SkelName Service Skel name
	SkelName string `json:"skelName"`
	// ActorSkelNames Actor Skel name list
	ActorSkelNames []string `json:"actorSkelNames"`
}

func (v *PortalSiteServiceOption) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.ActorSkelNames, rpc.JoinPath(path, "ActorSkelNames")); err != nil {
		return err
	}
	return nil
}

// PortalSiteUpdate Portal target site update parameters
type PortalSiteUpdate struct {
	// Name Target site name
	Name *string `json:"name"`
	// Type Target site type
	Type *PortalSiteType `json:"type"`
	// ActorSkelName Actor Skel name
	ActorSkelName *string `json:"actorSkelName"`
	// ActorVia Actor access method
	ActorVia *string `json:"actorVia"`
	// Cors CORS configuration
	Cors *PortalCors `json:"cors"`
	// WebName Web name
	WebName *string `json:"webName"`
}

func (v *PortalSiteUpdate) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Cors != nil {
		if err := v.Cors.Validate(rpc.JoinPath(path, "Cors")); err != nil {
			return err
		}
	}
	return nil
}

// PortalSiteWebOption Portal target site web options
type PortalSiteWebOption struct {
	// Name Web name
	Name string `json:"name"`
	// SkelName Web Skel name
	SkelName string `json:"skelName"`
	// ActorSkelNames Actor Skel name list
	ActorSkelNames []string `json:"actorSkelNames"`
}

func (v *PortalSiteWebOption) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.ActorSkelNames, rpc.JoinPath(path, "ActorSkelNames")); err != nil {
		return err
	}
	return nil
}

// SeedEntityDiff Seed entity differences
type SeedEntityDiff struct {
	// Kind Entity type
	Kind string `json:"kind"`
	// Name Entity name
	Name string `json:"name"`
	// Exists Whether the entity currently exists
	Exists bool `json:"exists"`
	// Fields Field differences
	Fields []SeedFieldDiff `json:"fields"`
}

func (v *SeedEntityDiff) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	return nil
}

// SeedFieldDiff Seed field differences
type SeedFieldDiff struct {
	// Name Field name
	Name string `json:"name"`
	// CurrentValue Current value
	CurrentValue string `json:"currentValue"`
	// SeedValue Seed value
	SeedValue string `json:"seedValue"`
	// Changed Whether the values differ
	Changed bool `json:"changed"`
}

// SeedItemSelection Seed entity selection
type SeedItemSelection struct {
	// Kind Entity type
	Kind string `json:"kind"`
	// Name Entity name
	Name string `json:"name"`
}

// SeedPreview Seed preview
type SeedPreview struct {
	// Items Entity differences
	Items []SeedEntityDiff `json:"items"`
}

func (v *SeedPreview) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Items, rpc.JoinPath(path, "Items")); err != nil {
		return err
	}
	for i0 := range v.Items {
		if err := (&v.Items[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Items"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// ServiceDebugActorItem Service Debug Actor options
type ServiceDebugActorItem struct {
	// Name Actor name
	Name string `json:"name"`
	// SkelName Actor Skel name
	SkelName string `json:"skelName"`
	// InfoSkelName Actor Info Skel name
	InfoSkelName string `json:"infoSkelName"`
	// ActorInfoJson Default Actor Info JSON
	ActorInfoJson skel.JSON `json:"actorInfoJson"`
}

// ServiceDebugAppInstance Application instance called by Service Debug
type ServiceDebugAppInstance struct {
	// AppName Application name
	AppName string `json:"appName"`
	// AppInstanceId Application instance ID
	AppInstanceId string `json:"appInstanceId"`
	// AppVersion Application version
	AppVersion string `json:"appVersion"`
	// Endpoint Application access address
	Endpoint string `json:"endpoint"`
}

// ServiceDebugDefaultInvokeRequest Service Debug default call request
type ServiceDebugDefaultInvokeRequest struct {
	// TraceId Trace ID
	TraceId string `json:"traceId"`
	// SpanId Span ID
	SpanId string `json:"spanId"`
	// Actors Actor options
	Actors []ServiceDebugActorItem `json:"actors"`
	// ActorSkelName Default Actor Skel name
	ActorSkelName *string `json:"actorSkelName"`
	// ActorInfoJson Default Actor Info JSON
	ActorInfoJson skel.JSON `json:"actorInfoJson"`
	// ParamsJson Default request parameters JSON
	ParamsJson skel.JSON `json:"paramsJson"`
}

func (v *ServiceDebugDefaultInvokeRequest) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Actors, rpc.JoinPath(path, "Actors")); err != nil {
		return err
	}
	return nil
}

// ServiceDebugInvokeRequest Service Debug call request
type ServiceDebugInvokeRequest struct {
	// AppName Application name
	AppName *string `json:"appName"`
	// AppInstanceId Application instance ID
	AppInstanceId *string `json:"appInstanceId"`
	// ServiceSkelName Service Skel name
	ServiceSkelName string `json:"serviceSkelName"`
	// SchemaHash Service schema hash
	SchemaHash string `json:"schemaHash"`
	// MethodSkelName Method Skel name
	MethodSkelName string `json:"methodSkelName"`
	// ParamsJson Request parameters JSON
	ParamsJson skel.JSON `json:"paramsJson"`
	// TimeoutSeconds Call timeout, in seconds
	TimeoutSeconds int `json:"timeoutSeconds"`
	// TraceId Trace ID
	TraceId *string `json:"traceId"`
	// SpanId Span ID
	SpanId *string `json:"spanId"`
	// ActorSkelName Actor Skel name
	ActorSkelName *string `json:"actorSkelName"`
	// ActorInfoJson Actor Info JSON
	ActorInfoJson skel.JSON `json:"actorInfoJson"`
}

// ServiceDebugInvokeResponse Service Debug call response
type ServiceDebugInvokeResponse struct {
	// HttpStatus HTTP status code
	HttpStatus int `json:"httpStatus"`
	// RpcStatus Rpc status code
	RpcStatus string `json:"rpcStatus"`
	// HeadersJson Response header JSON
	HeadersJson skel.JSON `json:"headersJson"`
	// BodyJson Response body JSON
	BodyJson skel.JSON `json:"bodyJson"`
}

// ServiceDebugMethodItem Method called by Service Debug
type ServiceDebugMethodItem struct {
	// Name Method name
	Name string `json:"name"`
	// SkelName Method Skel name
	SkelName string `json:"skelName"`
	// Description Method description
	Description *string `json:"description"`
	// InputDescription Input description
	InputDescription *string `json:"inputDescription"`
	// OutputDescription Output description
	OutputDescription *string `json:"outputDescription"`
	// Example Input example
	Example *string `json:"example"`
	// OutputExample Output example
	OutputExample *string `json:"outputExample"`
	// Arguments Parameter list
	Arguments []SkeletonField `json:"arguments"`
	// ResultType Return type
	ResultType string `json:"resultType"`
}

func (v *ServiceDebugMethodItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
}

// ServiceDebugServiceItem Service called by Service Debug
type ServiceDebugServiceItem struct {
	// ServiceSkelName Service Skel name
	ServiceSkelName string `json:"serviceSkelName"`
	// SchemaHash Service schema hash
	SchemaHash string `json:"schemaHash"`
}

// ServiceHandlerRegistration Rpc service processing capability registration information provided by the application
type ServiceHandlerRegistration struct {
	// ServiceSkelName Service Skel name
	ServiceSkelName string `json:"serviceSkelName"`
	// SchemaHash Service schema hash
	SchemaHash string `json:"schemaHash"`
	// Endpoint Service agent access address
	Endpoint string `json:"endpoint"`
}

// SkeletonActorItem SkeletonActor
type SkeletonActorItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Actor name
	Name string `json:"name"`
	// SkelName Actor Skel name
	SkelName string `json:"skelName"`
	// Description Actor description
	Description *string `json:"description"`
	// ActorVias Actor access method list
	ActorVias []string `json:"actorVias"`
	// AuthEnabled Whether to enable authentication
	AuthEnabled bool `json:"authEnabled"`
	// Credential Authentication credentials
	Credential *SkeletonData `json:"credential"`
	// Info Authentication information
	Info *SkeletonData `json:"info"`
	// AuthService Authentication services
	AuthService *SkeletonServiceItem `json:"authService"`
	// PermEnabled Whether to enable permissions
	PermEnabled bool `json:"permEnabled"`
	// PermService Permission service
	PermService *SkeletonServiceItem `json:"permService"`
	// PermMethod Permission method
	PermMethod *SkeletonMethod `json:"permMethod"`
	// Services Accessible Service List
	Services []SkeletonServiceItem `json:"services"`
	// Webs Accessible web list
	Webs []SkeletonWebItem `json:"webs"`
}

func (v *SkeletonActorItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.ActorVias, rpc.JoinPath(path, "ActorVias")); err != nil {
		return err
	}
	if v.Credential != nil {
		if err := v.Credential.Validate(rpc.JoinPath(path, "Credential")); err != nil {
			return err
		}
	}
	if v.Info != nil {
		if err := v.Info.Validate(rpc.JoinPath(path, "Info")); err != nil {
			return err
		}
	}
	if v.AuthService != nil {
		if err := v.AuthService.Validate(rpc.JoinPath(path, "AuthService")); err != nil {
			return err
		}
	}
	if v.PermService != nil {
		if err := v.PermService.Validate(rpc.JoinPath(path, "PermService")); err != nil {
			return err
		}
	}
	if v.PermMethod != nil {
		if err := v.PermMethod.Validate(rpc.JoinPath(path, "PermMethod")); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Services, rpc.JoinPath(path, "Services")); err != nil {
		return err
	}
	for i0 := range v.Services {
		if err := (&v.Services[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Services"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Webs, rpc.JoinPath(path, "Webs")); err != nil {
		return err
	}
	for i0 := range v.Webs {
		if err := (&v.Webs[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Webs"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonActorRef Skeleton Actor Reference
type SkeletonActorRef struct {
	// Name Actor name
	Name string `json:"name"`
	// SkelName Actor Skel name
	SkelName string `json:"skelName"`
	// Via Access method
	Via *string `json:"via"`
}

// SkeletonConfigItem SkeletonConfig
type SkeletonConfigItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Config name
	Name string `json:"name"`
	// SkelName Config Skel name
	SkelName string `json:"skelName"`
	// Description Config description
	Description *string `json:"description"`
	// Pub Whether the item is public
	Pub bool `json:"pub"`
	// Lifecycle Config lifecycle
	Lifecycle string `json:"lifecycle"`
	// Fields Field list
	Fields []SkeletonField `json:"fields"`
}

func (v *SkeletonConfigItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	return nil
}

// SkeletonData SkeletonData
type SkeletonData struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Data name
	Name string `json:"name"`
	// SkelName Data Skel name
	SkelName string `json:"skelName"`
	// Description Data description
	Description *string `json:"description"`
	// Enum Whether it is Enum
	Enum bool `json:"enum"`
	// TypeParameters Type parameter list
	TypeParameters []string `json:"typeParameters"`
	// Fields Field list
	Fields []SkeletonField `json:"fields"`
	// EnumItems List of enumeration items
	EnumItems []SkeletonEnumItem `json:"enumItems"`
}

func (v *SkeletonData) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.TypeParameters, rpc.JoinPath(path, "TypeParameters")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.EnumItems, rpc.JoinPath(path, "EnumItems")); err != nil {
		return err
	}
	return nil
}

// SkeletonDomain Domain skeleton version
type SkeletonDomain struct {
	// Domain Domain name
	Domain string `json:"domain"`
	// SchemaHash DomainSchema hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary DomainSchema hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether multiple active versions exist
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether this is the primary version
	IsMain bool `json:"isMain"`
	// Total Total number of skeleton items
	Total int `json:"total"`
	// Actors Actor list
	Actors []SkeletonActorItem `json:"actors"`
	// Services Service list
	Services []SkeletonServiceItem `json:"services"`
	// Resources Resource list
	Resources []SkeletonResourceItem `json:"resources"`
	// Data Data list
	Data []SkeletonData `json:"data"`
	// Configs Config list
	Configs []SkeletonConfigItem `json:"configs"`
	// Webs Web list
	Webs []SkeletonWebItem `json:"webs"`
	// Tasks Task list
	Tasks []SkeletonTask `json:"tasks"`
	// Events Event list
	Events []SkeletonEventItem `json:"events"`
}

func (v *SkeletonDomain) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Actors, rpc.JoinPath(path, "Actors")); err != nil {
		return err
	}
	for i0 := range v.Actors {
		if err := (&v.Actors[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Actors"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Services, rpc.JoinPath(path, "Services")); err != nil {
		return err
	}
	for i0 := range v.Services {
		if err := (&v.Services[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Services"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Resources, rpc.JoinPath(path, "Resources")); err != nil {
		return err
	}
	for i0 := range v.Resources {
		if err := (&v.Resources[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Resources"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Data, rpc.JoinPath(path, "Data")); err != nil {
		return err
	}
	for i0 := range v.Data {
		if err := (&v.Data[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Data"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Configs, rpc.JoinPath(path, "Configs")); err != nil {
		return err
	}
	for i0 := range v.Configs {
		if err := (&v.Configs[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Configs"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Webs, rpc.JoinPath(path, "Webs")); err != nil {
		return err
	}
	for i0 := range v.Webs {
		if err := (&v.Webs[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Webs"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Tasks, rpc.JoinPath(path, "Tasks")); err != nil {
		return err
	}
	for i0 := range v.Tasks {
		if err := (&v.Tasks[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Tasks"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Events, rpc.JoinPath(path, "Events")); err != nil {
		return err
	}
	for i0 := range v.Events {
		if err := (&v.Events[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Events"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonEnumItem Skeleton enumeration items
type SkeletonEnumItem struct {
	// Name Enumeration item name
	Name string `json:"name"`
	// Description Enumeration item description
	Description *string `json:"description"`
}

// SkeletonEventItem Skeleton event
type SkeletonEventItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Event name
	Name string `json:"name"`
	// SkelName Event Skel name
	SkelName string `json:"skelName"`
	// Description Event description
	Description *string `json:"description"`
	// Pub Whether the item is public
	Pub bool `json:"pub"`
	// Fields Field list
	Fields []SkeletonField `json:"fields"`
}

func (v *SkeletonEventItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Fields, rpc.JoinPath(path, "Fields")); err != nil {
		return err
	}
	return nil
}

// SkeletonField Skeleton field
type SkeletonField struct {
	// Name Field name
	Name string `json:"name"`
	// Type Field type
	Type string `json:"type"`
	// Description Field description
	Description *string `json:"description"`
	// Example Field example
	Example *string `json:"example"`
}

// SkeletonMethod Skeleton method
type SkeletonMethod struct {
	// Name Method name
	Name string `json:"name"`
	// SkelName Method Skel name
	SkelName string `json:"skelName"`
	// Description Method description
	Description *string `json:"description"`
	// InputDescription Input description
	InputDescription *string `json:"inputDescription"`
	// OutputDescription Output description
	OutputDescription *string `json:"outputDescription"`
	// Example Input example
	Example *string `json:"example"`
	// AuthMode Authentication mode
	AuthMode string `json:"authMode"`
	// Require Permission requirements
	Require *SkeletonPermExpr `json:"require"`
	// OutputExample Output example
	OutputExample *string `json:"outputExample"`
	// Arguments Parameter list
	Arguments []SkeletonField `json:"arguments"`
	// ResultType Return type
	ResultType string `json:"resultType"`
}

func (v *SkeletonMethod) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Require != nil {
		if err := v.Require.Validate(rpc.JoinPath(path, "Require")); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
}

// SkeletonPermCheck Skeleton permission verification call
type SkeletonPermCheck struct {
	// ResourceSkelName Resource Skel name
	ResourceSkelName string `json:"resourceSkelName"`
	// ActionName Action name
	ActionName string `json:"actionName"`
	// CheckName Check name
	CheckName string `json:"checkName"`
	// ServiceSkelName Check Service Skel name
	ServiceSkelName string `json:"serviceSkelName"`
	// MethodSkelName Check Method Skel name
	MethodSkelName string `json:"methodSkelName"`
	// Arguments Parameter list
	Arguments []SkeletonPermCheckArgument `json:"arguments"`
}

func (v *SkeletonPermCheck) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
}

// SkeletonPermCheckArgument Skeleton permission verification parameters
type SkeletonPermCheckArgument struct {
	// Name Parameter name
	Name string `json:"name"`
	// JsonPath Parameter JSON path
	JsonPath string `json:"jsonPath"`
	// Type Parameter type
	Type string `json:"type"`
}

// SkeletonPermExpr Skeleton permission expression
type SkeletonPermExpr struct {
	// Mode Permission expression pattern
	Mode string `json:"mode"`
	// Code Permission code
	Code *string `json:"code"`
	// Check Permission verification call
	Check *SkeletonPermCheck `json:"check"`
	// Children Subexpression
	Children []SkeletonPermExpr `json:"children"`
}

func (v *SkeletonPermExpr) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Check != nil {
		if err := v.Check.Validate(rpc.JoinPath(path, "Check")); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Children, rpc.JoinPath(path, "Children")); err != nil {
		return err
	}
	for i0 := range v.Children {
		if err := (&v.Children[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Children"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonResourceAction SkeletonResource Action
type SkeletonResourceAction struct {
	// Name Action name
	Name string `json:"name"`
	// PermissionCode Permission code
	PermissionCode string `json:"permissionCode"`
	// Description Action description
	Description *string `json:"description"`
	// Checks Check list
	Checks []SkeletonResourceCheck `json:"checks"`
}

func (v *SkeletonResourceAction) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Checks, rpc.JoinPath(path, "Checks")); err != nil {
		return err
	}
	for i0 := range v.Checks {
		if err := (&v.Checks[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Checks"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonResourceCheck SkeletonResource Check
type SkeletonResourceCheck struct {
	// Name Check name
	Name string `json:"name"`
	// MethodName Check method name
	MethodName string `json:"methodName"`
	// MethodSkelName Check method Skel name
	MethodSkelName string `json:"methodSkelName"`
	// Arguments Parameter list
	Arguments []SkeletonField `json:"arguments"`
}

func (v *SkeletonResourceCheck) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
}

// SkeletonResourceItem Skeleton Resource item
type SkeletonResourceItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Resource name
	Name string `json:"name"`
	// SkelName Resource Skel name
	SkelName string `json:"skelName"`
	// Description Resource description
	Description *string `json:"description"`
	// Checks Resource level Check list
	Checks []SkeletonResourceCheck `json:"checks"`
	// Actions Action list
	Actions []SkeletonResourceAction `json:"actions"`
	// CheckService Check service
	CheckService *SkeletonServiceItem `json:"checkService"`
}

func (v *SkeletonResourceItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Checks, rpc.JoinPath(path, "Checks")); err != nil {
		return err
	}
	for i0 := range v.Checks {
		if err := (&v.Checks[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Checks"), i0)); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Actions, rpc.JoinPath(path, "Actions")); err != nil {
		return err
	}
	for i0 := range v.Actions {
		if err := (&v.Actions[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Actions"), i0)); err != nil {
			return err
		}
	}
	if v.CheckService != nil {
		if err := v.CheckService.Validate(rpc.JoinPath(path, "CheckService")); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonServiceItem Skeleton service items
type SkeletonServiceItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Service name
	Name string `json:"name"`
	// SkelName Service Skel name
	SkelName string `json:"skelName"`
	// Description Service Description
	Description *string `json:"description"`
	// Pub Whether the item is public
	Pub bool `json:"pub"`
	// AuthMode Authentication mode
	AuthMode string `json:"authMode"`
	// Require Permission requirements
	Require *SkeletonPermExpr `json:"require"`
	// Actors Accessible Actor List
	Actors []SkeletonActorRef `json:"actors"`
	// Methods Method list
	Methods []SkeletonMethod `json:"methods"`
}

func (v *SkeletonServiceItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if v.Require != nil {
		if err := v.Require.Validate(rpc.JoinPath(path, "Require")); err != nil {
			return err
		}
	}
	if err := rpc.CheckValueNotNil(v.Actors, rpc.JoinPath(path, "Actors")); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Methods, rpc.JoinPath(path, "Methods")); err != nil {
		return err
	}
	for i0 := range v.Methods {
		if err := (&v.Methods[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Methods"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonTask Skeleton task
type SkeletonTask struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Task name
	Name string `json:"name"`
	// SkelName Task Skel name
	SkelName string `json:"skelName"`
	// Description Task description
	Description *string `json:"description"`
	// Triggers Trigger list
	Triggers []SkeletonTrigger `json:"triggers"`
}

func (v *SkeletonTask) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Triggers, rpc.JoinPath(path, "Triggers")); err != nil {
		return err
	}
	for i0 := range v.Triggers {
		if err := (&v.Triggers[i0]).Validate(rpc.JoinIndex(rpc.JoinPath(path, "Triggers"), i0)); err != nil {
			return err
		}
	}
	return nil
}

// SkeletonTrigger Skeleton task trigger
type SkeletonTrigger struct {
	// Name Trigger name
	Name string `json:"name"`
	// SkelName Trigger Skel name
	SkelName string `json:"skelName"`
	// Description Trigger description
	Description *string `json:"description"`
	// InputDescription Input description
	InputDescription *string `json:"inputDescription"`
	// Example Input example
	Example *string `json:"example"`
	// Arguments Parameter list
	Arguments []SkeletonField `json:"arguments"`
}

func (v *SkeletonTrigger) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
}

// SkeletonWebItem Skeleton web page
type SkeletonWebItem struct {
	// Domain Domain
	Domain string `json:"domain"`
	// SchemaHash Skeleton item hash
	SchemaHash string `json:"schemaHash"`
	// MainSchemaHash Primary skeleton item hash
	MainSchemaHash string `json:"mainSchemaHash"`
	// IsMultiVersion Whether there are multiple valid versions of the skeleton item
	IsMultiVersion bool `json:"isMultiVersion"`
	// IsMain Whether it is the main version of the skeleton item
	IsMain bool `json:"isMain"`
	// DomainSchemaHash Owning DomainSchema hash
	DomainSchemaHash string `json:"domainSchemaHash"`
	// Name Web page name
	Name string `json:"name"`
	// SkelName Web Skel name
	SkelName string `json:"skelName"`
	// Description Web page description
	Description *string `json:"description"`
	// Actors Accessible Actor List
	Actors []SkeletonActorRef `json:"actors"`
}

func (v *SkeletonWebItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Actors, rpc.JoinPath(path, "Actors")); err != nil {
		return err
	}
	return nil
}

// TaskDebugDefaultLaunchRequest Task Debug initiates a request by default
type TaskDebugDefaultLaunchRequest struct {
	// TraceId Trace ID
	TraceId string `json:"traceId"`
	// SpanId Span ID
	SpanId string `json:"spanId"`
	// ArgumentsJson Default task parameters JSON
	ArgumentsJson skel.JSON `json:"argumentsJson"`
}

// TaskDebugLaunchRequest Task Debug initiates a request
type TaskDebugLaunchRequest struct {
	// TaskSkelName Task Skel name
	TaskSkelName string `json:"taskSkelName"`
	// SchemaHash Task schema hash
	SchemaHash string `json:"schemaHash"`
	// TriggerSkelName Trigger Skel name
	TriggerSkelName string `json:"triggerSkelName"`
	// ArgumentsJson Task parameters JSON
	ArgumentsJson skel.JSON `json:"argumentsJson"`
	// TraceId Trace ID
	TraceId *string `json:"traceId"`
	// SpanId Span ID
	SpanId *string `json:"spanId"`
}

// TaskDebugTaskItem Task called by Task Debug
type TaskDebugTaskItem struct {
	// Name Task name
	Name string `json:"name"`
	// TaskSkelName Task Skel name
	TaskSkelName string `json:"taskSkelName"`
	// SchemaHash Task schema hash
	SchemaHash string `json:"schemaHash"`
	// Description Task description
	Description *string `json:"description"`
}

// TaskDebugTriggerItem Trigger called by Task Debug
type TaskDebugTriggerItem struct {
	// Name Trigger name
	Name string `json:"name"`
	// SkelName Trigger Skel name
	SkelName string `json:"skelName"`
	// Description Trigger description
	Description *string `json:"description"`
	// InputDescription Input description
	InputDescription *string `json:"inputDescription"`
	// Example Input example
	Example *string `json:"example"`
	// Arguments Parameter list
	Arguments []SkeletonField `json:"arguments"`
}

func (v *TaskDebugTriggerItem) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
	if err := rpc.CheckValueNotNil(v.Arguments, rpc.JoinPath(path, "Arguments")); err != nil {
		return err
	}
	return nil
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
	// Endpoint Web proxy access address
	Endpoint string `json:"endpoint"`
}
