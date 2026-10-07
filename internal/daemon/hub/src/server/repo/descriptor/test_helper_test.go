package descriptor

import (
	skeldesc "go.yorun.ai/skel/descriptor"
)

func testDomainDescriptor() *skeldesc.Domain {
	return &skeldesc.Domain{
		Name: "demo.user",
		Hash: "pkg-hash-1",
		Data: []*skeldesc.Data{
			{
				Name:     "User",
				SkelName: "demo.user.User",
				Hash:     "data-hash-1",
			},
		},
		Actors: []*skeldesc.Actor{
			{
				Name:     "AdminActor",
				SkelName: "demo.user.AdminActor",
				Hash:     "actor-hash-1",
				Vias:     []skeldesc.ActorViaKind{skeldesc.ActorViaClient},
			},
		},
		Configs: []*skeldesc.Config{
			{
				Name:      "MainConfig",
				SkelName:  "demo.user.MainConfig",
				Hash:      "config-hash-1",
				Lifecycle: skeldesc.ConfigLifecycleEternal,
			},
		},
		Services: []*skeldesc.Service{
			{
				Name: "UserService", Api: true,
				SkelName: "demo.user.UserService",
				Hash:     "service-hash-1",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "AdminActor", SkelName: "demo.user.AdminActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		},
		Tasks: []*skeldesc.Task{
			{
				Name:     "SyncTask",
				SkelName: "demo.user.SyncTask",
				Hash:     "task-hash-1",
			},
		},
		Events: []*skeldesc.Event{
			{
				Name:     "UserChangedEvent",
				SkelName: "demo.user.UserChangedEvent",
				Hash:     "event-hash-1",
			},
		},
		Webs: []*skeldesc.Web{
			{
				Name:     "DashboardWeb",
				SkelName: "demo.user.DashboardWeb",
				Hash:     "web-hash-1",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "AdminActor", SkelName: "demo.user.AdminActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
}
