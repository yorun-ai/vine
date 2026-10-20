package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/webtimeout"
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
	return &PortalSiteCore{PortalSiteRepo: repo, DescriptorRepo: &descriptorRepoSpy{}}
}

func TestMatchPortalSiteRpcgwServicesInDomainViewsIncludesVineDescriptors(t *testing.T) {
	site := PortalSite{
		Type:          PortalSiteTypeRPCGW,
		ActorSkelName: "vine.hub.admin.AdminActor",
	}
	views := []DomainDescriptorView{{
		DomainVersion: DomainDescriptorVersion{
			Main: true,
			Descriptor: &skeldesc.Domain{
				Services: []*skeldesc.Service{
					{
						Api:      true,
						SkelName: "vine.hub.admin.PortalSiteApiService",
						Audiences: []*skeldesc.ActorAudience{
							{SkelName: "vine.hub.admin.AdminActor"},
						}, AuthMode: skeldesc.AuthModeRequired,
					},
					{
						Api:      true,
						SkelName: "demo.UserService",
						Audiences: []*skeldesc.ActorAudience{
							{SkelName: "demo.UserActor"},
						}, AuthMode: skeldesc.AuthModeRequired,
					},
				}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			},
		},
	}}

	services := MatchPortalSiteRpcgwServicesInDomainViews(site, views)

	assert.Equal(t, []string{"vine.hub.admin.PortalSiteApiService"}, services)
}

func TestMatchPortalSiteRpcgwServicesInDomainViewsRequiresActorAudience(t *testing.T) {
	site := PortalSite{Type: PortalSiteTypeRPCGW, ActorSkelName: "demo.UserActor"}
	services := []*skeldesc.Service{
		{Api: true, SkelName: "demo.UserApiService", Audiences: []*skeldesc.ActorAudience{{SkelName: site.ActorSkelName}}},
		{Api: true, SkelName: "demo.OtherActorService", Audiences: []*skeldesc.ActorAudience{{SkelName: "demo.OtherActor"}}},
		{SkelName: "demo.InternalService"},
	}
	views := []DomainDescriptorView{{DomainVersion: DomainDescriptorVersion{
		Main: true, Descriptor: new(skeldesc.Domain{Services: services}),
	}}}
	assert.Equal(t, []string{"demo.UserApiService"}, MatchPortalSiteRpcgwServicesInDomainViews(site, views))
}

func testUserSite() PortalSite {
	return PortalSite{Name: "demo-web", Type: PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", WebName: "demo.Web"}
}

func TestPortalSiteInspectSwitchDefaultsAndUpdates(t *testing.T) {
	repo := new(portalSiteRepoSpy)
	target := newPortalSiteCoreForTest(repo)
	site := target.Create(PortalSiteCreation{Name: "api", Type: PortalSiteTypeRPCGW, ActorSkelName: "demo.Actor"})
	require.False(t, site.InspectEnabled)
	site = target.Update(site.Id, PortalSiteUpdate{InspectEnabled: new(true)})
	require.True(t, site.InspectEnabled)
	site = target.Update(site.Id, PortalSiteUpdate{Name: new("renamed")})
	require.True(t, site.InspectEnabled, "unrelated updates preserve the switch")
	site.FieldSources = FieldSources{"/inspectEnabled": {Source: "seed"}}
	repo.Save(site)
	site = target.Update(site.Id, PortalSiteUpdate{InspectEnabled: new(false)})
	require.False(t, site.InspectEnabled)
	require.Equal(t, "hub", site.FieldSources["/inspectEnabled"].Override)
	web := testUserSite()
	web.InspectEnabled = true
	require.Panics(t, func() { target.Validate(web) })
}

func TestPortalSiteMcpSwitchDefaultsAndUpdates(t *testing.T) {
	repo := new(portalSiteRepoSpy)
	target := newPortalSiteCoreForTest(repo)
	site := target.Create(PortalSiteCreation{Name: "api", Type: PortalSiteTypeRPCGW, ActorSkelName: "demo.Actor"})
	require.False(t, site.McpEnabled)
	site = target.Update(site.Id, PortalSiteUpdate{McpEnabled: new(true)})
	require.True(t, site.McpEnabled)
	site = target.Update(site.Id, PortalSiteUpdate{Name: new("renamed")})
	require.True(t, site.McpEnabled, "unrelated updates preserve the switch")
	site.FieldSources = FieldSources{"/mcpEnabled": {Source: "seed"}}
	repo.Save(site)
	site = target.Update(site.Id, PortalSiteUpdate{McpEnabled: new(false)})
	require.False(t, site.McpEnabled)
	require.Equal(t, "hub", site.FieldSources["/mcpEnabled"].Override)
	web := testUserSite()
	web.McpEnabled = true
	require.Panics(t, func() { target.Validate(web) })
}

func TestPortalSiteValidateWithoutStorage(t *testing.T) {
	target := &PortalSiteCore{}
	got := target.Validate(testUserSite())
	require.Equal(t, PortalCorsModeSameDomain, got.Cors.Mode)
	for name, mutate := range map[string]func(*PortalSite){
		"name":        func(s *PortalSite) { s.Name = " " },
		"type":        func(s *PortalSite) { s.Type = "unknown" },
		"actor":       func(s *PortalSite) { s.ActorSkelName = "" },
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

func TestWebTimeoutValidation(t *testing.T) {
	target := new(PortalSiteCore)
	site := PortalSite{Name: "files", Type: PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", WebName: "demo.Web"}
	site.WebTimeout = &webtimeout.Config{Default: &webtimeout.Policy{Total: "15m"}}
	require.NotPanics(t, func() { target.Validate(site) })
	site.Type = PortalSiteTypeRPCGW
	require.Panics(t, func() { target.Validate(site) })
	site.WebTimeout = &webtimeout.Config{}
	require.Nil(t, target.Validate(site).WebTimeout)
	site.Type = PortalSiteTypeWEBGW
	site.WebTimeout = &webtimeout.Config{Default: &webtimeout.Policy{Total: "0s"}}
	require.Panics(t, func() { target.Validate(site) })
}

func TestWebTimeoutUpdateReplacesPreservesAndClears(t *testing.T) {
	target := newPortalSiteCoreForTest(&portalSiteRepoSpy{})
	policy := &webtimeout.Config{Default: &webtimeout.Policy{Total: "15m"}, Paths: []webtimeout.Path{{Path: "/short", Total: "5s"}}}
	site := target.Create(PortalSiteCreation{Name: "files", Type: PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", WebName: "demo.Web", WebTimeout: policy})
	require.Equal(t, policy, target.Update(site.Id, PortalSiteUpdate{Name: new("renamed")}).WebTimeout)
	replacement := &webtimeout.Config{Default: &webtimeout.Policy{Total: "1m"}}
	require.Equal(t, replacement, target.Update(site.Id, PortalSiteUpdate{WebTimeout: replacement}).WebTimeout)
	require.Nil(t, target.Update(site.Id, PortalSiteUpdate{WebTimeout: &webtimeout.Config{}}).WebTimeout)
}
