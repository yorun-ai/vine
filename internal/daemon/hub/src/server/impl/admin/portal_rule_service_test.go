package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

// newTestPortalSiteCore builds a site core with the repositories Hub injects.
func newTestPortalSiteCore(siteRepo core.PortalSiteRepo) *core.PortalSiteCore {
	return &core.PortalSiteCore{PortalSiteRepo: siteRepo, SchemaRepo: &_SkeletonServiceSchemaRepo{}}
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

// newTestPortalCertCore builds a certificate core with an empty repository.
func newTestPortalCertCore() *core.PortalCertCore {
	return &core.PortalCertCore{PortalCertRepo: &_PortalCertRepoSpy{}}
}

// newTestPortalRuleCore builds a rule core with the chosen rule repository and
// an entry repository that starts empty.
func newTestPortalRuleCore(ruleRepo core.PortalRuleRepo, entryRepos ...core.PortalEntryRepo) *core.PortalRuleCore {
	entryRepo := core.PortalEntryRepo(newTestPortalEntryRepoSpy())
	if len(entryRepos) > 0 {
		entryRepo = entryRepos[0]
	}
	return &core.PortalRuleCore{
		PortalRuleRepo: ruleRepo,
		PortalEntryCore: &core.PortalEntryCore{
			PortalEntryRepo: entryRepo,
			PortalRuleRepo:  ruleRepo,
			PortalSiteRepo:  &_PortalSiteRepoSpy{items: map[string]*core.PortalSite{}},
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

func (s *_PortalEntryRepoSpy) GetByAccess(scheme string, host string, port int) (*core.PortalEntry, bool) {
	for _, entry := range s.List() {
		if entry.Scheme == scheme && entry.Host == host && entry.Port == port {
			return entry, true
		}
	}
	return nil, false
}

func (s *_PortalEntryRepoSpy) Save(entry *core.PortalEntry) {
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

func (s *_PortalRuleRepoSpy) Save(rule *core.PortalRule) {
	value := *rule
	s.rules[value.Id] = &value
}

func (s *_PortalRuleRepoSpy) Remove(id int) bool {
	if _, ok := s.rules[id]; !ok {
		return false
	}
	delete(s.rules, id)
	return true
}

func capturePanic(fn func()) (got any) {
	defer func() {
		got = recover()
	}()
	fn()
	return nil
}

func TestPortalRuleServiceGetReturnsFieldSources(t *testing.T) {
	repo := &_PortalRuleRepoSpy{rules: map[int]*core.PortalRule{
		3: {
			Id:            3,
			Name:          "demo.rule",
			EntryId:       1,
			RouteType:     core.PortalRuleRouteTypeSite,
			RouteSiteName: "demo-site",
			FieldSources:  core.FieldSources{"/matchScheme": {Source: "app/default", Override: "hub"}},
		},
	}}
	ruleCore := newTestPortalRuleCore(repo, newTestPortalEntryRepoSpy(&core.PortalEntry{Id: 1, Scheme: "http", Port: 80, Enabled: true}))
	service := &PortalRuleApiServiceServerImpl{PortalRuleCore: ruleCore, PortalEntryCore: ruleCore.PortalEntryCore}

	detail := service.Get(3)

	require.Len(t, detail.FieldSources, 1)
	assert.Equal(t, "/matchScheme", detail.FieldSources[0].Path)
	assert.Equal(t, "app/default", detail.FieldSources[0].Source)
	assert.Equal(t, "hub", detail.FieldSources[0].Override)
}
