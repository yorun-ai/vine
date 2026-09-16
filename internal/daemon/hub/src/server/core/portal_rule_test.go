package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
)

func TestResolvePortalRulePaths(t *testing.T) {
	rule := &PortalRule{
		RouteType:       PortalRuleRouteTypeSite,
		MatchPathPrefix: "/configured",
		RoutePathPrefix: "/backend",
	}
	tests := []struct {
		name  string
		site  *PortalSite
		match string
		route string
	}{
		{name: "without site", match: "/configured", route: "/backend"},
		{name: "empty mount", site: &PortalSite{WebMountPath: ""}, match: "/configured", route: "/backend"},
		{name: "root mount", site: &PortalSite{WebMountPath: "/"}, match: "/", route: ""},
		{name: "trim trailing slash", site: &PortalSite{WebMountPath: "/app/"}, match: "/app", route: "/app"},
		{name: "nested mount", site: &PortalSite{WebMountPath: "/app/admin"}, match: "/app/admin", route: "/app/admin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, route := ResolvePortalRulePaths(rule, tt.site)
			assert.Equal(t, tt.match, match)
			assert.Equal(t, tt.route, route)
		})
	}

	redirect := *rule
	redirect.RouteType = PortalRuleRouteTypeTemporaryRedirect
	match, route := ResolvePortalRulePaths(&redirect, &PortalSite{WebMountPath: "/app"})
	assert.Equal(t, "/configured", match)
	assert.Equal(t, "/backend", route)
}

func TestPortalRuleCoreUpdateBuiltInRule(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "vine.hub.dashboard-web", BuiltIn: true},
		},
	}
	core := newPortalRuleCoreForTest(repo, nil)

	panicValue := capturePanic(func() {
		core.Update(1, PortalRuleUpdate{})
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetById"}, repo.calls)
}

func TestPortalRuleCoreRemoveBuiltInRule(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: "vine.hub.dashboard-web", BuiltIn: true},
		},
	}
	core := newPortalRuleCoreForTest(repo, nil)

	panicValue := capturePanic(func() {
		core.Remove(1)
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, []string{"GetById"}, repo.calls)
}

func TestPortalRuleCoreUpdateDashboardAccess(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099, MatchPathPrefix: "/api", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchPort: 7099, MatchPathPrefix: "/", BuiltIn: true},
		},
	}
	certRepo := newTestPortalCertRepo()
	certRepo.Save(&PortalCert{
		Name:             "hub-cert",
		Domains:          []string{"hub.example.com"},
		PrivateKeyBase64: "pri",
	})
	core := newPortalRuleCoreForTest(repo, certRepo)

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
		"GetByName:" + DashboardAdminApiRuleName,
		"GetByName:" + DashboardWebRuleName,
		"List",
		"Save",
		"Save",
	}, repo.calls)
}

