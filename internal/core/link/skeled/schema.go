package skeled

import "go.yorun.ai/vine/internal/core/skel"

func init() {
	skel.RegisterDomainSchema(_DomainSchema)
}

var _DomainSchema = &skel.DomainSchema{
	Domain:      "vine.link",
	Description: "Internal API for Vine Link",
	Hash:        "b383d519",
	Full:        true,
	Generated: &skel.GeneratedInfo{
		CompilerVersion: "v0.9.0",
	},
	Data: []*skel.DataSchema{
		{
			Name:        "AppRegistration",
			SkelName:    "vine.link.AppRegistration",
			Description: "Application information registered by App to Link",
			Hash:        "2d94718c",
			Members: []*skel.MemberSchema{
				{
					Name:        "consoleEndpoint",
					Description: "Console access endpoint of the current application",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "serviceEndpoint",
					Description: "The Rpc service access endpoint of the current application",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "webEndpointPrefix",
					Description: "Web access endpoint prefix of the current application",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "eventEndpoint",
					Description: "The event processing access endpoint of the current application",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "taskEndpoint",
					Description: "The task execution endpoint of the current application",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "serviceHandlers",
					Description: "List of Rpc service processing capabilities provided by the current application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "ServiceHandlerRegistration",
							SkelName: "vine.link.ServiceHandlerRegistration",
						},
					},
				},
				{
					Name:        "webHandlers",
					Description: "List of web processing capabilities provided by the current application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "WebHandlerRegistration",
							SkelName: "vine.link.WebHandlerRegistration",
						},
					},
				},
				{
					Name:        "eventListeners",
					Description: "List of event listening capabilities provided by the current application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "EventListenerRegistration",
							SkelName: "vine.link.EventListenerRegistration",
						},
					},
				},
				{
					Name:        "taskRunners",
					Description: "List of task execution capabilities provided by the current application",
					Type: &skel.TypeSchema{
						Kind: skel.TypeKindList,
						Element: &skel.TypeSchema{
							Kind:     skel.TypeKindData,
							Name:     "TaskRunnerRegistration",
							SkelName: "vine.link.TaskRunnerRegistration",
						},
					},
				},
				{
					Name:        "domainSchemas",
					Description: "List of all DomainSchemas registered by the current application",
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
			Name:        "BootInfo",
			SkelName:    "vine.link.BootInfo",
			Description: "Key information obtained from Link when the App starts",
			Hash:        "5afa4f41",
			Members: []*skel.MemberSchema{
				{
					Name:        "rpcProxyEndpointPath",
					Description: "The endpoint path that should be accessed when the current application initiates Rpc",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "skipDomainSchemas",
					Description: "Whether to skip DomainSchema when app is registered",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarBool,
					},
				},
			},
		},
		{
			Name:        "EventEmission",
			SkelName:    "vine.link.EventEmission",
			Description: "Event dispatch information sent from App to Link",
			Hash:        "ae1412d6",
			Members: []*skel.MemberSchema{
				{
					Name:        "metadata",
					Description: "Event meta information",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "EventEmissionMeta",
						SkelName: "vine.link.EventEmissionMeta",
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
					Name:        "eventJson",
					Description: "Event JSON",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "EventEmissionMeta",
			SkelName:    "vine.link.EventEmissionMeta",
			Description: "Meta information carried when App initiates event sending to Link",
			Hash:        "2d0ec63d",
			Members: []*skel.MemberSchema{
				{
					Name:        "traceId",
					Description: "Trace ID of the current calling link",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "traceSpan",
					Description: "Trace Span of the current calling link",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appName",
					Description: "The application name that sent the event",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appVersion",
					Description: "The application version that sent the event",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appInstanceId",
					Description: "The application instance ID that sent the event",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarUuid,
					},
				},
			},
		},
		{
			Name:        "EventListenerRegistration",
			SkelName:    "vine.link.EventListenerRegistration",
			Description: "Event listening capability registration information provided by the application",
			Hash:        "13d38eca",
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
			Name:        "ServiceHandlerRegistration",
			SkelName:    "vine.link.ServiceHandlerRegistration",
			Description: "Rpc service processing capability registration information provided by the application",
			Hash:        "ae71bb58",
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
			Name:        "TaskLaunch",
			SkelName:    "vine.link.TaskLaunch",
			Description: "Task trigger information initiated by App to Link",
			Hash:        "2de22603",
			Members: []*skel.MemberSchema{
				{
					Name:        "metadata",
					Description: "Task trigger meta information",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "TaskLaunchMeta",
						SkelName: "vine.link.TaskLaunchMeta",
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
					Name:        "triggerSkelName",
					Description: "Task trigger Skel name",
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
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "TaskLaunchMeta",
			SkelName:    "vine.link.TaskLaunchMeta",
			Description: "The meta information carried when the App initiates a task to Link",
			Hash:        "c7f14ff5",
			Members: []*skel.MemberSchema{
				{
					Name:        "traceId",
					Description: "Trace ID of the current calling link",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "traceSpan",
					Description: "Trace Span of the current calling link",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appName",
					Description: "Application name that initiated the task",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appVersion",
					Description: "The application version that initiated the task",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:        "appInstanceId",
					Description: "The application instance ID that initiated the task",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarUuid,
					},
				},
			},
		},
		{
			Name:        "TaskRunnerCronScheduler",
			SkelName:    "vine.link.TaskRunnerCronScheduler",
			Description: "Task execution Cron schedule",
			Hash:        "66cfc915",
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
			SkelName:    "vine.link.TaskRunnerRegistration",
			Description: "Task execution capability registration information provided by the application",
			Hash:        "bfdb3e6b",
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
							SkelName: "vine.link.TaskRunnerCronScheduler",
						},
					},
				},
			},
		},
		{
			Name:        "WebHandlerRegistration",
			SkelName:    "vine.link.WebHandlerRegistration",
			Description: "Web processing capability registration information provided by the application",
			Hash:        "8062b39a",
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
			},
		},
	},
	Services: []*skel.ServiceSchema{
		{
			Name:        "BootService",
			SkelName:    "vine.link.BootService",
			Description: "Link's startup information service, called by the App",
			Hash:        "e8b9fae5",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "getInfo",
					SkelName:    "getInfo",
					Description: "Get key startup information",
					Hash:        "833dc898",
					AuthMode:    skel.AuthModeUnset,
					ResultType: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "BootInfo",
						SkelName: "vine.link.BootInfo",
					},
				},
			},
		},
		{
			Name:        "ConfigService",
			SkelName:    "vine.link.ConfigService",
			Description: "Link's application configuration service, called by the App",
			Hash:        "420d2cb7",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:              "getEternal",
					SkelName:          "getEternal",
					Description:       "Read Eternal configuration",
					Hash:              "0b48886e",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration JSON",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "key",
							Description: "Configuration key",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
				{
					Name:              "getInstant",
					SkelName:          "getInstant",
					Description:       "Read Instant configuration",
					Hash:              "50b06f0e",
					AuthMode:          skel.AuthModeUnset,
					OutputDescription: "Configuration JSON",
					Arguments: []*skel.MemberSchema{
						{
							Name:        "key",
							Description: "Configuration key",
							Type: &skel.TypeSchema{
								Kind:   skel.TypeKindScalar,
								Scalar: skel.ScalarString,
							},
						},
					},
					ResultType: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarString,
					},
				},
			},
		},
		{
			Name:        "EventService",
			SkelName:    "vine.link.EventService",
			Description: "Link's event service, called by App",
			Hash:        "f1d171cc",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "emitEvent",
					SkelName:    "emitEvent",
					Description: "Send event",
					Hash:        "0cdc54cf",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "emission",
							Description: "Event sending information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "EventEmission",
								SkelName: "vine.link.EventEmission",
							},
						},
					},
				},
			},
		},
		{
			Name:        "RegistryService",
			SkelName:    "vine.link.RegistryService",
			Description: "Link's application registration service, called by the App",
			Hash:        "6f5f2981",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "register",
					SkelName:    "register",
					Description: "Register the currently running application",
					Hash:        "1eb5f2a4",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "registration",
							Description: "Application instance registration information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "AppRegistration",
								SkelName: "vine.link.AppRegistration",
							},
						},
					},
				},
				{
					Name:        "unregister",
					SkelName:    "unregister",
					Description: "Log out of the currently running application and prepare to exit gracefully. This call may block until the current instance's in-progress work on the Link side is drained, or until the wait times out",
					Hash:        "3914ba71",
					AuthMode:    skel.AuthModeUnset,
				},
			},
		},
		{
			Name:        "TaskService",
			SkelName:    "vine.link.TaskService",
			Description: "Link's task service, called by the App",
			Hash:        "26a1aef1",
			Pub:         true,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "launchTask",
					SkelName:    "launchTask",
					Description: "Start a task",
					Hash:        "ea8345fb",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "launch",
							Description: "Task trigger information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "TaskLaunch",
								SkelName: "vine.link.TaskLaunch",
							},
						},
					},
				},
			},
		},
	},
}
