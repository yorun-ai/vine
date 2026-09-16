package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

type portalSiteRepoSpy struct {
	calls   []string
	entries map[int]*PortalSite
}

func (s *portalSiteRepoSpy) List() []*PortalSite {
	s.calls = append(s.calls, "List")
	entries := make([]*PortalSite, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}
	return vslice.SortBy(entries, func(a *PortalSite, b *PortalSite) bool {
		return a.Id < b.Id
	})
}

func (s *portalSiteRepoSpy) GetById(id int) (*PortalSite, bool) {
	s.calls = append(s.calls, "GetById")
	entry, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	value := *entry
	return &value, true
}

func (s *portalSiteRepoSpy) GetByName(name string) (*PortalSite, bool) {
	s.calls = append(s.calls, "GetByName:"+name)
	for _, entry := range s.entries {
		if entry.Name == name {
			value := *entry
			return &value, true
		}
	}
	return nil, false
}

func (s *portalSiteRepoSpy) Save(entry *PortalSite) {
	s.calls = append(s.calls, "Save")
	if s.entries == nil {
		s.entries = map[int]*PortalSite{}
	}
	value := *entry
	s.entries[value.Id] = &value
}

func (s *portalSiteRepoSpy) Remove(id int) bool {
	s.calls = append(s.calls, "Remove")
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
	incoming.Id = 99
	got := target.Save(incoming)
	require.Equal(t, 7, got.Id)
	require.Panics(t, func() { target.Update(7, PortalSiteUpdate{WebName: new("")}) })
	require.Equal(t, "demo.Web", repo.entries[7].WebName)
}
