package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/util/vslice"
)

type entryRuleRepoSpy struct {
	calls []string
	rules map[int]*PortalRule
}

func (s *entryRuleRepoSpy) List() []*PortalRule {
	s.calls = append(s.calls, "List")
	rules := make([]*PortalRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	return vslice.SortBy(rules, func(a *PortalRule, b *PortalRule) bool {
		return a.Id < b.Id
	})
}

func (s *entryRuleRepoSpy) GetById(id int) (*PortalRule, bool) {
	s.calls = append(s.calls, "GetById")
	rule, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	value := *rule
	return &value, true
}

func (s *entryRuleRepoSpy) GetByName(name string) (*PortalRule, bool) {
	s.calls = append(s.calls, "GetByName:"+name)
	for _, rule := range s.rules {
		if rule.Name == name {
			value := *rule
			return &value, true
		}
	}
	return nil, false
}

func (s *entryRuleRepoSpy) Save(rule *PortalRule) {
	s.calls = append(s.calls, "Save")
	if s.rules == nil {
		s.rules = map[int]*PortalRule{}
	}
	value := *rule
	s.rules[value.Id] = &value
}

func (s *entryRuleRepoSpy) Remove(id int) bool {
	s.calls = append(s.calls, "Remove")
	if _, ok := s.rules[id]; !ok {
		return false
	}
	delete(s.rules, id)
	return true
}

type portalEntryRepoSpy struct {
	calls   []string
	nextId  int
	entries map[int]*PortalEntry
}

func newPortalEntryRepoSpy(entries ...*PortalEntry) *portalEntryRepoSpy {
	spy := &portalEntryRepoSpy{
		nextId:  1,
		entries: map[int]*PortalEntry{},
	}
	for _, entry := range entries {
		spy.Save(entry)
		if entry.Id >= spy.nextId {
			spy.nextId = entry.Id + 1
		}
	}
	spy.calls = nil
	return spy
}

func (s *portalEntryRepoSpy) List() []*PortalEntry {
	s.calls = append(s.calls, "List")
	entries := make([]*PortalEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		value := *entry
		entries = append(entries, &value)
	}
	return vslice.SortBy(entries, func(a *PortalEntry, b *PortalEntry) bool {
		return a.Id < b.Id
	})
}

func (s *portalEntryRepoSpy) GetById(id int) (*PortalEntry, bool) {
	s.calls = append(s.calls, "GetById:"+fmt.Sprint(id))
	entry, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	value := *entry
	return &value, true
}

func (s *portalEntryRepoSpy) GetByAccess(scheme string, host string, port int) (*PortalEntry, bool) {
	s.calls = append(s.calls, fmt.Sprintf("GetByAccess:%s:%s:%d", scheme, host, port))
	for _, entry := range s.entries {
		if entry.BuiltIn {
			continue
		}
		if entry.Scheme == scheme && entry.Host == host && entry.Port == port {
			value := *entry
			return &value, true
		}
	}
	return nil, false
}

func (s *portalEntryRepoSpy) GetBuiltIn() (*PortalEntry, bool) {
	s.calls = append(s.calls, "GetBuiltIn")
	for _, entry := range s.entries {
		if entry.BuiltIn {
			value := *entry
			return &value, true
		}
	}
	return nil, false
}

func (s *portalEntryRepoSpy) Save(entry *PortalEntry) {
	s.calls = append(s.calls, "Save")
	value := *entry
	if value.Id == 0 {
		value.Id = s.nextId
		s.nextId++
	}
	value.Name = PortalEntryName(value.Scheme, value.Host, value.Port)
	s.entries[value.Id] = &value
	entry.Id = value.Id
	entry.Name = value.Name
}

func (s *portalEntryRepoSpy) Remove(id int) bool {
	s.calls = append(s.calls, "Remove:"+fmt.Sprint(id))
	if _, ok := s.entries[id]; !ok {
		return false
	}
	delete(s.entries, id)
	return true
}

// newPortalEntryCoreForTest builds an entry core with the repositories Hub
// injects, so tests only choose the repositories they exercise.
func newPortalEntryCoreForTest(ruleRepo PortalRuleRepo, entryRepo PortalEntryRepo, siteRepo PortalSiteRepo) *PortalEntryCore {
	if siteRepo == nil {
		siteRepo = &portalSiteRepoSpy{}
	}
	return &PortalEntryCore{
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  ruleRepo,
		PortalSiteRepo:  siteRepo,
	}
}

func TestPortalEntryCoreListGroupsRulesByEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Scheme: "https", Port: 443},
		&PortalEntry{Id: 2, Scheme: "http", Port: 8080},
		&PortalEntry{Id: 3, Scheme: "http", Host: "demo.local", Port: 8080},
		&PortalEntry{Id: 4, Scheme: "http", Port: 7099, BuiltIn: true},
		&PortalEntry{Id: 5, Scheme: "http", Port: 9090},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "admin", EntryId: 4, MatchPathPrefix: "/admin", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "admin-site", BuiltIn: true},
		2: {Id: 2, Name: "home", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "home-site"},
		3: {Id: 3, Name: "api", EntryId: 2, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypePermanentRedirect, RouteRedirectionPattern: "https://demo.local"},
		4: {Id: 4, Name: "ignored", EntryId: 2, MatchPathPrefix: "/", RouteType: "UNSUPPORTED"},
		5: {Id: 5, Name: "hosted", EntryId: 3, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "home-site"},
	}}
	siteRepo := &portalSiteRepoSpy{
		entries: map[int]*PortalSite{
			1: {Id: 1, Name: "admin-site"},
			2: {Id: 2, Name: "home-site"},
		},
	}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, siteRepo)

	entries := core.List()

	require.Len(t, entries, 3)
	assert.Equal(t, "https:443", entries[0].Name)
	assert.Equal(t, "https", entries[0].Scheme)
	assert.Equal(t, "", entries[0].Host)
	assert.Equal(t, 443, entries[0].Port)
	require.Len(t, entries[0].Rules, 1)
	assert.Equal(t, "home", entries[0].Rules[0].Rule.Name)
	assert.Equal(t, 2, entries[0].Rules[0].Site.Id)

	assert.Equal(t, "http:8080", entries[1].Name)
	require.Len(t, entries[1].Rules, 1)
	assert.Equal(t, "api", entries[1].Rules[0].Rule.Name)

	assert.Equal(t, "http:demo.local:8080", entries[2].Name)
	require.Len(t, entries[2].Rules, 1)
	assert.Equal(t, "hosted", entries[2].Rules[0].Rule.Name)
}

