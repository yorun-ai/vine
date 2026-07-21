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

func TestPortalSiteCoreUpdateBuiltInSite(t *testing.T) {
	repo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "vine.hub.DashboardWeb-web", BuiltIn: true},
		},
	}
	core := &PortalSiteCore{PortalSiteRepo: repo}

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
			1: {Id: 1, Name: "vine.hub.DashboardWeb-web", BuiltIn: true},
			2: {Id: 2, Name: "demo-booker"},
		},
	}
	core := &PortalSiteCore{PortalSiteRepo: repo}

	entries := core.List()

	require.Len(t, entries, 1)
	assert.Equal(t, "demo-booker", entries[0].Name)
	assert.Equal(t, []string{"ListEntries"}, repo.calls)
}

func TestPortalSiteCoreRemoveBuiltInSite(t *testing.T) {
	repo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "vine.hub.DashboardWeb-web", BuiltIn: true},
		},
	}
	core := &PortalSiteCore{PortalSiteRepo: repo}

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
		ActorSkelName: "vine.hub.AdminActor",
		ActorVia:      "client",
	}
	views := []DomainSchemaView{{
		DomainVersion: DomainSchemaVersion{
			Main: true,
			Schema: &skel.DomainSchema{
				Services: []*skel.ServiceSchema{
					{
						SkelName: "vine.hub.PortalSiteService",
						Audiences: []*skel.ActorAudienceSchema{
							{SkelName: "vine.hub.AdminActor"},
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

	assert.Equal(t, []string{"vine.hub.PortalSiteService"}, services)
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
