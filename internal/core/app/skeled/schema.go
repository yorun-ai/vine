package skeled

import "go.yorun.ai/vine/internal/core/skel"

func init() {
	skel.RegisterDomainSchema(_DomainSchema)
}

var _DomainSchema = &skel.DomainSchema{
	Domain:      "vine.app",
	Description: "Internal API for vine framework",
	Hash:        "c7fbe5cb",
	Full:        true,
	Generated: &skel.GeneratedInfo{
		CompilerVersion: "v0.9.0",
	},
	Data: []*skel.DataSchema{
		{
			Name:        "EventOn",
			SkelName:    "vine.app.EventOn",
			Description: "Link triggers event processing information to the App",
			Hash:        "e2794713",
			Members: []*skel.MemberSchema{
				{
					Name:        "metadata",
					Description: "Event meta information",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "EventOnMeta",
						SkelName: "vine.app.EventOnMeta",
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
			Name:        "EventOnMeta",
			SkelName:    "vine.app.EventOnMeta",
			Description: "Link triggers event processing meta-information to the App",
			Hash:        "39feb0b1",
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
				{
					Name:        "emittedAt",
					Description: "The time the event was sent",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarTimestamp,
					},
				},
			},
		},
		{
			Name:        "TaskRun",
			SkelName:    "vine.app.TaskRun",
			Description: "Link triggers task execution information to the App",
			Hash:        "1669399b",
			Members: []*skel.MemberSchema{
				{
					Name:        "metadata",
					Description: "Task trigger meta information",
					Type: &skel.TypeSchema{
						Kind:     skel.TypeKindData,
						Name:     "TaskRunMeta",
						SkelName: "vine.app.TaskRunMeta",
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
			Name:        "TaskRunMeta",
			SkelName:    "vine.app.TaskRunMeta",
			Description: "Link triggers meta-information of task execution to App",
			Hash:        "3a7960ca",
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
				{
					Name:        "launchedAt",
					Description: "Task launch time",
					Type: &skel.TypeSchema{
						Kind:   skel.TypeKindScalar,
						Scalar: skel.ScalarTimestamp,
					},
				},
			},
		},
	},
	Services: []*skel.ServiceSchema{
		{
			Name:        "ConsoleService",
			SkelName:    "vine.app.ConsoleService",
			Description: "App's console service, called by Link",
			Hash:        "edd76e05",
			Pub:         false,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "ping",
					SkelName:    "ping",
					Description: "Application health check",
					Hash:        "f08a8610",
					AuthMode:    skel.AuthModeUnset,
				},
			},
		},
		{
			Name:        "EventService",
			SkelName:    "vine.app.EventService",
			Description: "App's event processing service, called by Link",
			Hash:        "e2f6b7ae",
			Pub:         false,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "onEvent",
					SkelName:    "onEvent",
					Description: "Trigger event processing",
					Hash:        "15692206",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "on",
							Description: "Event handling information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "EventOn",
								SkelName: "vine.app.EventOn",
							},
						},
					},
				},
			},
		},
		{
			Name:        "TaskService",
			SkelName:    "vine.app.TaskService",
			Description: "App's task execution service, called by Link",
			Hash:        "56e140ce",
			Pub:         false,
			AuthMode:    skel.AuthModeUnset,
			Methods: []*skel.MethodSchema{
				{
					Name:        "runTask",
					SkelName:    "runTask",
					Description: "Trigger task execution",
					Hash:        "a440c23c",
					AuthMode:    skel.AuthModeUnset,
					Arguments: []*skel.MemberSchema{
						{
							Name:        "run",
							Description: "Task execution information",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "TaskRun",
								SkelName: "vine.app.TaskRun",
							},
						},
					},
				},
			},
		},
	},
}