func TestPortalRuleCoreDashboardAccess(t *testing.T) {
	repo := &entryRuleRepoSpy{
		rules: map[int]*PortalRule{
			1: {Id: 1, Name: DashboardAdminApiRuleName, EntryId: 4, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchScheme: "https", MatchHost: "hub.example.com", MatchPort: 8443, MatchPathPrefix: "/api", BuiltIn: true},
			2: {Id: 2, Name: DashboardWebRuleName, EntryId: 4, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "dashboard", MatchScheme: "https", MatchHost: "hub.example.com", MatchPort: 8443, MatchPathPrefix: "/hub", BuiltIn: true},
		},
	}
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 4, Scheme: "https", Host: "hub.example.com", Port: 8443, BuiltIn: true})
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	access := core.DashboardAccess()

	assert.Equal(t, "https", access.Scheme)
	assert.Equal(t, "hub.example.com", access.Host)
	assert.Equal(t, 8443, access.Port)
	assert.Equal(t, "/hub", access.PathPrefix)
	assert.Equal(t, []string{
		"GetByName:" + DashboardAdminApiRuleName,
		"GetByName:" + DashboardWebRuleName,
	}, repo.calls)
	assert.Equal(t, []string{"GetById:4"}, entryRepo.calls)
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
	core := newPortalRuleCoreForTest(repo, newTestPortalCertRepo())

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
	core := newPortalRuleCoreForTest(repo, nil)

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("http", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
	assert.Equal(t, 7099, repo.rules[1].MatchPort)
	assert.Equal(t, 7099, repo.rules[2].MatchPort)
	assert.Equal(t, []string{
		"GetByName:" + DashboardAdminApiRuleName,
		"GetByName:" + DashboardWebRuleName,
	}, repo.calls)
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsInvalidPort(t *testing.T) {
	core := newPortalRuleCoreForTest(&entryRuleRepoSpy{}, newTestPortalCertRepo())

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
	core := newPortalRuleCoreForTest(repo, newTestPortalCertRepo())

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
	core := newPortalRuleCoreForTest(&entryRuleRepoSpy{}, newTestPortalCertRepo())

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("ftp", "", 8080, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsHttpsWithoutHost(t *testing.T) {
	core := newPortalRuleCoreForTest(&entryRuleRepoSpy{}, newTestPortalCertRepo())

	panicValue := capturePanic(func() {
		core.UpdateDashboardAccess("https", "", 8443, "/")
	})

	err, ok := panicValue.(ex.Error)
	require.True(t, ok)
	assert.Equal(t, ex.OperationFailed, err.Code())
}

func TestPortalRuleCoreUpdateDashboardAccessRejectsHttpsWithoutCertificate(t *testing.T) {
	certRepo := newTestPortalCertRepo()
	certRepo.Save(&PortalCert{
		Name:             "other-cert",
		Domains:          []string{"other.example.com"},
		PrivateKeyBase64: "pri",
	})
	core := newPortalRuleCoreForTest(&entryRuleRepoSpy{}, certRepo)

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
	service := newPortalRuleCoreWithEntriesForTest(repo, nil, newTestPortalRuleEntryRepo())
	created := service.Create(PortalRuleCreation{Name: "mapping", EntryName: "http:80", RouteSiteName: "web", RouteType: "SITE", RoutePathPrefix: "/internal/"})
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
	// accessOnly marks a case only a rule carrying an access can produce: the
	// Admin API takes the access from the Portal entry, while a seed still
	// declares it on the rule.
	cases := map[string]struct {
		change     func(*PortalRule)
		accessOnly bool
	}{
		"empty name":               {change: func(r *PortalRule) { r.Name = " " }},
		"scheme":                   {change: func(r *PortalRule) { r.MatchScheme = "ftp" }, accessOnly: true},
		"negative port":            {change: func(r *PortalRule) { r.MatchPort = -1 }, accessOnly: true},
		"large port":               {change: func(r *PortalRule) { r.MatchPort = 65536 }, accessOnly: true},
		"host URL":                 {change: func(r *PortalRule) { r.MatchHost = "https://example.com" }, accessOnly: true},
		"host port":                {change: func(r *PortalRule) { r.MatchHost = "example.com:80" }, accessOnly: true},
		"relative prefix":          {change: func(r *PortalRule) { r.MatchPathPrefix = "api" }},
		"query prefix":             {change: func(r *PortalRule) { r.MatchPathPrefix = "/api?x=1" }},
		"dot prefix":               {change: func(r *PortalRule) { r.MatchPathPrefix = "/api/.." }},
		"route type":               {change: func(r *PortalRule) { r.RouteType = "UNKNOWN" }},
		"missing site":             {change: func(r *PortalRule) { r.RouteSiteName = "" }},
		"site redirect field":      {change: func(r *PortalRule) { r.RouteRedirectionPattern = "/login" }},
		"invalid route prefix":     {change: func(r *PortalRule) { r.RoutePathPrefix = "/../private" }},
		"redirect missing pattern": {change: func(r *PortalRule) { r.RouteType = PortalRuleRouteTypeTemporaryRedirect; r.RouteSiteName = "" }},
		"redirect with site": {change: func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteRedirectionPattern = "/login"
		}},
		"redirect with prefix": {change: func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "/login"
			r.RoutePathPrefix = "/x"
		}},
		"unknown placeholder": {change: func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "https://example.com{unknown}"
		}},
		"broken placeholder": {change: func(r *PortalRule) {
			r.RouteType = PortalRuleRouteTypeTemporaryRedirect
			r.RouteSiteName = ""
			r.RouteRedirectionPattern = "https://example.com{uri"
		}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			bad := validPortalRule()
			test.change(&bad)
			repo := &entryRuleRepoSpy{}
			service := newPortalRuleCoreWithEntriesForTest(repo, nil, newTestPortalRuleEntryRepo())
			if !test.accessOnly {
				require.Panics(t, func() {
					service.Create(PortalRuleCreation{
						Name: bad.Name, EntryName: "http:80",
						MatchPathPrefix: bad.MatchPathPrefix, RouteType: bad.RouteType, RouteSiteName: bad.RouteSiteName,
						RoutePathPrefix: bad.RoutePathPrefix, RouteRedirectionPattern: bad.RouteRedirectionPattern,
					})
				})
				require.NotContains(t, repo.calls, "Save")
			}
			repo.calls = nil
			require.Panics(t, func() { service.Save(bad) })
			require.NotContains(t, repo.calls, "Save")
			original := validPortalRule()
			original.Id = 7
			repo.rules = map[int]*PortalRule{7: &original}
			repo.calls = nil
			if !test.accessOnly {
				require.Panics(t, func() {
					service.Update(7, PortalRuleUpdate{
						Name:                    &bad.Name,
						MatchPathPrefix:         &bad.MatchPathPrefix,
						RouteType:               &bad.RouteType,
						RouteSiteName:           &bad.RouteSiteName,
						RoutePathPrefix:         &bad.RoutePathPrefix,
						RouteRedirectionPattern: &bad.RouteRedirectionPattern,
					})
				})
				require.NotContains(t, repo.calls, "Save")
			}
			require.Equal(t, original, *repo.rules[7])
		})
	}
}

