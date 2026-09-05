package core

import (
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

func (s *entryRuleRepoSpy) ListRules() []PortalRule {
	s.calls = append(s.calls, "ListRules")
	rules := make([]PortalRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, *rule)
	}
	return vslice.SortBy(rules, func(a PortalRule, b PortalRule) bool {
		return a.Id < b.Id
	})
}

func (s *entryRuleRepoSpy) GetRuleById(id int) (*PortalRule, bool) {
	s.calls = append(s.calls, "GetRuleById")
	rule, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	value := *rule
	return &value, true
}

func (s *entryRuleRepoSpy) GetRuleByName(name string) (*PortalRule, bool) {
	s.calls = append(s.calls, "GetRuleByName:"+name)
	for _, rule := range s.rules {
		if rule.Name == name {
			value := *rule
			return &value, true
		}
	}
	return nil, false
}

func (s *entryRuleRepoSpy) SaveRule(rule *PortalRule) {
	s.calls = append(s.calls, "SaveRule")
	if s.rules == nil {
		s.rules = map[int]*PortalRule{}
	}
	value := *rule
	s.rules[value.Id] = &value
}

func (s *entryRuleRepoSpy) RemoveRule(id int) bool {
	s.calls = append(s.calls, "RemoveRule")
	if _, ok := s.rules[id]; !ok {
		return false
	}
	delete(s.rules, id)
	return true
}

func TestPortalRuleCoreUpdateBuiltInRule(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "vine.hub.dashboard-web", BuiltIn: true},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo}

	panicValue := capturePanic(func() {
		core.Update(1, PortalRuleUpdate{})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetRuleById"}, repo.calls)
}

func TestPortalRuleCoreRemoveBuiltInRule(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "vine.hub.dashboard-web", BuiltIn: true},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo}

	panicValue := capturePanic(func() {
		core.Remove(1)
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetRuleById"}, repo.calls)
}

func TestPortalRuleCoreUpdateDashboardAccess(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099, MatchPathPrefix: "/api", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099, MatchPathPrefix: "/", BuiltIn: true},
		},
	}
	certRepo := newTestPortalCertRepo()
	certRepo.SaveCert(&PortalCert{
		Name:             "hub-cert",
		Domains:          []string{"hub.example.com"},
		PrivateKeyBase64: "pri",
	})
	core := &PortalRuleCore{PortalRuleRepo: repo, PortalCertRepo: certRepo}

	rules := core.UpdateDashboardAccess("https", "hub.example.com", 8443, "/hub")

	require.Len(t, rules, 2)
	assert.Equal(t, "https", rules[0].MatchScheme)
	assert.Equal(t, "https", rules[1].MatchScheme)
	assert.Equal(t, "hub.example.com", rules[0].MatchHost)
	assert.Equal(t, "hub.example.com", rules[1].MatchHost)
	assert.Equal(t, 8443, rules[0].MatchPort)
	assert.Equal(t, 8443, rules[1].MatchPort)
	assert.Equal(t, "/api", rules[0].MatchPathPrefix)
	assert.Equal(t, "/hub", rules[1].MatchPathPrefix)
	assert.Equal(t, "https", repo.rules[1].MatchScheme)
	assert.Equal(t, "https", repo.rules[2].MatchScheme)
	assert.Equal(t, "hub.example.com", repo.rules[1].MatchHost)
	assert.Equal(t, "hub.example.com", repo.rules[2].MatchHost)
	assert.Equal(t, 8443, repo.rules[1].MatchPort)
	assert.Equal(t, 8443, repo.rules[2].MatchPort)
	assert.Equal(t, "/api", repo.rules[1].MatchPathPrefix)
	assert.Equal(t, "/hub", repo.rules[2].MatchPathPrefix)
	assert.Equal(t, []string{
		"GetRuleByName:" + DashboardAdminApiRuleName,
		"GetRuleByName:" + DashboardWebRuleName,
		"SaveRule",
		"SaveRule",
	}, repo.calls)
}

