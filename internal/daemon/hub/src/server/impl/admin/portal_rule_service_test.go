package admin

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/descriptor"
	"go.yorun.ai/vine/util/vslice"
	"gorm.io/gorm"
)

func TestPortalRuleAddressesUseOnlyEffectiveTransports(t *testing.T) {
	entry := core.NormalizePortalEntry(core.PortalEntry{Protocol: "http", Http: new(core.PortalEntryHTTP{Host: "example.com", Http: "DISABLED"})})
	entry.EffectiveHTTP = new(entry.Http.Effective(false))
	require.Empty(t, toServerPortalRule(new(core.PortalRule), &entry, nil, nil).EntryAddresses)
	entry.EffectiveHTTP = new(entry.Http.Effective(true))
	require.Equal(t, []string{"https://example.com:443"}, toServerPortalRule(new(core.PortalRule), &entry, nil, nil).EntryAddresses)
}

func TestPortalRuleListBatchesRelatedRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rules.db")))
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })

	rules := model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](db)}
	certs := model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](db)}
	sites := model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](db)}
	rules.EnsureSchema()
	certs.EnsureSchema()
	sites.EnsureSchema()
	descriptors := new(descriptor.DescriptorRepo)
	service := PortalRuleApiServiceServerImpl{PortalRuleListRepo: new(repo.PortalRuleRepo{
		Dao:            &rules,
		PortalSiteRepo: new(repo.PortalSiteRepo{DescriptorRepo: descriptors}),
	})}

	queries := map[string]int{}
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("test:count", func(tx *gorm.DB) {
		queries[tx.Statement.Table]++
	}))
	require.Empty(t, service.List())
	require.Equal(t, map[string]int{"portal_rule": 1}, queries)

	entry := new(model.PortalEntry{Name: "public", Protocol: "http", HTTPConfig: `{"host":"example.com"}`, ListenIPs: "[]", Enabled: true})
	require.NoError(t, db.Create(entry).Error)
	for i := range 32 {
		require.NoError(t, db.Create(new(model.PortalRule{
			Name:                    fmt.Sprintf("redirect-%02d", i),
			EntryId:                 entry.Id,
			MatchPathPrefix:         fmt.Sprintf("/r/%d", i),
			RouteType:               core.PortalRuleRouteTypeTemporaryRedirect,
			RouteRedirectionPattern: "/docs",
			Enabled:                 true,
		})).Error)
	}
	clear(queries)
	items := service.List()
	require.Len(t, items, 32)
	require.Equal(t, "public", items[0].EntryName)
	require.Equal(t, []string{"http://example.com:80"}, items[0].EntryAddresses)
	require.Equal(t, "/r/0", items[0].ResolvedMatchPathPrefix)
	require.Equal(t, map[string]int{"portal_rule": 1, "portal_entry": 1, "portal_cert": 1}, queries)

	descriptors.SaveDomainDescriptors("app", "instance", []*skeldesc.Domain{{
		Name:      "demo",
		Hash:      "domain-hash",
		Generated: new(skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"}),
		Webs:      []*skeldesc.Web{{Name: "Docs", SkelName: "demo.Docs", Hash: "web-hash", MountPath: "/docs", AuthMode: skeldesc.AuthModeAnonymous}},
	}})
	require.NoError(t, db.Create(new(model.PortalCert{Name: "tls", Domains: `["example.com"]`, Enabled: true})).Error)
	site := new(model.PortalSite{Name: "docs", Type: string(core.PortalSiteTypeWEBGW), WebName: "demo.Docs", Enabled: true})
	require.NoError(t, db.Create(site).Error)
	for _, name := range []string{"docs", "missing"} {
		require.NoError(t, db.Create(new(model.PortalRule{
			Name:            "site-" + name,
			EntryId:         entry.Id,
			RouteType:       core.PortalRuleRouteTypeSite,
			RouteSiteName:   name,
			MatchPathPrefix: "/original",
			RoutePathPrefix: "/target",
		})).Error)
	}
	clear(queries)
	items = service.List()
	require.Len(t, items, 34)
	require.Equal(t, []string{"http://example.com:80", "https://example.com:443"}, items[32].EntryAddresses)
	require.Equal(t, "/docs", items[32].ResolvedMatchPathPrefix)
	require.Equal(t, "/docs", items[32].ResolvedRoutePathPrefix)
	require.Equal(t, "missing", items[33].RouteSiteName)
	require.Equal(t, "/original", items[33].ResolvedMatchPathPrefix)
	require.Equal(t, "/target", items[33].ResolvedRoutePathPrefix)
	require.Equal(t, map[string]int{"portal_rule": 1, "portal_entry": 1, "portal_cert": 1, "portal_site": 1}, queries)
}

