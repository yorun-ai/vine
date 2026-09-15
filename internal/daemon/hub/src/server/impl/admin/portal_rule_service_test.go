package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vslice"
)

func TestPortalRuleServiceGetDashboardAccessCanUpdate(t *testing.T) {
	service := newTestPortalRuleApiService(false)

	access := service.GetDashboardAccess()

	assert.Equal(t, "http", access.Scheme)
	assert.Equal(t, "", access.Host)
	assert.Equal(t, 7099, access.Port)
	assert.Equal(t, "/", access.PathPrefix)
	assert.True(t, access.CanUpdate)
}

func TestPortalRuleServiceGetDashboardAccessLockedByFlag(t *testing.T) {
	service := newTestPortalRuleApiService(true)

	access := service.GetDashboardAccess()

	assert.False(t, access.CanUpdate)
}

func TestPortalRuleServiceUpdateDashboardAccessRejectsLockedFlag(t *testing.T) {
	service := newTestPortalRuleApiService(true)

	panicValue := capturePanic(func() {
		service.UpdateDashboardAccess("http", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func newTestPortalRuleApiService(dashboardURLSet bool) *PortalRuleApiServiceServerImpl {
	return &PortalRuleApiServiceServerImpl{
		PortalRuleCore: newTestPortalRuleCore(
			&_PortalRuleRepoSpy{
				rules: map[int]*core.PortalRule{
					1: {Id: 1, Name: core.DashboardAdminApiRuleName, MatchScheme: "http", MatchPort: 7099, MatchPathPrefix: "/api", BuiltIn: true},
					2: {Id: 2, Name: core.DashboardWebRuleName, MatchScheme: "http", MatchPort: 7099, MatchPathPrefix: "/", BuiltIn: true},
				},
			},
		),
		Flag: &flag.Flag{DashboardURLSet: dashboardURLSet},
	}
}

// newTestPortalSiteCore builds a site core with the repositories Hub injects.
func newTestPortalSiteCore(siteRepo core.PortalSiteRepo) *core.PortalSiteCore {
	return &core.PortalSiteCore{PortalSiteRepo: siteRepo, SchemaRepo: &_SkeletonServiceSchemaRepo{}}
}

// _MaintenanceServicePortalCertRepo is a map-backed certificate repository.
type _MaintenanceServicePortalCertRepo struct {
	items map[string]*core.PortalCert
}

func (r *_MaintenanceServicePortalCertRepo) List() []*core.PortalCert {
	items := make([]*core.PortalCert, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalCert, b *core.PortalCert) bool { return a.Id < b.Id })
}

func (r *_MaintenanceServicePortalCertRepo) GetById(id int) (*core.PortalCert, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return item, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServicePortalCertRepo) GetByName(name string) (*core.PortalCert, bool) {
	item, ok := r.items[name]
	return item, ok
}

func (r *_MaintenanceServicePortalCertRepo) Save(cert *core.PortalCert) {
	if r.items == nil {
		r.items = map[string]*core.PortalCert{}
	}
	r.items[cert.Name] = cert
}

func (r *_MaintenanceServicePortalCertRepo) Remove(id int) bool {
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
	return &core.PortalCertCore{PortalCertRepo: &_MaintenanceServicePortalCertRepo{}}
}

// newTestPortalRuleCore builds a rule core with the chosen rule repository.
func newTestPortalRuleCore(ruleRepo core.PortalRuleRepo) *core.PortalRuleCore {
	return &core.PortalRuleCore{
		PortalRuleRepo: ruleRepo,
	}
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
			MatchScheme:   "http",
			MatchPort:     80,
			RouteType:     core.PortalRuleRouteTypeSite,
			RouteSiteName: "demo-site",
			FieldSources:  core.FieldSources{"/matchScheme": {Source: "app/default", Override: "hub"}},
		},
	}}
	service := &PortalRuleApiServiceServerImpl{PortalRuleCore: newTestPortalRuleCore(repo)}

	detail := service.Get(3)

	require.Len(t, detail.FieldSources, 1)
	assert.Equal(t, "/matchScheme", detail.FieldSources[0].Path)
	assert.Equal(t, "app/default", detail.FieldSources[0].Source)
	assert.Equal(t, "hub", detail.FieldSources[0].Override)
}
