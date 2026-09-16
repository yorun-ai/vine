package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPortalRuleCoreSaveJoinsNamedEntry(t *testing.T) {
	// A seed may name the entry a rule joins instead of declaring an access.
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 4, Name: "web", Scheme: "https", Host: "app.example.com", Port: 8443})
	repo := &entryRuleRepoSpy{}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	saved := core.Save(PortalRule{
		Name: "demo.web", EntryName: "web", MatchPathPrefix: "/",
		RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
	})

	assert.Equal(t, 4, saved.EntryId)
	assert.Equal(t, "https", saved.MatchScheme)
	assert.Equal(t, "app.example.com", saved.MatchHost)
	assert.Equal(t, 8443, saved.MatchPort)
	// The entry owns the access from here on.
	assert.Empty(t, saved.EntryName)

	// A rule declares an entry name or an access, never both.
	require.PanicsWithError(t,
		`portal rule "api": entryName cannot be mixed with matchScheme, matchHost, or matchPort type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Save(PortalRule{
				Name: "api", EntryName: "web", MatchScheme: "https", MatchHost: "app.example.com", MatchPort: 8443,
				MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
			})
		})

	// A named entry Hub does not store fails with its name.
	require.PanicsWithError(t, "portal entry missing not found type=APPLICATION code=OPERATION_FAILED",
		func() {
			core.Save(PortalRule{
				Name: "other", EntryName: "missing", MatchPathPrefix: "/",
				RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
			})
		})
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
	require.Empty(t, repo.calls)
}

func TestPortalRuleCoreRejectsRuleMatchingSameRequest(t *testing.T) {
	// Two rules that match the same request have no defined order in Portal, so
	// Hub rejects the second one even when it belongs to another entry.
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {
			Id: 1, Name: "vine", EntryId: 9, MatchScheme: "http", MatchPort: 7099,
			MatchPathPrefix: "/", RouteType: PortalRuleRouteTypeSite,
		},
	}}
	entryRepo := newPortalEntryRepoSpy(
		&PortalEntry{Id: 5, Scheme: "http", Port: 7099},
		&PortalEntry{Id: 9, Scheme: "http", Port: 7099},
	)
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	creation := PortalRuleCreation{
		Name: "demo.shadow", EntryName: "http:7099", MatchPathPrefix: "/",
		RouteType: PortalRuleRouteTypeSite, RouteSiteName: "demo-site",
	}
	require.PanicsWithError(t,
		`portal rule "vine" already matches http://*:7099/ type=APPLICATION code=OPERATION_FAILED`,
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