func TestPortalEntryCoreListSkipsEntriesWithoutRules(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "https", Port: 443})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	assert.Empty(t, core.List())
}

func TestPortalEntryCoreListRejectsUnknownScheme(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "tcp", Port: 9000})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "tcp", EntryId: 1, RouteType: PortalRuleRouteTypeSite},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	panicValue := capturePanic(func() {
		core.List()
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalEntryCoreUpdateAccessSavesEntryAndRepublishesRules(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Scheme: "http", Port: 7088},
		&PortalEntry{Id: 2, Scheme: "http", Host: "demo.local", Port: 7088},
		&PortalEntry{Id: 5, Scheme: "http", Port: 7099, BuiltIn: true},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "rpc-site"},
		3: {Id: 3, Name: "other-host", EntryId: 2, MatchScheme: "http", MatchHost: "demo.local", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		4: {Id: 4, Name: "redirect", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/old", RouteType: PortalRuleRouteTypePermanentRedirect, RouteRedirectionPattern: "https://demo.local"},
		5: {Id: 5, Name: "vine", EntryId: 5, MatchScheme: "http", MatchPort: 7099, MatchPathPrefix: "/vine", RouteType: PortalRuleRouteTypeSite, BuiltIn: true},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{
		Scheme: "https",
		Host:   "app.example.com",
		Port:   8443,
	})

	// The access is one stored row: Hub no longer rewrites it per rule.
	assert.Equal(t, 1, entry.Id)
	assert.Equal(t, "https:app.example.com:8443", entry.Name)
	assert.Equal(t, "https", entry.Scheme)
	assert.Equal(t, "app.example.com", entry.Host)
	assert.Equal(t, 8443, entry.Port)
	require.Len(t, entry.Rules, 3)
	assert.Equal(t, "https", entryRepo.entries[1].Scheme)
	assert.Equal(t, "app.example.com", entryRepo.entries[1].Host)
	assert.Equal(t, 8443, entryRepo.entries[1].Port)
	assert.Equal(t, []string{
		"GetByAccess:http::7088",
		"GetByAccess:https:app.example.com:8443",
		"Save",
	}, entryRepo.calls)
	// The access change republishes the rules of the entry after checking that
	// none of them matches the same request as another rule.
	assert.Equal(t, []string{"List", "List", "Save", "Save", "Save", "List"}, ruleRepo.calls)
	// The members of the entry are republished with the access they now resolve.
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	assert.Equal(t, "https", ruleRepo.rules[1].MatchScheme)
	assert.Equal(t, "app.example.com", ruleRepo.rules[1].MatchHost)
	assert.Equal(t, 8443, ruleRepo.rules[1].MatchPort)
	assert.Equal(t, "https", ruleRepo.rules[2].MatchScheme)
	assert.Equal(t, "https", ruleRepo.rules[4].MatchScheme)
	// Another entry and the built-in entry keep their own access.
	assert.Equal(t, "http", ruleRepo.rules[3].MatchScheme)
	assert.Equal(t, "demo.local", ruleRepo.rules[3].MatchHost)
	assert.Equal(t, "http", ruleRepo.rules[5].MatchScheme)
	assert.Equal(t, 7099, ruleRepo.rules[5].MatchPort)
}

func TestPortalEntryCoreUpdateAccessMergesIntoExistingEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Scheme: "http", Port: 7088},
		&PortalEntry{Id: 2, Scheme: "https", Host: "demo.local", Port: 8443},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 2, MatchScheme: "https", MatchHost: "demo.local", MatchPort: 8443, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "rpc-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{
		Scheme: "https",
		Host:   "demo.local",
		Port:   8443,
	})

	assert.Equal(t, 2, entry.Id)
	require.Len(t, entry.Rules, 2)
	assert.Equal(t, "api", entry.Rules[0].Rule.Name)
	assert.Equal(t, "web", entry.Rules[1].Rule.Name)
	assert.NotContains(t, entryRepo.entries, 1)
	assert.Equal(t, 2, ruleRepo.rules[1].EntryId)
	assert.Equal(t, "https", ruleRepo.rules[1].MatchScheme)
	assert.Equal(t, "demo.local", ruleRepo.rules[1].MatchHost)
}