func TestPortalRuleSaveIdentityAndPartialUpdate(t *testing.T) {
	original := validPortalRule()
	original.Id = 17
	original.RoutePathPrefix = "/internal"
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{17: &original}}
	service := newPortalRuleCoreForTest(repo, nil)
	next := validPortalRule()
	next.Id = 99
	next.RoutePathPrefix = "/next/"
	saved := service.Save(next)
	require.Equal(t, 17, saved.Id)
	require.Equal(t, "/next", saved.RoutePathPrefix)
	// Omitted fields survive a partial update, and the entry keeps the access.
	updated := service.Update(17, PortalRuleUpdate{MatchPathPrefix: new("/kept")})
	require.Equal(t, "/kept", updated.MatchPathPrefix)
	require.Equal(t, "/next", updated.RoutePathPrefix)
	require.Equal(t, "web", updated.RouteSiteName)
	require.Equal(t, "http", updated.MatchScheme)
	require.Equal(t, 80, updated.MatchPort)
	// A route transition must clear old fields and provide the new required fields.
	updated = service.Update(17, PortalRuleUpdate{RouteType: new(PortalRuleRouteTypePermanentRedirect), RouteSiteName: new(""), RoutePathPrefix: new(""), RouteRedirectionPattern: new("https://example.com{uri}")})
	require.Equal(t, PortalRuleRouteTypePermanentRedirect, updated.RouteType)
	repo.rules[17].BuiltIn = true
	repo.calls = nil
	require.Panics(t, func() { service.Save(next) })
	require.NotContains(t, repo.calls, "Save")
}

func TestPortalRuleValidHostsAndDefaultPort(t *testing.T) {
	for _, host := range []string{"", "example.com", "localhost", "127.0.0.1", "::1"} {
		rule := validPortalRule()
		rule.MatchHost = host
		require.NotPanics(t, rule.normalizeAndValidate)
		require.Zero(t, rule.MatchPort)
	}
}

