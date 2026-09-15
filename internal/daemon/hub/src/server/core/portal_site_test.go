package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

type portalSiteRepoSpy struct {
	calls   []string
	entries map[int]*PortalSite
}

func (s *portalSiteRepoSpy) ListEntries() []PortalSite {
	s.calls = append(s.calls, "ListEntries")
	entries := make([]PortalSite, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, *entry)
	}
	return vslice.SortBy(entries, func(a PortalSite, b PortalSite) bool {
		return a.Id < b.Id
	})
}

func (s *portalSiteRepoSpy) GetEntryById(id int) (*PortalSite, bool) {
	s.calls = append(s.calls, "GetEntryById")
	entry, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	value := *entry
	return &value, true
}

func (s *portalSiteRepoSpy) GetEntryByName(name string) (*PortalSite, bool) {
	s.calls = append(s.calls, "GetEntryByName:"+name)
	for _, entry := range s.entries {
		if entry.Name == name {
			value := *entry
			return &value, true
		}
	}
	return nil, false
}

func (s *portalSiteRepoSpy) SaveEntry(entry *PortalSite) {
	s.calls = append(s.calls, "SaveEntry")
	if s.entries == nil {
		s.entries = map[int]*PortalSite{}
	}
	value := *entry
	s.entries[value.Id] = &value
}

func (s *portalSiteRepoSpy) RemoveEntry(id int) bool {
	s.calls = append(s.calls, "RemoveEntry")
	if _, ok := s.entries[id]; !ok {
		return false
	}
	delete(s.entries, id)
	return true
}

// newPortalSiteCoreForTest builds a site core with the repositories Hub injects,
// so tests only choose the repositories they exercise.
func newPortalSiteCoreForTest(repo PortalSiteRepo) *PortalSiteCore {
	return &PortalSiteCore{PortalSiteRepo: repo, SchemaRepo: &schemaRepoSpy{}}
}

func TestPortalSiteCoreUpdateBuiltInSite(t *testing.T) {
	repo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "vine.hub.admin.DashboardWeb-web", BuiltIn: true},
		},
	}
	core := newPortalSiteCoreForTest(repo)

	panicValue := capturePanic(func() {
		core.Update(1, PortalSiteUpdate{})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetEntryById"}, repo.calls)
}

func TestPortalSiteCoreListSkipsBuiltInSites(t *testing.T) {
	repo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "vine.hub.admin.DashboardWeb-web", BuiltIn: true},
			2: {Id: 2, Name: "demo-booker"},
		},
	}
	core := newPortalSiteCoreForTest(repo)

	entries := core.List()

	require.Len(t, entries, 1)
	assert.Equal(t, "demo-booker", entries[0].Name)
	assert.Equal(t, []string{"ListEntries"}, repo.calls)
}

func TestPortalSiteCoreDerivesWebMountPath(t *testing.T) {
	repo := &portalSiteRepoSpy{entries: map[int]*PortalSite{
		1: {Id: 1, Name: "web", Type: PortalSiteTypeWEBGW, WebName: "demo.Web"},
		2: {Id: 2, Name: "plain", Type: PortalSiteTypeWEBGW, WebName: "app.Web"},
		3: {Id: 3, Name: "api", Type: PortalSiteTypeRPCGW, WebName: "demo.Web"},
	}}
	core := &PortalSiteCore{
		PortalSiteRepo: repo,
		SchemaRepo: &schemaRepoSpy{webs: []*skel.WebSchema{
			{Name: "demo.Web", SkelName: "demo.Web", MountPath: "/demo"},
			{Name: "app.Web", SkelName: "app.Web"},
		}},
	}

	entries := core.List()

	require.Len(t, entries, 3)
	assert.Equal(t, []string{"/demo", "", ""}, []string{
		entries[0].WebMountPath,
		entries[1].WebMountPath,
		entries[2].WebMountPath,
	})
	assert.Equal(t, "/demo", core.Get(1).WebMountPath)
	site, ok := core.GetByName("web")
	require.True(t, ok)
	assert.Equal(t, "/demo", site.WebMountPath)
	_, ok = core.GetByName("missing")
	assert.False(t, ok)
}

func TestPortalSiteCoreIgnoresProvidedWebMountPath(t *testing.T) {
	core := &PortalSiteCore{
		PortalSiteRepo: &portalSiteRepoSpy{},
		SchemaRepo:     &schemaRepoSpy{webs: []*skel.WebSchema{{Name: "demo.Web", SkelName: "demo.Web", MountPath: "/demo"}}},
	}

	web := core.withWebMountPath(PortalSite{Type: PortalSiteTypeWEBGW, WebName: "demo.Web", WebMountPath: "/forged"})
	assert.Equal(t, "/demo", web.WebMountPath)
	rpc := core.withWebMountPath(PortalSite{Type: PortalSiteTypeRPCGW, WebMountPath: "/forged"})
	assert.Empty(t, rpc.WebMountPath)
}