func TestPortalEntryCoreUpdateAccessRejectsMissingEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 7088})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	panicValue := capturePanic(func() {
		core.UpdateAccess("http", "missing.local", 7088, PortalEntryAccessUpdate{Scheme: "http", Port: 8080})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Empty(t, entryRepo.calls[1:])
}

func TestPortalEntryCoreUpdateAccessNormalizesLookup(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 2, Scheme: "http", Port: 80})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 2, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.UpdateAccess(" HTTP ", " ", 0, PortalEntryAccessUpdate{Scheme: "https", Host: "demo.local", Port: 0})

	assert.Equal(t, 2, entry.Id)
	assert.Equal(t, "https:demo.local:443", entry.Name)
	assert.Equal(t, []string{"GetByAccess:http::80", "GetByAccess:https:demo.local:443", "Save"}, entryRepo.calls)
}

func TestPortalEntryCoreUpdateAccessRejectsRequestTakenByAnotherEntry(t *testing.T) {
	// Changing the access of the entry would make its rule match the same request
	// as a rule of the built-in Dashboard entry.
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Scheme: "http", Port: 7088},
		&PortalEntry{Id: 2, Scheme: "http", Port: 7099, BuiltIn: true},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: DashboardWebRuleName, EntryId: 2, MatchScheme: "http", MatchPort: 7099, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, BuiltIn: true},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	require.PanicsWithError(t,
		`portal rule "vine.hub.dashboard-web" already matches http://*:7099/ type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{Scheme: "http", Port: 7099})
		})
	// Neither the entry nor its rules changed.
	assert.Equal(t, 7088, entryRepo.entries[1].Port)
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	assert.Equal(t, 7088, ruleRepo.rules[1].MatchPort)
}

func TestPortalEntryCoreUpdateAccessRejectsMergedPathClash(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Scheme: "http", Port: 7088},
		&PortalEntry{Id: 2, Scheme: "http", Host: "demo.local", Port: 8080},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 2, MatchScheme: "http", MatchHost: "demo.local", MatchPort: 8080, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	require.PanicsWithError(t,
		`portal rule "api" already matches http://demo.local:8080/ type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{Scheme: "http", Host: "demo.local", Port: 8080})
		})
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	// The rejected update changes neither entry: the rules stay where they are.
	assert.Contains(t, entryRepo.entries, 1)
	assert.Contains(t, entryRepo.entries, 2)
}