// _PortalCertRepoSpy is a map-backed certificate repository.
type _PortalCertRepoSpy struct {
	items map[string]*core.PortalCert
}

func (r *_PortalCertRepoSpy) List() []*core.PortalCert {
	items := make([]*core.PortalCert, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalCert, b *core.PortalCert) bool { return a.Id < b.Id })
}

func (r *_PortalCertRepoSpy) GetById(id int) (*core.PortalCert, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return item, true
		}
	}
	return nil, false
}

func (r *_PortalCertRepoSpy) GetByName(name string) (*core.PortalCert, bool) {
	item, ok := r.items[name]
	return item, ok
}

func (r *_PortalCertRepoSpy) Save(cert *core.PortalCert) {
	if r.items == nil {
		r.items = map[string]*core.PortalCert{}
	}
	r.items[cert.Name] = cert
}

func (r *_PortalCertRepoSpy) Remove(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}

// newTestPortalRuleCore builds a rule core with the chosen rule repository and
// an entry repository that starts empty.
func newTestPortalRuleCore(ruleRepo core.PortalRuleRepo, entryRepos ...core.PortalEntryRepo) *core.PortalRuleCore {
	entryRepo := core.PortalEntryRepo(newTestPortalEntryRepoSpy())
	if len(entryRepos) > 0 {
		entryRepo = entryRepos[0]
	}
	siteRepo := &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}}
	return &core.PortalRuleCore{
		PortalRuleRepo: ruleRepo,
		PortalSiteRepo: siteRepo,
		PortalEntryCore: &core.PortalEntryCore{
			PortalEntryRepo: entryRepo,
			PortalRuleRepo:  ruleRepo,
			PortalSiteRepo:  siteRepo,
		},
	}
}

// newTestPortalEntryRepoSpy builds a map-backed entry repository.
func newTestPortalEntryRepoSpy(entries ...*core.PortalEntry) *_PortalEntryRepoSpy {
	spy := &_PortalEntryRepoSpy{
		nextId:  1,
		entries: map[int]*core.PortalEntry{},
	}
	for _, entry := range entries {
		spy.Save(entry)
		if entry.Id >= spy.nextId {
			spy.nextId = entry.Id + 1
		}
	}
	return spy
}

// _PortalEntryRepoSpy is a map-backed entry repository.
type _PortalEntryRepoSpy struct {
	nextId  int
	entries map[int]*core.PortalEntry
	removed []int
	certs   core.PortalCertRepo
}

func (s *_PortalEntryRepoSpy) List() []*core.PortalEntry {
	entries := make([]*core.PortalEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		value := *entry
		entries = append(entries, &value)
	}
	return vslice.SortBy(entries, func(a *core.PortalEntry, b *core.PortalEntry) bool {
		return a.Id < b.Id
	})
}

func (s *_PortalEntryRepoSpy) GetById(id int) (*core.PortalEntry, bool) {
	entry, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	value := *entry
	return &value, true
}

func (s *_PortalEntryRepoSpy) GetByName(name string) (*core.PortalEntry, bool) {
	for _, entry := range s.List() {
		if entry.Name == name {
			return entry, true
		}
	}
	return nil, false
}

func (s *_PortalEntryRepoSpy) Save(entry *core.PortalEntry) {
	var certs []*core.PortalCert
	if s.certs != nil {
		certs = s.certs.List()
	}
	entry.EffectiveHTTP = new(entry.Http.Effective(core.HasPortalCertificate(entry.Http.Host, certs)))
	value := *entry
	if value.Id == 0 {
		value.Id = s.nextId
		s.nextId++
	}
	s.entries[value.Id] = &value
	entry.Id = value.Id
	entry.Name = value.Name
}

func (s *_PortalEntryRepoSpy) Remove(id int) bool {
	if _, ok := s.entries[id]; !ok {
		return false
	}
	s.removed = append(s.removed, id)
	delete(s.entries, id)
	return true
}

type _PortalRuleRepoSpy struct {
	rules map[int]*core.PortalRule
}

