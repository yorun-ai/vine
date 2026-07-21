package schema

import "go.yorun.ai/vine/internal/core/skel"

func resetMemorySchemaRepoForTest() {
	memoryDomainSchemaMutex.Lock()
	defer memoryDomainSchemaMutex.Unlock()
	memoryDomainSchemaSequence = 0
	memoryDomainSchemaByHash = map[string]*_MemoryDomainSchemaEntry{}
	memoryDomainSchemaHashesByDomain = map[string]map[string]struct{}{}
	memorySchemaHashesByOwner = map[string]map[string]struct{}{}
	memorySchemaSnapshot = _MemorySchemaSnapshot{}
}

func testDomainSchema() *skel.DomainSchema {
	return &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "pkg-hash-1",
		Data: []*skel.DataSchema{
			{
				Name:     "User",
				SkelName: "demo.user.User",
				Hash:     "data-hash-1",
			},
		},
		Actors: []*skel.ActorSchema{
			{
				Name:     "AdminActor",
				SkelName: "demo.user.AdminActor",
				Hash:     "actor-hash-1",
				Vias:     []skel.ActorVia{skel.ActorViaClient},
			},
		},
		Configs: []*skel.ConfigSchema{
			{
				Name:      "MainConfig",
				SkelName:  "demo.user.MainConfig",
				Hash:      "config-hash-1",
				Lifecycle: "ETERNAL",
			},
		},
		Services: []*skel.ServiceSchema{
			{
				Name:     "UserService",
				SkelName: "demo.user.UserService",
				Hash:     "service-hash-1",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "AdminActor", SkelName: "demo.user.AdminActor"},
				},
			},
		},
		Tasks: []*skel.TaskSchema{
			{
				Name:     "SyncTask",
				SkelName: "demo.user.SyncTask",
				Hash:     "task-hash-1",
			},
		},
		Events: []*skel.EventSchema{
			{
				Name:     "UserChangedEvent",
				SkelName: "demo.user.UserChangedEvent",
				Hash:     "event-hash-1",
			},
		},
		Webs: []*skel.WebSchema{
			{
				Name:     "DashboardWeb",
				SkelName: "demo.user.DashboardWeb",
				Hash:     "web-hash-1",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "AdminActor", SkelName: "demo.user.AdminActor"},
				},
			},
		},
	}
}
