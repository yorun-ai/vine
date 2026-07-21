package impl

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
	service := newTestPortalRuleService(false)

	access := service.GetDashboardAccess()

	assert.Equal(t, "http", access.Scheme)
	assert.Equal(t, "", access.Host)
	assert.Equal(t, 7099, access.Port)
	assert.Equal(t, "/", access.PathPrefix)
	assert.True(t, access.CanUpdate)
}

func TestPortalRuleServiceGetDashboardAccessLockedByFlag(t *testing.T) {
	service := newTestPortalRuleService(true)

	access := service.GetDashboardAccess()

	assert.False(t, access.CanUpdate)
}

func TestPortalRuleServiceUpdateDashboardAccessRejectsLockedFlag(t *testing.T) {
	service := newTestPortalRuleService(true)

	panicValue := capturePanic(func() {
		service.UpdateDashboardAccess("http", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func newTestPortalRuleService(dashboardURLSet bool) *PortalRuleServiceServerImpl {
	return &PortalRuleServiceServerImpl{
		PortalRuleCore: &core.PortalRuleCore{
			PortalRuleRepo: &_PortalRuleRepoSpy{
				rules: map[int]*core.PortalRule{
					1: {Id: 1, Name: core.DashboardAdminApiRuleName, Scheme: "http", Port: 7099, PathPrefix: "/api", BuiltIn: true},
					2: {Id: 2, Name: core.DashboardWebRuleName, Scheme: "http", Port: 7099, PathPrefix: "/", BuiltIn: true},
				},
			},
		},
		Flag: &flag.Flag{DashboardURLSet: dashboardURLSet},
	}
}

type _PortalRuleRepoSpy struct {
	rules map[int]*core.PortalRule
}

func (s *_PortalRuleRepoSpy) ListRules() []core.PortalRule {
	rules := make([]core.PortalRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, *rule)
	}
	return vslice.SortBy(rules, func(a core.PortalRule, b core.PortalRule) bool {
		return a.Id < b.Id
	})
}

func (s *_PortalRuleRepoSpy) GetRuleById(id int) (*core.PortalRule, bool) {
	rule, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	value := *rule
	return &value, true
}

func (s *_PortalRuleRepoSpy) GetRuleByName(name string) (*core.PortalRule, bool) {
	for _, rule := range s.rules {
		if rule.Name == name {
			value := *rule
			return &value, true
		}
	}
	return nil, false
}

func (s *_PortalRuleRepoSpy) SaveRule(rule *core.PortalRule) {
	value := *rule
	s.rules[value.Id] = &value
}

func (s *_PortalRuleRepoSpy) RemoveRule(id int) bool {
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
