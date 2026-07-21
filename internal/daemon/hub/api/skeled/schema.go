package skeled

import "go.yorun.ai/vine/internal/core/skel"

func init() {
	skel.RegisterDomainSchema(_DomainSchema)
}

var _DomainSchema = &skel.DomainSchema{
	Domain:      "vine.hub",
	Description: "Internal API for vine framework",
	Hash:        "1e590da9",
	Full:        true,
	Generated: &skel.GeneratedInfo{
		CompilerVersion: "v0.9.0",
	},
	Enums: []*skel.EnumSchema{
		{
			Name:        "PortalCorsMode",
			SkelName:    "vine.hub.PortalCorsMode",
			Description: "Portal site CORS mode",
			Hash:        "198614d9",
			Items: []*skel.EnumItemSchema{
				{
					Name:        "DISABLED",
					Description: "Disable CORS",
				},
				{
					Name:        "SAME_DOMAIN",
					Description: "Allow origins in the same domain as the entry rule",
				},
				{
					Name:        "STRICT",
					Description: "Only allow Origins in the configuration list",
				},
			},
		},
		{
			Name:        "PortalSiteType",
			SkelName:    "vine.hub.PortalSiteType",
			Description: "Portal target site type",
			Hash:        "24616fb2",
			Items: []*skel.EnumItemSchema{
				{
					Name:        "RPCGW",
					Description: "Rpc gateway",
				},
				{
					Name:        "WEBGW",
					Description: "Web gateway",
				},
			},
		},
	},
	Data: []*skel.DataSchema{
		{
			Name:        "AppConfigCreation",
			SkelName:    "vine.hub.AppConfigCreation",
			Description: "Configuration creation parameters",
			Hash:        "a8196614",
			Members: []*skel.MemberSchema{
				{
					Name:        "skelName",
					Description: "Configuration Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "value",
					Description: "Configuration JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "AppConfigItem",
			SkelName:    "vine.hub.AppConfigItem",
			Description: "Configuration items",
			Hash:        "27bd3741",
			Members: []*skel.MemberSchema{
				{
					Name:        "id",
					Description: "Configuration ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "key",
					Description: "Configuration key",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "status",
					Description: "Configuration status",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "lifecycle",
					Description: "Configuration lifecycle",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "value",
					Description: "Configuration JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schema",
					Description: "Configuration schema",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "AppConfigSchema",
						SkelName: "vine.hub.AppConfigSchema",
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "AppConfigSchema",
			SkelName:    "vine.hub.AppConfigSchema",
			Description: "Configuration schema items",
			Hash:        "02453d65",
			Members: []*skel.MemberSchema{
				{
					Name:        "skelName",
					Description: "Configuration Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Configuration name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Configuration description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "lifecycle",
					Description: "Configuration lifecycle",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "fields",
					Description: "Configuration field list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "AppConfigSchemaField",
							SkelName: "vine.hub.AppConfigSchemaField",
						},
					},
				},
			},
		},
		{
			Name:        "AppConfigSchemaEnumItem",
			SkelName:    "vine.hub.AppConfigSchemaEnumItem",
			Description: "Configuration schema enumeration options",
			Hash:        "e248cb0d",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Enum option name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Enumeration options description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "AppConfigSchemaField",
			SkelName:    "vine.hub.AppConfigSchemaField",
			Description: "Configuration schema fields",
			Hash:        "94266eea",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Field name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "type",
					Description: "Field type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Field description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "enumItems",
					Description: "Enumeration options list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "AppConfigSchemaEnumItem",
							SkelName: "vine.hub.AppConfigSchemaEnumItem",
						},
					},
				},
			},
		},
		{
			Name:        "AppConfigUpdate",
			SkelName:    "vine.hub.AppConfigUpdate",
			Description: "Configuration update parameters",
			Hash:        "278f2c00",
			Members: []*skel.MemberSchema{
				{
					Name:        "value",
					Description: "Configuration JSON",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "AppRegistration",
			SkelName:    "vine.hub.AppRegistration",
			Description: "Link application instance information registered with Hub",
			Hash:        "85ea6e7f",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Application name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "instanceId",
					Description: "Application instance ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarUuid,
					},
				},
				{
					Name:        "version",
					Description: "Application version",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "endpoint",
					Description: "Application access address",
					Example:     "\"http://10.1.2.3:23001\"",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "serviceHandlers",
					Description: "List of Rpc service processing capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceHandlerRegistration",
							SkelName: "vine.hub.ServiceHandlerRegistration",
						},
					},
				},
				{
					Name:        "webHandlers",
					Description: "List of web processing capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "WebHandlerRegistration",
							SkelName: "vine.hub.WebHandlerRegistration",
						},
					},
				},
				{
					Name:        "eventListeners",
					Description: "List of event listening capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "EventListenerRegistration",
							SkelName: "vine.hub.EventListenerRegistration",
						},
					},
				},
				{
					Name:        "taskRunners",
					Description: "List of task execution capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskRunnerRegistration",
							SkelName: "vine.hub.TaskRunnerRegistration",
						},
					},
				},
				{
					Name:        "domainSchemas",
					Description: "List of all DomainSchemas registered by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarJson,
						},
					},
				},
			},
		},
		{
			Name:        "AppStatus",
			SkelName:    "vine.hub.AppStatus",
			Description: "Application instance status information, used for heartbeat refresh",
			Hash:        "70b00a09",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Application name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "instanceId",
					Description: "Application instance ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarUuid,
					},
				},
			},
		},
		{
			Name:        "AppStatusView",
			SkelName:    "vine.hub.AppStatusView",
			Description: "Application instance status view for Dashboard display",
			Hash:        "e77c653b",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Application name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "instanceId",
					Description: "Application instance ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "version",
					Description: "Application version",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "endpoint",
					Description: "Application access address",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "serviceHandlers",
					Description: "List of Rpc service processing capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceHandlerRegistration",
							SkelName: "vine.hub.ServiceHandlerRegistration",
						},
					},
				},
				{
					Name:        "webHandlers",
					Description: "List of web processing capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "WebHandlerRegistration",
							SkelName: "vine.hub.WebHandlerRegistration",
						},
					},
				},
				{
					Name:        "eventListeners",
					Description: "List of event listening capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "EventListenerRegistration",
							SkelName: "vine.hub.EventListenerRegistration",
						},
					},
				},
				{
					Name:        "taskRunners",
					Description: "List of task execution capabilities provided by the application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskRunnerRegistration",
							SkelName: "vine.hub.TaskRunnerRegistration",
						},
					},
				},
			},
		},
		{
			Name:        "EventDebugDefaultEmitRequest",
			SkelName:    "vine.hub.EventDebugDefaultEmitRequest",
			Description: "Default Event Debug send request",
			Hash:        "50c73825",
			Members: []*skel.MemberSchema{
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "eventJson",
					Description: "Default event JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "EventDebugEmitRequest",
			SkelName:    "vine.hub.EventDebugEmitRequest",
			Description: "Event Debug send request",
			Hash:        "8451d651",
			Members: []*skel.MemberSchema{
				{
					Name:        "eventSkelName",
					Description: "Event Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Event schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "eventJson",
					Description: "Event JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "EventDebugEventItem",
			SkelName:    "vine.hub.EventDebugEventItem",
			Description: "Event called by Event Debug",
			Hash:        "e89b5f2b",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Event name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "eventSkelName",
					Description: "Event Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Event schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Event description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "fields",
					Description: "Field list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "EventListenerRegistration",
			SkelName:    "vine.hub.EventListenerRegistration",
			Description: "Event listening capability registration information provided by the application",
			Hash:        "7ba0316b",
			Members: []*skel.MemberSchema{
				{
					Name:        "eventSkelName",
					Description: "Event Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Event schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "timeoutMs",
					Description: "Execution timeout, in milliseconds",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "concurrency",
					Description: "Maximum concurrency",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "noRetry",
					Description: "Whether to disallow retrying after failure",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "Info",
			SkelName:    "vine.hub.Info",
			Description: "Hub information",
			Hash:        "57cf2dfa",
			Members: []*skel.MemberSchema{
				{
					Name:        "apiPort",
					Description: "API service port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "redisPort",
					Description: "Redis service port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "natsPort",
					Description: "NATS service port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "mqEndpoint",
					Description: "Standalone MQ service address",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalCert",
			SkelName:    "vine.hub.PortalCert",
			Description: "Portal site certificate",
			Hash:        "b08ca38a",
			Members: []*skel.MemberSchema{
				{
					Name:        "id",
					Description: "Certificate ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "name",
					Description: "Certificate name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "issuer",
					Description: "Certificate issuer",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "domains",
					Description: "Certificate domain name",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
				{
					Name:        "publicKeyBase64",
					Description: "Certificate Base64",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "privateKeyConfigured",
					Description: "Whether the private key has been configured",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "validFrom",
					Description: "Validity start time",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarTimestamp,
					},
				},
				{
					Name:        "validTo",
					Description: "Validity end time",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarTimestamp,
					},
				},
			},
		},
		{
			Name:        "PortalCertCreation",
			SkelName:    "vine.hub.PortalCertCreation",
			Description: "Portal site certificate creation parameters",
			Hash:        "d721b615",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Certificate name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "publicKeyBase64",
					Description: "Certificate Base64",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "privateKeyBase64",
					Description: "Private key Base64",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalCertUpdate",
			SkelName:    "vine.hub.PortalCertUpdate",
			Description: "Portal site certificate update parameters",
			Hash:        "0946dd75",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Certificate name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "publicKeyBase64",
					Description: "Certificate Base64",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "privateKeyBase64",
					Description: "Private key Base64",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "PortalCors",
			SkelName:    "vine.hub.PortalCors",
			Description: "Portal site CORS configuration",
			Hash:        "ac6f9690",
			Members: []*skel.MemberSchema{
				{
					Name:        "mode",
					Description: "CORS mode: DISABLED/SAME_DOMAIN/STRICT",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindEnum,
						Name:     "PortalCorsMode",
						SkelName: "vine.hub.PortalCorsMode",
					},
				},
				{
					Name:        "allowedOrigins",
					Description: "List of origins allowed in strict mode",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
			},
		},
		{
			Name:        "PortalDashboardAccess",
			SkelName:    "vine.hub.PortalDashboardAccess",
			Description: "Hub Dashboard access entry",
			Hash:        "62b35e0c",
			Members: []*skel.MemberSchema{
				{
					Name:        "scheme",
					Description: "Entry protocol",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "port",
					Description: "Entry port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "pathPrefix",
					Description: "Match path prefix",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "canUpdate",
					Description: "Whether to allow modification of Dashboard access entry",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "PortalEntry",
			SkelName:    "vine.hub.PortalEntry",
			Description: "Portal access entry",
			Hash:        "6e8d32a1",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Entry name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "scheme",
					Description: "Entry protocol",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "port",
					Description: "Entry port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "rules",
					Description: "Entry rule list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalEntryRule",
							SkelName: "vine.hub.PortalEntryRule",
						},
					},
				},
			},
		},
		{
			Name:        "PortalEntryAccessUpdate",
			SkelName:    "vine.hub.PortalEntryAccessUpdate",
			Description: "Portal access entry configuration update parameters",
			Hash:        "edbbc926",
			Members: []*skel.MemberSchema{
				{
					Name:        "scheme",
					Description: "Entry protocol",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "port",
					Description: "Entry port",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
			},
		},
		{
			Name:        "PortalEntryRule",
			SkelName:    "vine.hub.PortalEntryRule",
			Description: "Portal access entry rules",
			Hash:        "19c6fb6d",
			Members: []*skel.MemberSchema{
				{
					Name:        "rule",
					Description: "Entry rules",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalRule",
						SkelName: "vine.hub.PortalRule",
					},
				},
				{
					Name:        "site",
					Description: "Target site",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalSite",
						SkelName: "vine.hub.PortalSite",
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "PortalRule",
			SkelName:    "vine.hub.PortalRule",
			Description: "Portal entry rules",
			Hash:        "55af1f7a",
			Members: []*skel.MemberSchema{
				{
					Name:        "id",
					Description: "Rule ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "name",
					Description: "Rule name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "scheme",
					Description: "Matching protocol",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "port",
					Description: "Match port, 0 means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "pathPrefix",
					Description: "Match path prefix, empty string means match all paths",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "targetType",
					Description: "Target type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "siteName",
					Description: "Site name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "redirectionPattern",
					Description: "Redirect Pattern",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalRuleCreation",
			SkelName:    "vine.hub.PortalRuleCreation",
			Description: "Portal entry rule creation parameters",
			Hash:        "2f2a64c3",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Rule name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "scheme",
					Description: "Matching protocol",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "port",
					Description: "Match port, 0 means no restriction",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "pathPrefix",
					Description: "Match path prefix, empty string means match all paths",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "targetType",
					Description: "Target type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "siteName",
					Description: "Site name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "redirectionPattern",
					Description: "Redirect Pattern",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalRuleUpdate",
			SkelName:    "vine.hub.PortalRuleUpdate",
			Description: "Portal entry rule update parameters",
			Hash:        "e0f8c1a3",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Rule name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "scheme",
					Description: "Matching protocol",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "host",
					Description: "Match Host, empty string means no restriction",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "port",
					Description: "Match port, 0 means no restriction",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarInt,
						Nullable: true,
					},
				},
				{
					Name:        "pathPrefix",
					Description: "Match path prefix, empty string means match all paths",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "targetType",
					Description: "Target type",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "siteName",
					Description: "Site name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "redirectionPattern",
					Description: "Redirect Pattern",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "PortalSite",
			SkelName:    "vine.hub.PortalSite",
			Description: "Portal target site",
			Hash:        "825470d7",
			Members: []*skel.MemberSchema{
				{
					Name:        "id",
					Description: "Target site id",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "name",
					Description: "Target site name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "type",
					Description: "Target site type",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindEnum,
						Name:     "PortalSiteType",
						SkelName: "vine.hub.PortalSiteType",
					},
				},
				{
					Name:        "actorSkelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorVia",
					Description: "Actor access method",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "rpcgwServices",
					Description: "Rpc gateway service Skel name list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
				{
					Name:        "cors",
					Description: "CORS configuration",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCors",
						SkelName: "vine.hub.PortalCors",
						Nullable: true,
					},
				},
				{
					Name:        "webName",
					Description: "Web name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalSiteActorOption",
			SkelName:    "vine.hub.PortalSiteActorOption",
			Description: "Portal target site Actor options",
			Hash:        "085e02ca",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Actor name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorVias",
					Description: "Actor access method list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
			},
		},
		{
			Name:        "PortalSiteCreation",
			SkelName:    "vine.hub.PortalSiteCreation",
			Description: "Portal target site creation parameters",
			Hash:        "c49a5c08",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Target site name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "type",
					Description: "Target site type",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindEnum,
						Name:     "PortalSiteType",
						SkelName: "vine.hub.PortalSiteType",
					},
				},
				{
					Name:        "actorSkelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorVia",
					Description: "Actor access method",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "cors",
					Description: "CORS configuration",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCors",
						SkelName: "vine.hub.PortalCors",
						Nullable: true,
					},
				},
				{
					Name:        "webName",
					Description: "Web name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "PortalSiteOptions",
			SkelName:    "vine.hub.PortalSiteOptions",
			Description: "Portal target site form options",
			Hash:        "0683539d",
			Members: []*skel.MemberSchema{
				{
					Name:        "actors",
					Description: "Actor options",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalSiteActorOption",
							SkelName: "vine.hub.PortalSiteActorOption",
						},
					},
				},
				{
					Name:        "services",
					Description: "Rpc service options",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalSiteServiceOption",
							SkelName: "vine.hub.PortalSiteServiceOption",
						},
					},
				},
				{
					Name:        "webs",
					Description: "Web options",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalSiteWebOption",
							SkelName: "vine.hub.PortalSiteWebOption",
						},
					},
				},
			},
		},
		{
			Name:        "PortalSiteServiceOption",
			SkelName:    "vine.hub.PortalSiteServiceOption",
			Description: "Portal target site service options",
			Hash:        "5bf45d62",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Service name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorSkelNames",
					Description: "Actor Skel name list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
			},
		},
		{
			Name:        "PortalSiteUpdate",
			SkelName:    "vine.hub.PortalSiteUpdate",
			Description: "Portal target site update parameters",
			Hash:        "09d7113c",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Target site name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "type",
					Description: "Target site type",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindEnum,
						Name:     "PortalSiteType",
						SkelName: "vine.hub.PortalSiteType",
						Nullable: true,
					},
				},
				{
					Name:        "actorSkelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actorVia",
					Description: "Actor access method",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "cors",
					Description: "CORS configuration",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCors",
						SkelName: "vine.hub.PortalCors",
						Nullable: true,
					},
				},
				{
					Name:        "webName",
					Description: "Web name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "PortalSiteWebOption",
			SkelName:    "vine.hub.PortalSiteWebOption",
			Description: "Portal target site web options",
			Hash:        "79a9441e",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Web name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Web Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorSkelNames",
					Description: "Actor Skel name list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
			},
		},
		{
			Name:        "SeedEntityDiff",
			SkelName:    "vine.hub.SeedEntityDiff",
			Description: "Seed entity differences",
			Hash:        "1b6c6e4b",
			Members: []*skel.MemberSchema{
				{
					Name:        "kind",
					Description: "Entity type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Entity name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "exists",
					Description: "Whether the entity currently exists",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "fields",
					Description: "Field differences",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SeedFieldDiff",
							SkelName: "vine.hub.SeedFieldDiff",
						},
					},
				},
			},
		},
		{
			Name:        "SeedFieldDiff",
			SkelName:    "vine.hub.SeedFieldDiff",
			Description: "Seed field differences",
			Hash:        "479e6382",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Field name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "currentValue",
					Description: "Current value",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "seedValue",
					Description: "Seed value",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "changed",
					Description: "Whether the values differ",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "SeedItemSelection",
			SkelName:    "vine.hub.SeedItemSelection",
			Description: "Seed entity selection",
			Hash:        "5b8155e4",
			Members: []*skel.MemberSchema{
				{
					Name:        "kind",
					Description: "Entity type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Entity name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "SeedPreview",
			SkelName:    "vine.hub.SeedPreview",
			Description: "Seed preview",
			Hash:        "dcd8cb36",
			Members: []*skel.MemberSchema{
				{
					Name:        "items",
					Description: "Entity differences",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SeedEntityDiff",
							SkelName: "vine.hub.SeedEntityDiff",
						},
					},
				},
			},
		},
		{
			Name:        "ServiceDebugActorItem",
			SkelName:    "vine.hub.ServiceDebugActorItem",
			Description: "Service Debug Actor options",
			Hash:        "181e40fd",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Actor name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "infoSkelName",
					Description: "Actor Info Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actorInfoJson",
					Description: "Default Actor Info JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugAppInstance",
			SkelName:    "vine.hub.ServiceDebugAppInstance",
			Description: "Application instance called by Service Debug",
			Hash:        "2d06f9ba",
			Members: []*skel.MemberSchema{
				{
					Name:        "appName",
					Description: "Application name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appInstanceId",
					Description: "Application instance ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appVersion",
					Description: "Application version",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "endpoint",
					Description: "Application access address",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugDefaultInvokeRequest",
			SkelName:    "vine.hub.ServiceDebugDefaultInvokeRequest",
			Description: "Service Debug default call request",
			Hash:        "8832afd1",
			Members: []*skel.MemberSchema{
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actors",
					Description: "Actor options",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceDebugActorItem",
							SkelName: "vine.hub.ServiceDebugActorItem",
						},
					},
				},
				{
					Name:        "actorSkelName",
					Description: "Default Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actorInfoJson",
					Description: "Default Actor Info JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
				{
					Name:        "paramsJson",
					Description: "Default request parameters JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugInvokeRequest",
			SkelName:    "vine.hub.ServiceDebugInvokeRequest",
			Description: "Service Debug call request",
			Hash:        "a4f612c6",
			Members: []*skel.MemberSchema{
				{
					Name:        "appName",
					Description: "Application name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "appInstanceId",
					Description: "Application instance ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "serviceSkelName",
					Description: "Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Service schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "methodSkelName",
					Description: "Method Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "paramsJson",
					Description: "Request parameters JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
				{
					Name:        "timeoutSeconds",
					Description: "Call timeout, in seconds",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actorSkelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actorInfoJson",
					Description: "Actor Info JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugInvokeResponse",
			SkelName:    "vine.hub.ServiceDebugInvokeResponse",
			Description: "Service Debug call response",
			Hash:        "6dedee5a",
			Members: []*skel.MemberSchema{
				{
					Name:        "httpStatus",
					Description: "HTTP status code",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "rpcStatus",
					Description: "Rpc status code",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "headersJson",
					Description: "Response header JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
				{
					Name:        "bodyJson",
					Description: "Response body JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugMethodItem",
			SkelName:    "vine.hub.ServiceDebugMethodItem",
			Description: "Method called by Service Debug",
			Hash:        "013da93b",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Method name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Method Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Method description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "inputDescription",
					Description: "Input description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "outputDescription",
					Description: "Output description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "example",
					Description: "Input example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "outputExample",
					Description: "Output example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
				{
					Name:        "resultType",
					Description: "Return type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugServiceItem",
			SkelName:    "vine.hub.ServiceDebugServiceItem",
			Description: "Service called by Service Debug",
			Hash:        "dca3c721",
			Members: []*skel.MemberSchema{
				{
					Name:        "serviceSkelName",
					Description: "Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Service schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "ServiceHandlerRegistration",
			SkelName:    "vine.hub.ServiceHandlerRegistration",
			Description: "Rpc service processing capability registration information provided by the application",
			Hash:        "2ab4622c",
			Members: []*skel.MemberSchema{
				{
					Name:        "serviceSkelName",
					Description: "Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Service schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "endpoint",
					Description: "Service agent access address",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "SkeletonActorItem",
			SkelName:    "vine.hub.SkeletonActorItem",
			Description: "SkeletonActor",
			Hash:        "2f7c7bec",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Actor name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Actor description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actorVias",
					Description: "Actor access method list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
				{
					Name:        "authEnabled",
					Description: "Whether to enable authentication",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "credential",
					Description: "Authentication credentials",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonData",
						SkelName: "vine.hub.SkeletonData",
						Nullable: true,
					},
				},
				{
					Name:        "info",
					Description: "Authentication information",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonData",
						SkelName: "vine.hub.SkeletonData",
						Nullable: true,
					},
				},
				{
					Name:        "authService",
					Description: "Authentication services",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonServiceItem",
						SkelName: "vine.hub.SkeletonServiceItem",
						Nullable: true,
					},
				},
				{
					Name:        "permEnabled",
					Description: "Whether to enable permissions",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "permService",
					Description: "Permission service",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonServiceItem",
						SkelName: "vine.hub.SkeletonServiceItem",
						Nullable: true,
					},
				},
				{
					Name:        "permMethod",
					Description: "Permission method",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonMethod",
						SkelName: "vine.hub.SkeletonMethod",
						Nullable: true,
					},
				},
				{
					Name:        "services",
					Description: "Accessible Service List",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonServiceItem",
							SkelName: "vine.hub.SkeletonServiceItem",
						},
					},
				},
				{
					Name:        "webs",
					Description: "Accessible web list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonWebItem",
							SkelName: "vine.hub.SkeletonWebItem",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonActorRef",
			SkelName:    "vine.hub.SkeletonActorRef",
			Description: "Skeleton Actor Reference",
			Hash:        "cf797fff",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Actor name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Actor Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "via",
					Description: "Access method",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "SkeletonConfigItem",
			SkelName:    "vine.hub.SkeletonConfigItem",
			Description: "SkeletonConfig",
			Hash:        "28cedd90",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Config name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Config Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Config description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "pub",
					Description: "Whether the item is public",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "lifecycle",
					Description: "Config lifecycle",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "fields",
					Description: "Field list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonData",
			SkelName:    "vine.hub.SkeletonData",
			Description: "SkeletonData",
			Hash:        "011bd126",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Data name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Data Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Data description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "enum",
					Description: "Whether it is Enum",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "typeParameters",
					Description: "Type parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:   skel.TypeKindScalar,
							Scalar: skel.ScalarString,
						},
					},
				},
				{
					Name:        "fields",
					Description: "Field list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
				{
					Name:        "enumItems",
					Description: "List of enumeration items",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonEnumItem",
							SkelName: "vine.hub.SkeletonEnumItem",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonDomain",
			SkelName:    "vine.hub.SkeletonDomain",
			Description: "Domain skeleton version",
			Hash:        "3aa935dc",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether multiple active versions exist",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether this is the primary version",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "total",
					Description: "Total number of skeleton items",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "actors",
					Description: "Actor list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonActorItem",
							SkelName: "vine.hub.SkeletonActorItem",
						},
					},
				},
				{
					Name:        "services",
					Description: "Service list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonServiceItem",
							SkelName: "vine.hub.SkeletonServiceItem",
						},
					},
				},
				{
					Name:        "resources",
					Description: "Resource list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonResourceItem",
							SkelName: "vine.hub.SkeletonResourceItem",
						},
					},
				},
				{
					Name:        "data",
					Description: "Data list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonData",
							SkelName: "vine.hub.SkeletonData",
						},
					},
				},
				{
					Name:        "configs",
					Description: "Config list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonConfigItem",
							SkelName: "vine.hub.SkeletonConfigItem",
						},
					},
				},
				{
					Name:        "webs",
					Description: "Web list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonWebItem",
							SkelName: "vine.hub.SkeletonWebItem",
						},
					},
				},
				{
					Name:        "tasks",
					Description: "Task list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonTask",
							SkelName: "vine.hub.SkeletonTask",
						},
					},
				},
				{
					Name:        "events",
					Description: "Event list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonEventItem",
							SkelName: "vine.hub.SkeletonEventItem",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonEnumItem",
			SkelName:    "vine.hub.SkeletonEnumItem",
			Description: "Skeleton enumeration items",
			Hash:        "46850c7c",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Enumeration item name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Enumeration item description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "SkeletonEventItem",
			SkelName:    "vine.hub.SkeletonEventItem",
			Description: "Skeleton event",
			Hash:        "0516d997",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Event name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Event Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Event description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "pub",
					Description: "Whether the item is public",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "fields",
					Description: "Field list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonField",
			SkelName:    "vine.hub.SkeletonField",
			Description: "Skeleton field",
			Hash:        "fa5a1007",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Field name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "type",
					Description: "Field type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Field description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "example",
					Description: "Field example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "SkeletonMethod",
			SkelName:    "vine.hub.SkeletonMethod",
			Description: "Skeleton method",
			Hash:        "3fa42b8d",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Method name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Method Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Method description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "inputDescription",
					Description: "Input description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "outputDescription",
					Description: "Output description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "example",
					Description: "Input example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "authMode",
					Description: "Authentication mode",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "require",
					Description: "Permission requirements",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonPermExpr",
						SkelName: "vine.hub.SkeletonPermExpr",
						Nullable: true,
					},
				},
				{
					Name:        "outputExample",
					Description: "Output example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
				{
					Name:        "resultType",
					Description: "Return type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "SkeletonPermCheck",
			SkelName:    "vine.hub.SkeletonPermCheck",
			Description: "Skeleton permission verification call",
			Hash:        "88cae7f5",
			Members: []*skel.MemberSchema{
				{
					Name:        "resourceSkelName",
					Description: "Resource Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "actionName",
					Description: "Action name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "checkName",
					Description: "Check name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "serviceSkelName",
					Description: "Check Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "methodSkelName",
					Description: "Check Method Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonPermCheckArgument",
							SkelName: "vine.hub.SkeletonPermCheckArgument",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonPermCheckArgument",
			SkelName:    "vine.hub.SkeletonPermCheckArgument",
			Description: "Skeleton permission verification parameters",
			Hash:        "367ecf9a",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Parameter name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "jsonPath",
					Description: "Parameter JSON path",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "type",
					Description: "Parameter type",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "SkeletonPermExpr",
			SkelName:    "vine.hub.SkeletonPermExpr",
			Description: "Skeleton permission expression",
			Hash:        "cf0aac0e",
			Members: []*skel.MemberSchema{
				{
					Name:        "mode",
					Description: "Permission expression pattern",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "code",
					Description: "Permission code",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "check",
					Description: "Permission verification call",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonPermCheck",
						SkelName: "vine.hub.SkeletonPermCheck",
						Nullable: true,
					},
				},
				{
					Name:        "children",
					Description: "Subexpression",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonPermExpr",
							SkelName: "vine.hub.SkeletonPermExpr",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonResourceAction",
			SkelName:    "vine.hub.SkeletonResourceAction",
			Description: "SkeletonResource Action",
			Hash:        "cb8b88ba",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Action name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "permissionCode",
					Description: "Permission code",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Action description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "checks",
					Description: "Check list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonResourceCheck",
							SkelName: "vine.hub.SkeletonResourceCheck",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonResourceCheck",
			SkelName:    "vine.hub.SkeletonResourceCheck",
			Description: "SkeletonResource Check",
			Hash:        "3401669b",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Check name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "methodName",
					Description: "Check method name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "methodSkelName",
					Description: "Check method Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonResourceItem",
			SkelName:    "vine.hub.SkeletonResourceItem",
			Description: "Skeleton Resource item",
			Hash:        "5dd2767c",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Resource name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Resource Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Resource description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "checks",
					Description: "Resource level Check list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonResourceCheck",
							SkelName: "vine.hub.SkeletonResourceCheck",
						},
					},
				},
				{
					Name:        "actions",
					Description: "Action list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonResourceAction",
							SkelName: "vine.hub.SkeletonResourceAction",
						},
					},
				},
				{
					Name:        "checkService",
					Description: "Check service",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonServiceItem",
						SkelName: "vine.hub.SkeletonServiceItem",
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "SkeletonServiceItem",
			SkelName:    "vine.hub.SkeletonServiceItem",
			Description: "Skeleton service items",
			Hash:        "a2c05661",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Service name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Service Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Service Description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "pub",
					Description: "Whether the item is public",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "authMode",
					Description: "Authentication mode",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "require",
					Description: "Permission requirements",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SkeletonPermExpr",
						SkelName: "vine.hub.SkeletonPermExpr",
						Nullable: true,
					},
				},
				{
					Name:        "actors",
					Description: "Accessible Actor List",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonActorRef",
							SkelName: "vine.hub.SkeletonActorRef",
						},
					},
				},
				{
					Name:        "methods",
					Description: "Method list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonMethod",
							SkelName: "vine.hub.SkeletonMethod",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonTask",
			SkelName:    "vine.hub.SkeletonTask",
			Description: "Skeleton task",
			Hash:        "32a7ae30",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Task name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Task Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Task description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "triggers",
					Description: "Trigger list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonTrigger",
							SkelName: "vine.hub.SkeletonTrigger",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonTrigger",
			SkelName:    "vine.hub.SkeletonTrigger",
			Description: "Skeleton task trigger",
			Hash:        "90abe45a",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Trigger name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Trigger Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Trigger description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "inputDescription",
					Description: "Input description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "example",
					Description: "Input example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "SkeletonWebItem",
			SkelName:    "vine.hub.SkeletonWebItem",
			Description: "Skeleton web page",
			Hash:        "97a21b66",
			Members: []*skel.MemberSchema{
				{
					Name:        "domain",
					Description: "Domain",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "mainSchemaHash",
					Description: "Primary skeleton item hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "isMultiVersion",
					Description: "Whether there are multiple valid versions of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "isMain",
					Description: "Whether it is the main version of the skeleton item",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "domainSchemaHash",
					Description: "Owning DomainSchema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "name",
					Description: "Web page name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Web Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Web page description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "actors",
					Description: "Accessible Actor List",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonActorRef",
							SkelName: "vine.hub.SkeletonActorRef",
						},
					},
				},
			},
		},
		{
			Name:        "TaskDebugDefaultLaunchRequest",
			SkelName:    "vine.hub.TaskDebugDefaultLaunchRequest",
			Description: "Task Debug initiates a request by default",
			Hash:        "84d95346",
			Members: []*skel.MemberSchema{
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "argumentsJson",
					Description: "Default task parameters JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
			},
		},
		{
			Name:        "TaskDebugLaunchRequest",
			SkelName:    "vine.hub.TaskDebugLaunchRequest",
			Description: "Task Debug initiates a request",
			Hash:        "49bd9af1",
			Members: []*skel.MemberSchema{
				{
					Name:        "taskSkelName",
					Description: "Task Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Task schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "triggerSkelName",
					Description: "Trigger Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "argumentsJson",
					Description: "Task parameters JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarJson,
					},
				},
				{
					Name:        "traceId",
					Description: "Trace ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "spanId",
					Description: "Span ID",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "TaskDebugTaskItem",
			SkelName:    "vine.hub.TaskDebugTaskItem",
			Description: "Task called by Task Debug",
			Hash:        "0fbcd8bc",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Task name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "taskSkelName",
					Description: "Task Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Task schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Task description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
			},
		},
		{
			Name:        "TaskDebugTriggerItem",
			SkelName:    "vine.hub.TaskDebugTriggerItem",
			Description: "Trigger called by Task Debug",
			Hash:        "1b3a76ef",
			Members: []*skel.MemberSchema{
				{
					Name:        "name",
					Description: "Trigger name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skelName",
					Description: "Trigger Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "description",
					Description: "Trigger description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "inputDescription",
					Description: "Input description",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "example",
					Description: "Input example",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindScalar,
						Scalar:   skel.ScalarString,
						Nullable: true,
					},
				},
				{
					Name:        "arguments",
					Description: "Parameter list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonField",
							SkelName: "vine.hub.SkeletonField",
						},
					},
				},
			},
		},
		{
			Name:        "TaskRunnerCronScheduler",
			SkelName:    "vine.hub.TaskRunnerCronScheduler",
			Description: "Task execution Cron schedule",
			Hash:        "bf26a252",
			Members: []*skel.MemberSchema{
				{
					Name:        "triggerSkelName",
					Description: "Trigger Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "cronExpr",
					Description: "Cron expression",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "TaskRunnerRegistration",
			SkelName:    "vine.hub.TaskRunnerRegistration",
			Description: "Task execution capability registration information provided by the application",
			Hash:        "e86328cc",
			Members: []*skel.MemberSchema{
				{
					Name:        "taskSkelName",
					Description: "Task Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Task schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "timeoutMs",
					Description: "Execution timeout, in milliseconds",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "concurrency",
					Description: "Maximum concurrency",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarInt,
					},
				},
				{
					Name:        "noRetry",
					Description: "Whether to disallow retrying after failure",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
				{
					Name:        "cronSchedulers",
					Description: "Cron schedule list",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskRunnerCronScheduler",
							SkelName: "vine.hub.TaskRunnerCronScheduler",
						},
					},
				},
			},
		},
		{
			Name:        "WebHandlerRegistration",
			SkelName:    "vine.hub.WebHandlerRegistration",
			Description: "Web processing capability registration information provided by the application",
			Hash:        "1291b942",
			Members: []*skel.MemberSchema{
				{
					Name:        "webSkelName",
					Description: "Web Skel name",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "schemaHash",
					Description: "Web schema hash",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "endpoint",
					Description: "Web proxy access address",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
	},
	Webs: []*skel.WebSchema{
		{
			Name:        "DashboardWeb",
			SkelName:    "vine.hub.DashboardWeb",
			Description: "Hub Dashboard Web",
			Hash:        "0a818f1f",
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
		},
	},
	Actors: []*skel.ActorSchema{
		{
			Name:     "AdminActor",
			SkelName: "vine.hub.AdminActor",
			Hash:     "e3c26b2d",
			Vias: []skel.ActorVia{
				skel.ActorViaClient,
			},
			AuthEnabled: false,
			PermEnabled: false,
		},
	},
	Services: []*skel.ServiceSchema{
		{
			Name:        "AppConfigService",
			SkelName:    "vine.hub.AppConfigService",
			Description: "Hub's application configuration service, called by Client",
			Hash:        "358424b9",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "list",
					SkelName:          "list",
					Description:       "List configuration items",
					Hash:              "28569096",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration item list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "AppConfigItem",
							SkelName: "vine.hub.AppConfigItem",
						},
					},
				},
				{
					Name:              "get",
					SkelName:          "get",
					Description:       "Read configuration",
					Hash:              "c22f3d5d",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration items",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Configuration ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "AppConfigItem",
						SkelName: "vine.hub.AppConfigItem",
					},
				},
				{
					Name:              "update",
					SkelName:          "update",
					Description:       "Modify configuration",
					Hash:              "8f781bfd",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration items",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Configuration ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "update",
							Description: "Configuration update parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "AppConfigUpdate",
								SkelName: "vine.hub.AppConfigUpdate",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "AppConfigItem",
						SkelName: "vine.hub.AppConfigItem",
					},
				},
				{
					Name:              "create",
					SkelName:          "create",
					Description:       "Create configuration",
					Hash:              "66c6a8f2",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration items",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "creation",
							Description: "Configuration creation parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "AppConfigCreation",
								SkelName: "vine.hub.AppConfigCreation",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "AppConfigItem",
						SkelName: "vine.hub.AppConfigItem",
					},
				},
				{
					Name:              "remove",
					SkelName:          "remove",
					Description:       "Delete unused configuration",
					Hash:              "8a48c541",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Whether deletion succeeded",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Configuration ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "AppStatusService",
			SkelName:    "vine.hub.AppStatusService",
			Description: "Hub Dashboard's application status service",
			Hash:        "2d56f3ea",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:        "list",
					SkelName:    "list",
					Description: "List application instance statuses currently stored in Redis",
					Hash:        "39c286af",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "AppStatusView",
							SkelName: "vine.hub.AppStatusView",
						},
					},
				},
			},
		},
		{
			Name:        "EventDebugService",
			SkelName:    "vine.hub.EventDebugService",
			Description: "Hub Dashboard Event Debugging Service",
			Hash:        "7ca4a329",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:        "listEvents",
					SkelName:    "listEvents",
					Description: "List the events monitored by the application instance",
					Hash:        "a02ce785",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "EventDebugEventItem",
							SkelName: "vine.hub.EventDebugEventItem",
						},
					},
				},
				{
					Name:        "buildDefaultEmitRequest",
					SkelName:    "buildDefaultEmitRequest",
					Description: "Generate a default Event send request",
					Hash:        "71147d22",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "eventSkelName",
							Description: "Event Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Event schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "EventDebugDefaultEmitRequest",
						SkelName: "vine.hub.EventDebugDefaultEmitRequest",
					},
				},
				{
					Name:        "emitEvent",
					SkelName:    "emitEvent",
					Description: "Send Event",
					Hash:        "f9285eb9",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "request",
							Description: "Debug send request",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "EventDebugEmitRequest",
								SkelName: "vine.hub.EventDebugEmitRequest",
							},
						},
					},
				},
			},
		},
		{
			Name:        "InfoService",
			SkelName:    "vine.hub.InfoService",
			Description: "Hub's information service, called by Link",
			Hash:        "b28553d7",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:              "getInfo",
					SkelName:          "getInfo",
					Description:       "Read Hub information",
					Hash:              "294c9cf9",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Hub information",
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "Info",
						SkelName: "vine.hub.Info",
					},
				},
			},
		},
		{
			Name:        "MaintenanceService",
			SkelName:    "vine.hub.MaintenanceService",
			Description: "Hub maintenance service",
			Hash:        "689cac11",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "previewSeedYaml",
					SkelName:          "previewSeedYaml",
					Description:       "Preview Seed YAML differences",
					Hash:              "6e6cfd95",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Seed preview",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "content",
							Description: "Seed YAML content",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SeedPreview",
						SkelName: "vine.hub.SeedPreview",
					},
				},
				{
					Name:              "applySeedYaml",
					SkelName:          "applySeedYaml",
					Description:       "Apply Seed YAML entity updates",
					Hash:              "77a003af",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Updated Seed preview",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "content",
							Description: "Seed YAML content",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "selections",
							Description: "Entity to update",
							Type: &skel.TypeSchema{
								Kind: skel.TypeKindList,
								Element: &skel.TypeSchema{
									Kind:     skel.TypeKindData,
									Name:     "SeedItemSelection",
									SkelName: "vine.hub.SeedItemSelection",
								},
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "SeedPreview",
						SkelName: "vine.hub.SeedPreview",
					},
				},
			},
		},
		{
			Name:        "PortalCertService",
			SkelName:    "vine.hub.PortalCertService",
			Description: "Hub's Portal site certificate service, called by the Portal management client",
			Hash:        "6bacbfdb",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "list",
					SkelName:          "list",
					Description:       "List Portal site certificates",
					Hash:              "a0b61d80",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal site certificate list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalCert",
							SkelName: "vine.hub.PortalCert",
						},
					},
				},
				{
					Name:              "get",
					SkelName:          "get",
					Description:       "Read the Portal site certificate",
					Hash:              "1eddd9c5",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal site certificate",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Certificate ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCert",
						SkelName: "vine.hub.PortalCert",
					},
				},
				{
					Name:              "create",
					SkelName:          "create",
					Description:       "Create Portal site certificate",
					Hash:              "74a2e85e",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal site certificate",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "creation",
							Description: "Portal site certificate creation parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalCertCreation",
								SkelName: "vine.hub.PortalCertCreation",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCert",
						SkelName: "vine.hub.PortalCert",
					},
				},
				{
					Name:              "update",
					SkelName:          "update",
					Description:       "Modify Portal site certificate",
					Hash:              "074b4aae",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal site certificate",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Certificate ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "update",
							Description: "Portal site certificate update parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalCertUpdate",
								SkelName: "vine.hub.PortalCertUpdate",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalCert",
						SkelName: "vine.hub.PortalCert",
					},
				},
				{
					Name:        "remove",
					SkelName:    "remove",
					Description: "Delete Portal site certificate",
					Hash:        "43f54e86",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Certificate ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
				},
			},
		},
		{
			Name:        "PortalEntryService",
			SkelName:    "vine.hub.PortalEntryService",
			Description: "Hub's Portal access entry service, called by the Portal management client",
			Hash:        "0abee2e7",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "list",
					SkelName:          "list",
					Description:       "List Portal access entries",
					Hash:              "e49ef0bf",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal access entry list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalEntry",
							SkelName: "vine.hub.PortalEntry",
						},
					},
				},
				{
					Name:              "updateAccess",
					SkelName:          "updateAccess",
					Description:       "Modify Portal access configuration",
					Hash:              "32f99bea",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal access entry",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "scheme",
							Description: "Entry protocol",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "host",
							Description: "Match Host, empty string means no restriction",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "port",
							Description: "Entry port",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "update",
							Description: "Portal access entry configuration update parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalEntryAccessUpdate",
								SkelName: "vine.hub.PortalEntryAccessUpdate",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalEntry",
						SkelName: "vine.hub.PortalEntry",
					},
				},
			},
		},
		{
			Name:        "PortalRuleService",
			SkelName:    "vine.hub.PortalRuleService",
			Description: "Hub's Portal entry rule service, called by the Portal management client",
			Hash:        "5232897d",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "list",
					SkelName:          "list",
					Description:       "List Portal entry rules",
					Hash:              "b3f41c0d",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal entry rule list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalRule",
							SkelName: "vine.hub.PortalRule",
						},
					},
				},
				{
					Name:              "get",
					SkelName:          "get",
					Description:       "Read Portal entry rules",
					Hash:              "75d83e0b",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal entry rules",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Rule ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalRule",
						SkelName: "vine.hub.PortalRule",
					},
				},
				{
					Name:              "create",
					SkelName:          "create",
					Description:       "Create Portal entry rules",
					Hash:              "fd503260",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal entry rules",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "creation",
							Description: "Portal entry rule creation parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalRuleCreation",
								SkelName: "vine.hub.PortalRuleCreation",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalRule",
						SkelName: "vine.hub.PortalRule",
					},
				},
				{
					Name:              "update",
					SkelName:          "update",
					Description:       "Modify Portal entry rules",
					Hash:              "15306795",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal entry rules",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Rule ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "update",
							Description: "Portal entry rule update parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalRuleUpdate",
								SkelName: "vine.hub.PortalRuleUpdate",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalRule",
						SkelName: "vine.hub.PortalRule",
					},
				},
				{
					Name:        "remove",
					SkelName:    "remove",
					Description: "Delete Portal entry rules",
					Hash:        "2a299015",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Rule ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
				},
				{
					Name:              "getDashboardAccess",
					SkelName:          "getDashboardAccess",
					Description:       "Get the Hub Dashboard access entry",
					Hash:              "3990edf3",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Hub Dashboard access entry",
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalDashboardAccess",
						SkelName: "vine.hub.PortalDashboardAccess",
					},
				},
				{
					Name:              "updateDashboardAccess",
					SkelName:          "updateDashboardAccess",
					Description:       "Modify Hub Dashboard access entry",
					Hash:              "6979bcd9",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Hub Dashboard entry rules",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "scheme",
							Description: "Hub Dashboard entry protocol",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "host",
							Description: "Hub Dashboard entry host",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "port",
							Description: "Hub Dashboard entry port",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "pathPrefix",
							Description: "Hub Dashboard entry path prefix",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalRule",
							SkelName: "vine.hub.PortalRule",
						},
					},
				},
			},
		},
		{
			Name:        "PortalSiteService",
			SkelName:    "vine.hub.PortalSiteService",
			Description: "Hub's Portal target site service, called by the Portal management client",
			Hash:        "a4d7f889",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "list",
					SkelName:          "list",
					Description:       "List Portal target sites",
					Hash:              "29227f9a",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal target site list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "PortalSite",
							SkelName: "vine.hub.PortalSite",
						},
					},
				},
				{
					Name:              "listOptions",
					SkelName:          "listOptions",
					Description:       "List Portal target site form options",
					Hash:              "f3bcda2f",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal target site form options",
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalSiteOptions",
						SkelName: "vine.hub.PortalSiteOptions",
					},
				},
				{
					Name:              "get",
					SkelName:          "get",
					Description:       "Read the Portal target site",
					Hash:              "ce6b962c",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal target site",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Target site id",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalSite",
						SkelName: "vine.hub.PortalSite",
					},
				},
				{
					Name:              "create",
					SkelName:          "create",
					Description:       "Create Portal target site",
					Hash:              "36b08aa6",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal target site",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "creation",
							Description: "Portal target site creation parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalSiteCreation",
								SkelName: "vine.hub.PortalSiteCreation",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalSite",
						SkelName: "vine.hub.PortalSite",
					},
				},
				{
					Name:              "update",
					SkelName:          "update",
					Description:       "Modify Portal target site",
					Hash:              "799ef4b5",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Portal target site",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Target site id",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
						{
							Name:        "update",
							Description: "Portal target site update parameters",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "PortalSiteUpdate",
								SkelName: "vine.hub.PortalSiteUpdate",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "PortalSite",
						SkelName: "vine.hub.PortalSite",
					},
				},
				{
					Name:        "remove",
					SkelName:    "remove",
					Description: "Delete Portal target site",
					Hash:        "a061405c",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "id",
							Description: "Target site id",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarInt,
							},
						},
					},
				},
			},
		},
		{
			Name:        "RegistryService",
			SkelName:    "vine.hub.RegistryService",
			Description: "Hub's application registration service, called by Link",
			Hash:        "b8e96f22",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "register",
					SkelName:    "register",
					Description: "Register application instance",
					Hash:        "2b2400e1",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "registration",
							Description: "Application instance registration information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "AppRegistration",
								SkelName: "vine.hub.AppRegistration",
							},
						},
					},
				},
				{
					Name:        "unregister",
					SkelName:    "unregister",
					Description: "Unregister an application instance",
					Hash:        "252099ac",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "name",
							Description: "Application name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "instanceId",
							Description: "Application instance ID",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarUuid,
							},
						},
					},
				},
				{
					Name:              "heartbeat",
					SkelName:          "heartbeat",
					Description:       "Application instance heartbeat",
					Hash:              "13306cc3",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Whether the current instance is still registered in the Hub",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "status",
							Description: "Application instance ID",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "AppStatus",
								SkelName: "vine.hub.AppStatus",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "ServiceDebugService",
			SkelName:    "vine.hub.ServiceDebugService",
			Description: "Hub Dashboard Service debugging service",
			Hash:        "fc747d85",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:        "listAppInstances",
					SkelName:    "listAppInstances",
					Description: "List application instances",
					Hash:        "1f187ad5",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceDebugAppInstance",
							SkelName: "vine.hub.ServiceDebugAppInstance",
						},
					},
				},
				{
					Name:        "listServices",
					SkelName:    "listServices",
					Description: "List the services provided by the application instance",
					Hash:        "ccf073aa",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceDebugServiceItem",
							SkelName: "vine.hub.ServiceDebugServiceItem",
						},
					},
				},
				{
					Name:        "listServiceAppInstances",
					SkelName:    "listServiceAppInstances",
					Description: "List application instances that provide the specified service",
					Hash:        "5e64200a",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "serviceSkelName",
							Description: "Service Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Service schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceDebugAppInstance",
							SkelName: "vine.hub.ServiceDebugAppInstance",
						},
					},
				},
				{
					Name:        "listMethods",
					SkelName:    "listMethods",
					Description: "List Service methods",
					Hash:        "0bc9fc82",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "serviceSkelName",
							Description: "Service Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Service schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceDebugMethodItem",
							SkelName: "vine.hub.ServiceDebugMethodItem",
						},
					},
				},
				{
					Name:        "buildDefaultInvokeRequest",
					SkelName:    "buildDefaultInvokeRequest",
					Description: "Generate default Service call request",
					Hash:        "7e15083e",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "serviceSkelName",
							Description: "Service Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Service schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "methodSkelName",
							Description: "Method Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "ServiceDebugDefaultInvokeRequest",
						SkelName: "vine.hub.ServiceDebugDefaultInvokeRequest",
					},
				},
				{
					Name:        "invokeService",
					SkelName:    "invokeService",
					Description: "Call Service method",
					Hash:        "713e122a",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "request",
							Description: "Debug call request",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "ServiceDebugInvokeRequest",
								SkelName: "vine.hub.ServiceDebugInvokeRequest",
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "ServiceDebugInvokeResponse",
						SkelName: "vine.hub.ServiceDebugInvokeResponse",
					},
				},
			},
		},
		{
			Name:        "SkeletonService",
			SkelName:    "vine.hub.SkeletonService",
			Description: "Hub's skeleton service, called by the Portal management client",
			Hash:        "6d2dd869",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:              "listDomains",
					SkelName:          "listDomains",
					Description:       "List Domain skeleton",
					Hash:              "cab1fd4a",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Domain skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonDomain",
							SkelName: "vine.hub.SkeletonDomain",
						},
					},
				},
				{
					Name:              "listActors",
					SkelName:          "listActors",
					Description:       "List Actor Skeleton",
					Hash:              "ea3e3980",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Actor skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonActorItem",
							SkelName: "vine.hub.SkeletonActorItem",
						},
					},
				},
				{
					Name:              "listServices",
					SkelName:          "listServices",
					Description:       "List Service skeleton",
					Hash:              "433b6284",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Service skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonServiceItem",
							SkelName: "vine.hub.SkeletonServiceItem",
						},
					},
				},
				{
					Name:              "listResources",
					SkelName:          "listResources",
					Description:       "List Resource skeleton",
					Hash:              "bccf567d",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Resource skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonResourceItem",
							SkelName: "vine.hub.SkeletonResourceItem",
						},
					},
				},
				{
					Name:              "listWebs",
					SkelName:          "listWebs",
					Description:       "List Web Skeletons",
					Hash:              "710e1526",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Web skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonWebItem",
							SkelName: "vine.hub.SkeletonWebItem",
						},
					},
				},
				{
					Name:              "listTasks",
					SkelName:          "listTasks",
					Description:       "List Task skeleton",
					Hash:              "8598497d",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Task skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonTask",
							SkelName: "vine.hub.SkeletonTask",
						},
					},
				},
				{
					Name:              "listEvents",
					SkelName:          "listEvents",
					Description:       "List Event skeletons",
					Hash:              "9f2d9673",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Event skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonEventItem",
							SkelName: "vine.hub.SkeletonEventItem",
						},
					},
				},
				{
					Name:              "listData",
					SkelName:          "listData",
					Description:       "List Data skeleton",
					Hash:              "ef387f2a",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Data skeleton list, including Enum",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonData",
							SkelName: "vine.hub.SkeletonData",
						},
					},
				},
				{
					Name:              "listConfigs",
					SkelName:          "listConfigs",
					Description:       "List Config skeleton",
					Hash:              "c7afc1fa",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Config skeleton list",
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "SkeletonConfigItem",
							SkelName: "vine.hub.SkeletonConfigItem",
						},
					},
				},
			},
		},
		{
			Name:        "TaskDebugService",
			SkelName:    "vine.hub.TaskDebugService",
			Description: "Hub Dashboard Task Debugging Service",
			Hash:        "230a17bd",
			Pub:         false,
			AuthMode:    skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
			},
			Methods: []*skel.MethodSchema{
				{
					Name:        "listTasks",
					SkelName:    "listTasks",
					Description: "List the tasks provided by the application instance",
					Hash:        "7ee3f91c",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskDebugTaskItem",
							SkelName: "vine.hub.TaskDebugTaskItem",
						},
					},
				},
				{
					Name:        "listTriggers",
					SkelName:    "listTriggers",
					Description: "List Task triggers",
					Hash:        "5360f9a4",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "taskSkelName",
							Description: "Task Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Task schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskDebugTriggerItem",
							SkelName: "vine.hub.TaskDebugTriggerItem",
						},
					},
				},
				{
					Name:        "buildDefaultLaunchRequest",
					SkelName:    "buildDefaultLaunchRequest",
					Description: "Generate a default Task launch request",
					Hash:        "ab8fa029",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "taskSkelName",
							Description: "Task Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "schemaHash",
							Description: "Task schema hash",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
						{
							Name:        "triggerSkelName",
							Description: "Trigger Skel name",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "TaskDebugDefaultLaunchRequest",
						SkelName: "vine.hub.TaskDebugDefaultLaunchRequest",
					},
				},
				{
					Name:        "launchTask",
					SkelName:    "launchTask",
					Description: "Initiate Task",
					Hash:        "d79ac768",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "request",
							Description: "Debug launch request",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "TaskDebugLaunchRequest",
								SkelName: "vine.hub.TaskDebugLaunchRequest",
							},
						},
					},
				},
			},
		},
	},
}