func (s *_PortalRuleRepoSpy) List() []*core.PortalRule {
	rules := make([]*core.PortalRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	return vslice.SortBy(rules, func(a *core.PortalRule, b *core.PortalRule) bool {
		return a.Id < b.Id
	})
}

func (s *_PortalRuleRepoSpy) GetById(id int) (*core.PortalRule, bool) {
	rule, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	value := *rule
	return &value, true
}

func (s *_PortalRuleRepoSpy) GetByName(name string) (*core.PortalRule, bool) {
	for _, rule := range s.rules {
		if rule.Name == name {
			value := *rule
			return &value, true
		}
	}
	return nil, false
}

func (s *_PortalRuleRepoSpy) Update(id int, edit func(*core.PortalRule)) (*core.PortalRule, bool) {
	rule, ok := s.GetById(id)
	if ok {
		edit(rule)
		s.Save(rule)
	}
	return rule, ok
}

func (s *_PortalRuleRepoSpy) Save(rules ...*core.PortalRule) {
	for _, rule := range rules {
		value := *rule
		s.rules[value.Id] = &value
	}
}

func (s *_PortalRuleRepoSpy) Remove(id int) bool {
	if _, ok := s.rules[id]; !ok {
		return false
	}
	delete(s.rules, id)
	return true
}

func TestPortalRuleServiceGetReturnsFieldSources(t *testing.T) {
	repo := &_PortalRuleRepoSpy{
		rules: map[int]*core.PortalRule{
			3: {
				Id:            3,
				Name:          "demo.rule",
				EntryId:       1,
				RouteType:     core.PortalRuleRouteTypeSite,
				RouteSiteName: "demo-site",
				FieldSources:  core.FieldSources{"/matchScheme": {Source: "app/default", Override: "hub"}},
			},
		},
	}
	ruleCore := newTestPortalRuleCore(repo, newTestPortalEntryRepoSpy(&core.PortalEntry{Id: 1, Enabled: true, Protocol: "http", Http: &core.PortalEntryHTTP{Http: "ENABLED", Https: "DISABLED", HttpPort: 80}}))
	service := &PortalRuleApiServiceServerImpl{PortalRuleCore: ruleCore, PortalEntryCore: ruleCore.PortalEntryCore}

	detail := service.Get(3)

	require.Len(t, detail.FieldSources, 1)
	assert.Equal(t, "/matchScheme", detail.FieldSources[0].Path)
	assert.Equal(t, "app/default", detail.FieldSources[0].Source)
	assert.Equal(t, "hub", detail.FieldSources[0].Override)
}

// Hub reports the rules that match the same request, so the Dashboard shows the
// operator what Portal cannot order on its own.
func TestPortalRuleServiceListConflicts(t *testing.T) {
	entryRepo := newTestPortalEntryRepoSpy(&core.PortalEntry{Id: 1, Name: "http:80", Enabled: true, Protocol: "http", Http: &core.PortalEntryHTTP{Http: "ENABLED", Https: "DISABLED", HttpPort: 80}})
	ruleRepo := &_NamedPortalRuleRepoSpy{
		items: map[string]*core.PortalRule{
			"web":   {Id: 1, Name: "web", EntryId: 1, RouteType: core.PortalRuleRouteTypeSite, RouteSiteName: "demo.Web", Enabled: true},
			"fixed": {Id: 2, Name: "fixed", EntryId: 1, RouteType: core.PortalRuleRouteTypeSite, RouteSiteName: "demo.Fixed", Enabled: true},
		},
	}
	service := &PortalRuleApiServiceServerImpl{PortalRuleCore: newTestPortalRuleCore(ruleRepo, entryRepo)}

	conflicts := service.ListConflicts()

	require.Len(t, conflicts, 1)
	assert.Equal(t, "fixed", conflicts[0].Rule)
	assert.Equal(t, 2, conflicts[0].RuleId)
	assert.Equal(t, "web", conflicts[0].ConflictRule)
	assert.Equal(t, 1, conflicts[0].ConflictRuleId)
	assert.Equal(t, "http:80", conflicts[0].Entry)
	assert.Equal(t, "http://<IP>:80", conflicts[0].Match)
	// Hub publishes the rule whose name sorts first, so the Dashboard can say
	// which one serves the request.
	assert.Equal(t, "fixed", conflicts[0].PublishedRule)
	assert.Equal(t, 2, conflicts[0].PublishedRuleId)
	assert.Equal(t, "web", conflicts[0].SuppressedRule)
	assert.Equal(t, 1, conflicts[0].SuppressedRuleId)
}