func TestPortalRuleCoreDashboardAccess(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchScheme: "https", MatchHost: "hub.example.com", MatchPort: 8443, MatchPathPrefix: "/api", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchScheme: "https", MatchHost: "hub.example.com", MatchPort: 8443, MatchPathPrefix: "/hub", BuiltIn: true},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo}

	access := core.DashboardAccess()

	assert.Equal(t, "https", access.Scheme)
	assert.Equal(t, "hub.example.com", access.Host)
	assert.Equal(t, 8443, access.Port)
	assert.Equal(t, "/hub", access.PathPrefix)
	assert.Equal(t, []string{
		"GetRuleByName:" + DashboardAdminApiRuleName,
		"GetRuleByName:" + DashboardWebRuleName,
	}, repo.calls)
}

func TestPortalRuleCoreListSkipsBuiltInRules(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", BuiltIn: true},
			3: {Id: 3, Name: DashboardWebRuleName},
			4: {Id: 4, Name: "demo", BuiltIn: true},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo, PortalCertRepo: newTestPortalCertRepo()}

	rules := core.List()

	require.Len(t, rules, 1)
	assert.Equal(t, 3, rules[0].Id)
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsNormalRule(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099, BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo}

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("http", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, 7099, repo.rules[1].MatchPort)
	assert.Equal(t, 7099, repo.rules[2].MatchPort)
	assert.Equal(t, []string{
		"GetRuleByName:" + DashboardAdminApiRuleName,
		"GetRuleByName:" + DashboardWebRuleName,
	}, repo.calls)
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsInvalidPort(t *testing.T) {
	core := &PortalRuleCore{PortalRuleRepo: &entryRuleRepoSpy{}, PortalCertRepo: newTestPortalCertRepo()}

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("http", "", -1, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalRuleCoreUpdateDashboardAccessNormalizesInput(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPathPrefix: "/api", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPathPrefix: "/", BuiltIn: true},
		},
	}
	core := &PortalRuleCore{PortalRuleRepo: repo, PortalCertRepo: newTestPortalCertRepo()}

	rules := core.UpdateDashboardAccess(" HTTP ", " hub.example.com ", 8080, "hub")

	require.Len(t, rules, 2)
	assert.Equal(t, "http", rules[0].MatchScheme)
	assert.Equal(t, "http", rules[1].MatchScheme)
	assert.Equal(t, "hub.example.com", rules[0].MatchHost)
	assert.Equal(t, "hub.example.com", rules[1].MatchHost)
	assert.Equal(t, "/api", rules[0].MatchPathPrefix)
	assert.Equal(t, "/hub", rules[1].MatchPathPrefix)
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsInvalidScheme(t *testing.T) {
	core := &PortalRuleCore{PortalRuleRepo: &entryRuleRepoSpy{}, PortalCertRepo: newTestPortalCertRepo()}

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("ftp", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsHttpsWithoutHost(t *testing.T) {
	core := &PortalRuleCore{PortalRuleRepo: &entryRuleRepoSpy{}, PortalCertRepo: newTestPortalCertRepo()}

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("https", "", 8443, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsHttpsWithoutCertificate(t *testing.T) {
	certRepo := newTestPortalCertRepo()
	certRepo.SaveCert(&PortalCert{
		Name:             "other-cert",
		Domains:          []string{"other.example.com"},
		PrivateKeyBase64: "pri",
	})
	core := &PortalRuleCore{PortalRuleRepo: &entryRuleRepoSpy{}, PortalCertRepo: certRepo}

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("https", "hub.example.com", 8443, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalCertDomainMatchesHost(t *testing.T) {
	assert.True(t, portalCertDomainMatchesHost("hub.example.com", "hub.example.com"))
	assert.True(t, portalCertDomainMatchesHost("*.example.com", "hub.example.com"))
	assert.False(t, portalCertDomainMatchesHost("*.example.com", "deep.hub.example.com"))
	assert.False(t, portalCertDomainMatchesHost("*.example.com", "example.com"))
}

func TestNormalizePortalRuleRoutePathPrefix(t *testing.T) {
	for input, want := range map[string]string{"": "", "/": "", "/internal/": "/internal", "/a%20b": "/a%20b"} {
		assert.Equal(t, want, normalizePortalRuleRoutePathPrefix("SITE", input))
	}
	for _, input := range []string{"relative", "//host/path", "https://host/path", "/a?x=1", "/a#f", "/../a", "/a/%2e%2e/b", "/a\\b", "/a%00", "/bad%", "/a b"} {
		t.Run(input, func(t *testing.T) { assert.Panics(t, func() { normalizePortalRuleRoutePathPrefix("SITE", input) }) })
	}
	assert.Panics(t, func() { normalizePortalRuleRoutePathPrefix("PERMANENT_REDIRECT", "/x") })
}

func TestPortalRuleTargetPathCreateUpdateClear(t *testing.T) {
	repo := &entryRuleRepoSpy{}
	service := &PortalRuleCore{PortalRuleRepo: repo}
	created := service.Create(PortalRuleCreation{Name: "mapping", MatchScheme: "http", RouteSiteName: "web", RouteType: "SITE", RoutePathPrefix: "/internal/"})
	assert.Equal(t, "/internal", created.RoutePathPrefix)
	updated := service.Update(created.Id, PortalRuleUpdate{RouteSiteName: new("next")})
	assert.Equal(t, "/internal", updated.RoutePathPrefix)
	assert.Panics(t, func() { service.Update(created.Id, PortalRuleUpdate{RouteType: new("TEMPORARY_REDIRECT")}) })
	cleared := service.Update(created.Id, PortalRuleUpdate{RoutePathPrefix: new("")})
	assert.Empty(t, cleared.RoutePathPrefix)
}

func validPortalRule() PortalRule {
	return PortalRule{Name: "rule", MatchScheme: "http", MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web"}
}

func TestPortalRuleValidationAcrossCreateUpdateSave(t *testing.T) {
	cases := map[string]func(*PortalRule){
		"empty name":               func(r *PortalRule) { r.Name = " " },
		"scheme":                   func(r *PortalRule) { r.MatchScheme = "ftp" },
		"negative port":            func(r *PortalRule) { r.MatchPort = -1 },
		"large port":               func(r *PortalRule) { r.MatchPort = 65536 },
		"host URL":                 func(r *PortalRule) { r.MatchHost = "https://example.com" },
		"host port":                func(r *PortalRule) { r.MatchHost = "example.com:80" },
		"relative prefix":          func(r *PortalRule) { r.MatchPathPrefix = "api" },
		"query prefix":             func(r *PortalRule) { r.MatchPathPrefix = "/api?x=1" },
		"dot prefix":               func(r *PortalRule) { r.MatchPathPrefix = "/api/.." },
		"route type":               func(r *PortalRule) { r.RouteType = "UNKNOWN" },
		"missing site":             func(r *PortalRule) { r.RouteSiteName = "" },
		"site redirect field":      func(r *PortalRule) { r.RouteRedirectionPattern = "/login" },
		"invalid route prefix":     func(r *PortalRule) { r.RoutePathPrefix = "/../private" },
		"redirect missing pattern": func(r *PortalRule) { r.RouteType = PortalRuleRouteTypeTemporaryRedirect; r.RouteSiteName = "" },
		"redirect with site": func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteRedirectionPattern = "/login"
		},
		"redirect with prefix": func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "/login"
			r.RoutePathPrefix = "/x"
		},
		"unknown placeholder": func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "https://example.com{unknown}"
		},
		"broken placeholder": func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "https://example.com{uri"
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			bad := validPortalRule()
			change(&bad)
			repo := &entryRuleRepoSpy{}
			service := &PortalRuleCore{PortalRuleRepo: repo}
			require.Panics(t, func() {
				service.Create(PortalRuleCreation{
					Name: bad.Name, MatchScheme: bad.MatchScheme, MatchHost: bad.MatchHost, MatchPort: bad.MatchPort,
					MatchPathPrefix: bad.MatchPathPrefix, RouteType: bad.RouteType, RouteSiteName: bad.RouteSiteName,
					RoutePathPrefix: bad.RoutePathPrefix, RouteRedirectionPattern: bad.RouteRedirectionPattern,
				})
			})
			require.NotContains(t, repo.calls, "SaveRule")
			repo.calls = nil
			require.Panics(t, func() { service.Save(bad) })
			require.NotContains(t, repo.calls, "SaveRule")
			original := validPortalRule()
			original.Id = 7
			repo.rules = map[int]*PortalRule{7: &original}
			repo.calls = nil
			require.Panics(t, func() {
				service.Update(7, PortalRuleUpdate{
					Name: &bad.Name, MatchScheme: &bad.MatchScheme, MatchHost: &bad.MatchHost, MatchPort: &bad.MatchPort,
					MatchPathPrefix: &bad.MatchPathPrefix, RouteType: &bad.RouteType, RouteSiteName: &bad.RouteSiteName,
					RoutePathPrefix: &bad.RoutePathPrefix, RouteRedirectionPattern: &bad.RouteRedirectionPattern,
				})
			})
			require.NotContains(t, repo.calls, "SaveRule")
			require.Equal(t, original, *repo.rules[7])
		})
	}
}

func TestPortalRuleSaveIdentityAndPartialUpdate(t *testing.T) {
	original := validPortalRule()
	original.Id = 17
	original.RoutePathPrefix = "/internal"
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{17: &original}}
	service := &PortalRuleCore{PortalRuleRepo: repo}
	next := validPortalRule()
	next.Id = 99
	next.RoutePathPrefix = "/next/"
	saved := service.Save(next)
	require.Equal(t, 17, saved.Id)
	require.Equal(t, "/next", saved.RoutePathPrefix)
	// Omitted route fields survive an access-only update.
	updated := service.Update(17, PortalRuleUpdate{MatchPort: new(443)})
	require.Equal(t, "/next", updated.RoutePathPrefix)
	require.Equal(t, "web", updated.RouteSiteName)
	// A route transition must clear old fields and provide the new required fields.
	updated = service.Update(17, PortalRuleUpdate{RouteType: new(PortalRuleRouteTypePermanentRedirect), RouteSiteName: new(""), RoutePathPrefix: new(""), RouteRedirectionPattern: new("https://example.com{uri}")})
	require.Equal(t, PortalRuleRouteTypePermanentRedirect, updated.RouteType)
	repo.rules[17].BuiltIn = true
	repo.calls = nil
	require.Panics(t, func() { service.Save(next) })
	require.NotContains(t, repo.calls, "SaveRule")
}

func TestPortalRuleValidHostsAndDefaultPort(t *testing.T) {
	for _, host := range []string{"", "example.com", "localhost", "127.0.0.1", "::1"} {
		rule := validPortalRule()
		rule.MatchHost = host
		require.NotPanics(t, rule.normalizeAndValidate)
		require.Zero(t, rule.MatchPort)
	}
}

func TestPortalRuleCoreValidateDoesNotAccessStorage(t *testing.T) {
	repo := &entryRuleRepoSpy{}
	service := &PortalRuleCore{PortalRuleRepo: repo}
	rule := validPortalRule()
	rule.RoutePathPrefix = "/internal/"
	normalized := service.Validate(rule)
	require.Equal(t, "/internal", normalized.RoutePathPrefix)
	require.Equal(t, "/internal/", rule.RoutePathPrefix)
	for _, name := range []string{DashboardAdminApiRuleName, DashboardWebRuleName} {
		rule.Name = name
		require.Panics(t, func() { service.Validate(rule) })
	}
	require.Empty(t, repo.calls)
}
