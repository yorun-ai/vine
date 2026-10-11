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
	removed []int
}

func newPortalEntryRepoSpy(entries ...*PortalEntry) *portalEntryRepoSpy {
	spy := &portalEntryRepoSpy{
		nextId:  1,
		entries: map[int]*PortalEntry{},
	}
	for _, entry := range entries {
		if entry.Name == "" {
			entry.Name = fmt.Sprintf("%s:%d", entry.Accesses()[0].Scheme, entry.Accesses()[0].Port)
			if entry.Http.Host != "" {
				entry.Name = fmt.Sprintf("%s:%s:%d", entry.Accesses()[0].Scheme, entry.Http.Host, entry.Accesses()[0].Port)
			}
		}
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

func (s *portalEntryRepoSpy) GetByName(name string) (*PortalEntry, bool) {
	s.calls = append(s.calls, "GetByName:"+name)
	for _, entry := range s.entries {
		if entry.Name == name {
			value := *entry
			return &value, true
		}
	}
	return nil, false
}

func (s *portalEntryRepoSpy) Save(entry *PortalEntry) {
	s.calls = append(s.calls, "Save")
	value := *entry
	if value.Protocol == "http" {
		value = NormalizePortalEntry(value)
	}
	if value.Id == 0 {
		value.Id = s.nextId
		s.nextId++
	}
	s.entries[value.Id] = &value
	entry.Id = value.Id
}

func (s *portalEntryRepoSpy) Remove(id int) bool {
	s.calls = append(s.calls, "Remove:"+fmt.Sprint(id))
	if _, ok := s.entries[id]; !ok {
		return false
	}
	s.removed = append(s.removed, id)
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

func TestPortalEntryCoreSaveReplacesByName(t *testing.T) {
	// A seed applies entries before rules: an entry keeps its identity, and
	// replacing it republishes the rules it routes.
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Name: "web", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "app", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	saved := core.Save(PortalEntry{Name: "web", Protocol: "http", Http: &PortalEntryHTTP{Host: "app.example.com", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8443}})

	assert.Equal(t, 1, saved.Id)
	assert.Equal(t, "web", saved.Name)
	assert.Equal(t, 8443, saved.Accesses()[0].Port)
	// The rule follows the entry it belongs to.
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)

	// A name is required, and two entries never serve one access.
	require.PanicsWithError(t, "portal entry name is required type=APPLICATION code=OPERATION_FAILED",
		func() {
			core.Save(PortalEntry{Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}})
		})
	require.PanicsWithError(t, `portal entry "web" already serves https:app.example.com:8443 type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Save(PortalEntry{Name: "api", Protocol: "http", Http: &PortalEntryHTTP{Host: "app.example.com", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8443}})
		})
}

func TestPortalEntryCoreSaveRejectsAccessTakenByAnotherEntry(t *testing.T) {
	// Replacing an entry moves the access of every rule it routes. Hub rejects a
	// move onto an access another entry already serves.
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Name: "web", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8088}},
		&PortalEntry{Id: 2, Name: "vine", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7099}},
		&PortalEntry{Id: 3, Name: "api", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8099}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "vine", EntryId: 2, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	require.PanicsWithError(t,
		`portal entry "vine" already serves http:7099 type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Save(PortalEntry{Name: "web", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7099}})
		})
	assert.Equal(t, 8088, entryRepo.entries[1].Accesses()[0].Port)

	// Two entries never serve one access.
	require.PanicsWithError(t,
		`portal entry "api" already serves http:8099 type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Save(PortalEntry{Name: "other", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8099}})
		})
}

func TestPortalEntryCoreListGroupsRulesByEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: false, HttpsEnabled: true, HttpsPort: 443}},
		&PortalEntry{Id: 2, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}},
		&PortalEntry{Id: 3, Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}},
		&PortalEntry{Id: 5, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 9090}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
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

	// The entry that routes nothing is listed as well.
	require.Len(t, entries, 4)
	assert.Equal(t, "https:443", entries[0].Name)
	assert.Equal(t, "https", entries[0].Accesses()[0].Scheme)
	assert.Equal(t, "", entries[0].Http.Host)
	assert.Equal(t, 443, entries[0].Accesses()[0].Port)
	require.Len(t, entries[0].Rules, 1)
	assert.Equal(t, "home", entries[0].Rules[0].Rule.Name)
	assert.Equal(t, 2, entries[0].Rules[0].Site.Id)

	assert.Equal(t, "http:8080", entries[1].Name)
	require.Len(t, entries[1].Rules, 1)
	assert.Equal(t, "api", entries[1].Rules[0].Rule.Name)

	assert.Equal(t, "http:demo.local:8080", entries[2].Name)
	require.Len(t, entries[2].Rules, 1)
	assert.Equal(t, "hosted", entries[2].Rules[0].Rule.Name)

	assert.Equal(t, "http:9090", entries[3].Name)
	assert.Empty(t, entries[3].Rules)
}

func TestPortalEntryCoreListIncludesEntriesWithoutRules(t *testing.T) {
	// An operator creates an entry before the rules that use it, so the entry
	// list shows it while it routes nothing.
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: false, HttpsEnabled: true, HttpsPort: 443}})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	entries := core.List()

	require.Len(t, entries, 1)
	assert.Equal(t, "https:443", entries[0].Name)
	assert.Empty(t, entries[0].Rules)
}

func TestPortalEntryCoreCreate(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy()
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	entry := core.Create(PortalEntryCreation{Name: " web ", Protocol: "http", Http: &PortalEntryHTTP{Host: " demo.local ", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 0}})

	// The operator names an entry; the access is normalized as Hub stores it.
	assert.Equal(t, "web", entry.Name)
	assert.Equal(t, "https", entry.Accesses()[0].Scheme)
	assert.Equal(t, "demo.local", entry.Http.Host)
	assert.Equal(t, 443, entry.Accesses()[0].Port)
	assert.Empty(t, entry.Rules)
	require.Len(t, entryRepo.entries, 1)
	assert.Equal(t, []string{"GetByName:web", "List", "List", "Save"}, entryRepo.calls)

	// The name identifies one entry, and an access serves one user entry.
	require.PanicsWithError(t,
		`portal entry "web" already exists type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Create(PortalEntryCreation{Name: "web", Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 443}})
		})
	require.PanicsWithError(t,
		`portal entry "web" already serves https:demo.local:443 type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Create(PortalEntryCreation{Name: "api", Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 443}})
		})
	panicValue := capturePanic(func() {
		core.Create(PortalEntryCreation{Name: " "})
	})
	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Len(t, entryRepo.entries, 1)
}

func TestPortalEntryCoreRemove(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	// An entry keeps its rules: Hub refuses to leave them without an access.
	require.PanicsWithError(t,
		`portal entry "http:7088" still routes 1 rules; remove them first type=APPLICATION code=OPERATION_FAILED`,
		func() { core.Remove(1) })
	assert.Contains(t, entryRepo.entries, 1)

	// An entry that routes nothing is removed.
	core.Create(PortalEntryCreation{Name: "idle", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}})
	created, ok := entryRepo.GetByName("idle")
	require.True(t, ok)
	assert.Equal(t, "idle", created.Name)
	core.Remove(created.Id)
	_, ok = entryRepo.GetByName("idle")
	assert.False(t, ok)
	assert.Contains(t, entryRepo.removed, created.Id)
}

func TestPortalEntryCoreListRejectsUnknownScheme(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Name: "invalid", Protocol: "tcp", Http: &PortalEntryHTTP{HttpEnabled: "tcp" == "http", HttpsEnabled: "tcp" == "https", HttpPort: 9000, HttpsPort: 9000}})
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

func TestPortalEntryCoreUpdateSavesEntryAndRepublishesRules(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
		&PortalEntry{Id: 2, Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 1, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "rpc-site"},
		3: {Id: 3, Name: "other-host", EntryId: 2, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		4: {Id: 4, Name: "redirect", EntryId: 1, MatchPathPrefix: "/old", RouteType: PortalRuleRouteTypePermanentRedirect, RouteRedirectionPattern: "https://demo.local"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.Update(1, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{Host: new("app.example.com"), HttpEnabled: new(false), HttpsEnabled: new(true), HttpsPort: new(8443)}})

	// The access is one stored row: Hub no longer rewrites it per rule.
	assert.Equal(t, 1, entry.Id)
	// The name stays the one the entry already carries.
	assert.Equal(t, "http:7088", entry.Name)
	assert.Equal(t, "https", entry.Accesses()[0].Scheme)
	assert.Equal(t, "app.example.com", entry.Http.Host)
	assert.Equal(t, 8443, entry.Accesses()[0].Port)
	require.Len(t, entry.Rules, 3)
	assert.Equal(t, "https", entryRepo.entries[1].Accesses()[0].Scheme)
	assert.Equal(t, "app.example.com", entryRepo.entries[1].Http.Host)
	assert.Equal(t, 8443, entryRepo.entries[1].Accesses()[0].Port)
	assert.Equal(t, []string{
		"GetById:1",
		"List",
		"Save",
	}, entryRepo.calls)
	// The access change republishes the rules of the entry.
	assert.Equal(t, []string{"List", "Save", "Save", "Save", "List"}, ruleRepo.calls)
	// The members of the entry stay where they are: the entry owns the access
	// they resolve through.
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	assert.Equal(t, 1, ruleRepo.rules[2].EntryId)
	assert.Equal(t, 1, ruleRepo.rules[4].EntryId)
	// Another entry keeps its own rules and access.
	assert.Equal(t, 2, ruleRepo.rules[3].EntryId)
	assert.Equal(t, "http", entryRepo.entries[2].Accesses()[0].Scheme)
	assert.Equal(t, "demo.local", entryRepo.entries[2].Http.Host)
}

func TestPortalEntryCoreUpdateRejectsExistingAccess(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
		&PortalEntry{Id: 2, Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8443}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 2, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "rpc-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	require.Panics(t, func() {
		core.Update(1, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{Host: new("demo.local"), HttpEnabled: new(false), HttpPort: new(80), HttpsEnabled: new(true), HttpsPort: new(8443)}})
	})
	assert.Len(t, entryRepo.entries, 2)
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	assert.Equal(t, 2, ruleRepo.rules[2].EntryId)
	assert.Empty(t, entryRepo.removed)

}

func TestPortalEntryCoreUpdateRejectsMissingEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}})
	core := newPortalEntryCoreForTest(&entryRuleRepoSpy{}, entryRepo, nil)

	panicValue := capturePanic(func() {
		core.Update(99, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{HttpEnabled: new(true), HttpsEnabled: new(false), HttpPort: new(8080)}})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetById:99"}, entryRepo.calls)
}

func TestPortalEntryCoreUpdateNormalizesPatch(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 2, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 80}})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 2, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.Update(2, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{Host: new(" demo.local "), HttpEnabled: new(false), HttpsEnabled: new(true), HttpPort: new(0)}})

	assert.Equal(t, 2, entry.Id)
	assert.Equal(t, "http:80", entry.Name)
	assert.Equal(t, "https", entry.Accesses()[0].Scheme)
	assert.Equal(t, "demo.local", entry.Http.Host)
	assert.Equal(t, 443, entry.Accesses()[0].Port)
	assert.Equal(t, []string{
		"GetById:2",
		"List",
		"Save",
	}, entryRepo.calls)
	assert.Equal(t, []string{"List", "Save", "List"}, ruleRepo.calls)
}

func TestPortalEntryCoreUpdateKeepsEntryOnUnchangedPatch(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}})
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.Update(1, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{HttpEnabled: new(true), HttpsEnabled: new(false), HttpPort: new(7088)}})

	// Hub writes nothing when the patch changes nothing.
	assert.Equal(t, 1, entry.Id)
	assert.Equal(t, "http:7088", entry.Name)
	require.Len(t, entry.Rules, 1)
	assert.Equal(t, "web", entry.Rules[0].Rule.Name)
	assert.Equal(t, []string{"GetById:1"}, entryRepo.calls)
	assert.Equal(t, []string{"List"}, ruleRepo.calls)
}

func TestPortalEntryCoreUpdateRenamesEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 1, Name: "web", Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
		&PortalEntry{Id: 2, Name: "api", Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 7088}},
	)
	ruleRepo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalEntryCoreForTest(ruleRepo, entryRepo, nil)

	entry := core.Update(1, PortalEntryUpdate{Name: new("web-console")})

	// A rename keeps the access: Portal reaches the entry through its rules, so
	// the rules stay where they are and Hub republishes none of them.
	assert.Equal(t, 1, entry.Id)
	assert.Equal(t, "web-console", entry.Name)
	assert.Equal(t, "http", entry.Accesses()[0].Scheme)
	assert.Equal(t, 7088, entry.Accesses()[0].Port)
	require.Len(t, entry.Rules, 1)
	assert.Equal(t, 1, ruleRepo.rules[1].EntryId)
	assert.Equal(t, []string{"GetById:1", "GetByName:web-console", "Save"}, entryRepo.calls)
	assert.Equal(t, []string{"List"}, ruleRepo.calls)

	// Two entries never share a label.
	require.PanicsWithError(t, `portal entry "api" already exists type=APPLICATION code=OPERATION_FAILED`, func() {
		core.Update(1, PortalEntryUpdate{Name: new("api")})
	})
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

func TestPortalEntryWildcardHostValidation(t *testing.T) {
	for _, host := range []string{"*.example.com", "*.EXAMPLE.com", "*.dev"} {
		assert.True(t, portalEntryHostAccepted(host), host)
	}
	for _, host := range []string{"*", "*example.com", "foo*.example.com", "*.*.example.com", "*.", "*.example..com", "*.127.0.0.1", "*.example.com:80", "*.-bad.com"} {
		assert.False(t, portalEntryHostAccepted(host), host)
	}
}

func TestPortalEntryWildcardChangeRejectsNonWebRules(t *testing.T) {
	for _, save := range []bool{false, true} {
		repo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Name: "entry", Protocol: "http", Http: &PortalEntryHTTP{Host: "api.example.com", HttpEnabled: true, HttpsEnabled: false, HttpPort: 80}})
		rules := &entryRuleRepoSpy{rules: map[int]*PortalRule{1: {Id: 1, Name: "rpc", EntryId: 1, RouteType: "SITE", RouteSiteName: "rpc"}}}
		sites := &portalSiteRepoSpy{entries: map[int]*PortalSite{1: {Id: 1, Name: "rpc", Type: PortalSiteTypeRPCGW}}}
		target := newPortalEntryCoreForTest(rules, repo, sites)
		assert.Panics(t, func() {
			if save {
				target.Save(PortalEntry{Name: "entry", Protocol: "http", Http: &PortalEntryHTTP{Host: "*.example.com", HttpEnabled: true, HttpsEnabled: false, HttpPort: 80}})
			} else {
				target.Update(1, PortalEntryUpdate{Http: &PortalEntryHTTPUpdate{Host: new("*.example.com")}})
			}
		})
		entry, _ := repo.GetById(1)
		assert.Equal(t, "api.example.com", entry.Http.Host)
	}
}

func TestPortalEntryListenIPsCreateUpdateAndReset(t *testing.T) {
	repo := newPortalEntryRepoSpy()
	target := &PortalEntryCore{PortalEntryRepo: repo, PortalRuleRepo: &entryRuleRepoSpy{}}
	created := target.Create(PortalEntryCreation{Name: "local", ListenIPs: []string{"::1", "127.0.0.1", "::ffff:127.0.0.1"}, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}})
	assert.Equal(t, []string{"127.0.0.1", "::1"}, created.ListenIPs)
	updated := target.Update(created.Id, PortalEntryUpdate{ListenIPs: new([]string{"127.0.0.2"})})
	assert.Equal(t, []string{"127.0.0.2"}, updated.ListenIPs)
	unchanged := target.Update(created.Id, PortalEntryUpdate{Name: new("renamed")})
	assert.Equal(t, updated.ListenIPs, unchanged.ListenIPs)
	reset := target.Update(created.Id, PortalEntryUpdate{ListenIPs: new([]string{})})
	assert.Empty(t, reset.ListenIPs)
}

func TestPortalEntryListenIPsRejectConflictsAndShareListeners(t *testing.T) {
	repo := newPortalEntryRepoSpy()
	target := &PortalEntryCore{PortalEntryRepo: repo, PortalRuleRepo: &entryRuleRepoSpy{}}
	target.Create(PortalEntryCreation{Name: "one", ListenIPs: []string{"127.0.0.1"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "one.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}})
	target.Create(PortalEntryCreation{Name: "two", ListenIPs: []string{"127.0.0.1"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "two.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}})
	target.Create(PortalEntryCreation{Name: "v6", ListenIPs: []string{"::1"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "v6.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8080}})
	for _, creation := range []PortalEntryCreation{
		{Name: "wildcard", ListenIPs: []string{"0.0.0.0"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "wildcard.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}},
		{Name: "default", Protocol: "http", Http: &PortalEntryHTTP{Host: "default.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}},
		{Name: "tls", ListenIPs: []string{"127.0.0.1"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "tls.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8080}},
		{Name: "bad", ListenIPs: []string{"localhost"}, Protocol: "http", Http: &PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8090}},
	} {
		require.Panics(t, func() { target.Create(creation) })
	}
	assert.Len(t, repo.List(), 3)
	disabled := target.Create(PortalEntryCreation{Name: "disabled", ListenIPs: []string{"127.0.0.1"}, Enabled: new(false), Protocol: "http", Http: &PortalEntryHTTP{Host: "disabled.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8080}})
	require.Panics(t, func() { target.Update(disabled.Id, PortalEntryUpdate{Enabled: new(true)}) })
	assert.False(t, target.Get(disabled.Id).Enabled)
}