func TestPortalEntryCoreUpdateAccessKeepsEntryOnUnchangedAccess(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 7088})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 7088, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.UpdateAccess("http", "", 7088, PortalEntryAccessUpdate{Scheme: "http", Port: 7088})

	// Hub writes nothing when the access already matches.
	assert.Equal(t, 1, entry.Id)
	assert.Equal(t, "http:7088", entry.Name)
	require.Len(t, entry.Rules, 1)
	assert.Equal(t, "web", entry.Rules[0].Rule.Name)
	assert.Equal(t, []string{"GetByAccess:http::7088", "GetByAccess:http::7088"}, entryRepo.calls)
}

func TestPortalEntryCoreEnsureAccessNormalizesAndReusesEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 3, Scheme: "http", Port: 80})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	entry := core.EnsureAccess(" HTTP ", " ", 0)

	// " HTTP " and an unset port address the entry Portal already serves.
	assert.Equal(t, 3, entry.Id)
	assert.Empty(t, entryRepo.calls[1:])

	created := core.EnsureAccess("HTTPS", " demo.local ", 0)
	assert.Equal(t, "https", created.Scheme)
	assert.Equal(t, "demo.local", created.Host)
	assert.Equal(t, 443, created.Port)
	assert.Equal(t, "https:demo.local:443", created.Name)
}

func TestPortalEntryCoreEnsureBuiltInAccessKeepsConfiguredAccess(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 7, Scheme: "https", Host: "hub.example.com", Port: 8443, BuiltIn: true})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	kept := core.EnsureBuiltInAccess("http", "", 7099, false)
	assert.Equal(t, 7, kept.Id)
	assert.Equal(t, "https", kept.Scheme)
	assert.Equal(t, "hub.example.com", kept.Host)

	refreshed := core.EnsureBuiltInAccess("http", "", 7099, true)
	assert.Equal(t, 7, refreshed.Id)
	assert.Equal(t, "http", refreshed.Scheme)
	assert.Equal(t, "", refreshed.Host)
	assert.Equal(t, 7099, refreshed.Port)
	assert.Equal(t, "http:7099", refreshed.Name)
	assert.True(t, refreshed.BuiltIn)
	assert.Len(t, entryRepo.entries, 1)
}

func TestPortalEntryCoreGetRejectsMissingEntry(t *testing.T) {
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, newPortalEntryRepoSpy(), nil)

	panicValue := capturePanic(func() {
		core.Get(9)
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}