func TestPortalRuleCoreValidateKeepsRuleRepoUntouched(t *testing.T) {
	repo := &entryRuleRepoSpy{}
	service := newPortalRuleCoreForTest(repo, nil)
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

func TestPortalRuleCoreRejectsRuleMatchingSameRequest(t *testing.T) {
	// Two rules that match the same request have no defined order in Portal, so
	// Hub rejects the second one even when it belongs to another entry.
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {
			Id: 1, Name: DashboardWebRuleName, EntryId: 9, MatchScheme: "http", MatchPort: 7099,
			MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, BuiltIn: true,
		},
	}}
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 5, Scheme: "http", Port: 7099},
		&PortalEntry{Id: 9, Scheme: "http", Port: 7099, BuiltIn: true},
	)
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	creation := PortalRuleCreation{
		Name: "demo.shadow", EntryName: "http:7099", MatchPathPrefix: "/",
		RouteType: PortalRuleRouteTypeSite, RouteSiteName: "demo-site",
	}
	require.PanicsWithError(t,
		`portal rule "vine.hub.dashboard-web" already matches http://*:7099/ type=APPLICATION code=OPERATION_FAILED`,
		func() { core.Create(creation) })
	require.NotContains(t, repo.calls, "Save")

	// The same access remains available for another path, and another access for
	// the same path: that is the entry aggregation Hub supports.
	creation.MatchPathPrefix = "/app"
	require.NotNil(t, core.Create(creation))
}

func TestPortalRuleCoreUpdateRejectsMatchTakenByAnotherRule(t *testing.T) {
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 80, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
		2: {Id: 2, Name: "api", EntryId: 1, MatchScheme: "http", MatchPort: 80, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80}))

	require.PanicsWithError(t,
		`portal rule "web" already matches http://*:80/ type=APPLICATION code=OPERATION_FAILED`,
		func() { core.Update(2, PortalRuleUpdate{MatchPathPrefix: new("/")}) })
	require.Equal(t, "/api", repo.rules[2].MatchPathPrefix)
}

func TestPortalRuleCoreRejectsRuleWithDefaultPortMatchingAnotherRule(t *testing.T) {
	// Hub does not move a rule aside on its own: a rule that matches the request
	// of a stored rule fails, so the operator corrects the seed or the import.
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80})
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 80, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	require.PanicsWithError(t,
		`portal rule "web" already matches http://*:80/ type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Create(PortalRuleCreation{
				Name: "demo", EntryName: "http:80", MatchPathPrefix: "/",
				RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
			})
		})
	require.NotContains(t, repo.calls, "Save")
}

func TestPortalRuleCoreKeepsPathOfRuleWithExplicitPort(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80})
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, MatchScheme: "http", MatchPort: 80, MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site"},
	}}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	created := core.Create(PortalRuleCreation{
		Name: "api", EntryName: "http:80", MatchPathPrefix: "/api",
		RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
	})

	assert.Equal(t, "/api", created.MatchPathPrefix)
	assert.Equal(t, 1, created.EntryId)
}

func TestPortalRuleCorePreservesConfiguredPathsWithoutResolvingSites(t *testing.T) {
	service := newPortalRuleCoreForTest(&entryRuleRepoSpy{}, nil)
	for _, path := range []string{"", "/", "/configured"} {
		rule := validPortalRule()
		rule.MatchPathPrefix = path
		rule.RoutePathPrefix = path
		saved := service.Save(rule)
		require.Equal(t, path, saved.MatchPathPrefix)
		require.Equal(t, strings.TrimRight(path, "/"), saved.RoutePathPrefix)
	}
}

// newTestPortalRuleEntryRepo builds the entry repository Hub stores a rule in
// when the Admin API creates it: one entry, named after the access it serves.
func newTestPortalRuleEntryRepo() *portalEntryRepoSpy {
	return newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80})
}

// newPortalRuleCoreForTest builds a rule core with the repositories Hub injects,
// so tests only choose the repositories they exercise. Rules resolve the entry
// that owns their access, so the core also carries an entry repository.
func newPortalRuleCoreForTest(ruleRepo PortalRuleRepo, certRepo PortalCertRepo) *PortalRuleCore {
	return newPortalRuleCoreWithEntriesForTest(ruleRepo, certRepo, newPortalEntryRepoSpy())
}

func newPortalRuleCoreWithEntriesForTest(ruleRepo PortalRuleRepo, certRepo PortalCertRepo, entryRepo PortalEntryRepo) *PortalRuleCore {
	if certRepo == nil {
		certRepo = newTestPortalCertRepo()
	}
	return &PortalRuleCore{
		PortalRuleRepo:  ruleRepo,
		PortalCertRepo:  certRepo,
		PortalEntryCore: newPortalEntryCoreForTest(ruleRepo, entryRepo, nil),
	}
}