func TestPortalSiteCoreRemoveBuiltInSite(t *testing.T) {
	repo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "vine.hub.admin.DashboardWeb-web", BuiltIn: true},
		},
	}
	core := newPortalSiteCoreForTest(repo)

	panicValue := capturePanic(func() {
		core.Remove(1)
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetEntryById"}, repo.calls)
}

func TestMatchPortalSiteRpcgwServicesInDomainViewsIncludesVineSchemas(t *testing.T) {
	site := PortalSite{
		Type:          PortalSiteTypeRPCGW,
		ActorSkelName: "vine.hub.admin.AdminActor",
		ActorVia:      "client",
	}
	views := []DomainSchemaView{{
		DomainVersion: DomainSchemaVersion{
			Main: true,
			Schema: &skel.DomainSchema{
				Services: []*skel.ServiceSchema{
					{
						SkelName: "vine.hub.admin.PortalSiteApiService",
						Audiences: []*skel.ActorAudienceSchema{
							{SkelName: "vine.hub.admin.AdminActor"},
						},
					},
					{
						SkelName: "demo.UserService",
						Audiences: []*skel.ActorAudienceSchema{
							{SkelName: "demo.UserActor"},
						},
					},
				},
			},
		},
	}}

	services := MatchPortalSiteRpcgwServicesInDomainViews(site, views)

	assert.Equal(t, []string{"vine.hub.admin.PortalSiteApiService"}, services)
}

func TestMatchPortalSiteRpcgwServicesInDomainViewsMatchesActorVia(t *testing.T) {
	site := PortalSite{
		Type:          PortalSiteTypeRPCGW,
		ActorSkelName: "demo.UserActor",
		ActorVia:      "client",
	}
	views := []DomainSchemaView{{
		DomainVersion: DomainSchemaVersion{
			Main: true,
			Schema: &skel.DomainSchema{Services: []*skel.ServiceSchema{
				{
					SkelName: "demo.ClientService",
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.UserActor", Via: skel.ActorViaClient},
					},
				},
				{
					SkelName: "demo.AgentService",
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.UserActor", Via: skel.ActorViaAgent},
					},
				},
				{
					SkelName: "demo.AllViaService",
					Audiences: []*skel.ActorAudienceSchema{
						{SkelName: "demo.UserActor"},
					},
				},
			}},
		},
	}}

	services := MatchPortalSiteRpcgwServicesInDomainViews(site, views)

	assert.Equal(t, []string{"demo.AllViaService", "demo.ClientService"}, services)
}

func testUserSite() PortalSite {
	return PortalSite{Name: "demo-web", Type: PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", ActorVia: "client", WebName: "demo.Web"}
}

func TestPortalSiteValidateWithoutStorage(t *testing.T) {
	target := &PortalSiteCore{}
	got := target.Validate(testUserSite())
	require.Equal(t, PortalCorsModeSameDomain, got.Cors.Mode)
	for name, mutate := range map[string]func(*PortalSite){
		"name":        func(s *PortalSite) { s.Name = " " },
		"reserved":    func(s *PortalSite) { s.Name = DashboardWebSiteName },
		"type":        func(s *PortalSite) { s.Type = "unknown" },
		"actor":       func(s *PortalSite) { s.ActorSkelName = "" },
		"via":         func(s *PortalSite) { s.ActorVia = "unknown" },
		"web":         func(s *PortalSite) { s.WebName = "" },
		"cors mode":   func(s *PortalSite) { s.Cors.Mode = "unknown" },
		"cors origin": func(s *PortalSite) { s.Cors.AllowedOrigins = []string{"https://demo.local/path"} },
	} {
		t.Run(name, func(t *testing.T) {
			site := testUserSite()
			mutate(&site)
			require.Panics(t, func() { target.Validate(site) })
		})
	}
}

func TestPortalSiteSaveAndUpdateProtectIdentityAndValidate(t *testing.T) {
	site := testUserSite()
	site.Id = 7
	repo := &portalSiteRepoSpy{entries: map[int]*PortalSite{7: &site}}
	target := newPortalSiteCoreForTest(repo)
	incoming := testUserSite()
	incoming.Id, incoming.BuiltIn = 99, true
	got := target.Save(incoming)
	require.Equal(t, 7, got.Id)
	require.False(t, got.BuiltIn)
	require.Panics(t, func() { target.Update(7, PortalSiteUpdate{WebName: new("")}) })
	require.Equal(t, "demo.Web", repo.entries[7].WebName)
	repo.entries[7].BuiltIn = true
	require.Panics(t, func() { target.Save(incoming) })
	require.True(t, repo.entries[7].BuiltIn)
}

func TestEnsureDashboardSitePreservesIdentity(t *testing.T) {
	site := testUserSite()
	site.Name, site.Id = DashboardWebSiteName, 7
	repo := &portalSiteRepoSpy{entries: map[int]*PortalSite{7: &site}}
	target := newPortalSiteCoreForTest(repo)
	incoming := site
	incoming.Id = 99
	target.EnsureDashboardSite(incoming)
	require.True(t, repo.entries[7].BuiltIn)
	require.Len(t, repo.entries, 1)
	require.Panics(t, func() { target.Save(site) })
	require.Panics(t, func() { target.EnsureDashboardSite(testUserSite()) })
}
