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
	created := service.Create(PortalRule{Name: "mapping", EntryId: 1, RouteSiteName: "web", RouteType: "SITE", RoutePathPrefix: "/internal/"})
	assert.Equal(t, "/internal", created.RoutePathPrefix)
	updated := service.Update(created.Id, PortalRuleUpdate{RouteSiteName: new("next")})
	assert.Equal(t, "/internal", updated.RoutePathPrefix)
	assert.Panics(t, func() { service.Update(created.Id, PortalRuleUpdate{RouteType: new("TEMPORARY_REDIRECT")}) })
	cleared := service.Update(created.Id, PortalRuleUpdate{RoutePathPrefix: new("")})
	assert.Empty(t, cleared.RoutePathPrefix)
}

func validPortalRule() PortalRule {
	return PortalRule{Name: "rule", EntryId: 1, MatchPathPrefix: "/api", RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web"}
}

func TestPortalRuleValidationAcrossCreateUpdateSave(t *testing.T) {
	// The rule owns routing only: the access belongs to the entry it joins, and
	// the entry validates it.
	cases := map[string]struct {
		change func(*PortalRule)
	}{
		"empty name":               {change: func(r *PortalRule) { r.Name = " " }},
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
			require.Panics(t, func() {
				service.Create(PortalRule{
					Name: bad.Name, EntryId: bad.EntryId,
					MatchPathPrefix: bad.MatchPathPrefix, RouteType: bad.RouteType, RouteSiteName: bad.RouteSiteName,
					RoutePathPrefix: bad.RoutePathPrefix, RouteRedirectionPattern: bad.RouteRedirectionPattern,
				})
			})
			require.NotContains(t, repo.calls, "Save")
			repo.calls = nil
			require.Panics(t, func() { service.Save(bad) })
			require.NotContains(t, repo.calls, "Save")
			original := validPortalRule()
			original.Id = 7
			repo.rules = map[int]*PortalRule{7: &original}
			repo.calls = nil
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
			require.Equal(t, original, *repo.rules[7])
		})
	}
}

// The caller resolves the entry a rule joins, so the rule core stores a complete
// rule and rejects one that names no entry at all.
func TestPortalRuleCoreSaveKeepsEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 4, Name: "web", Scheme: "https", Host: "app.example.com", Port: 8443})
	repo := &entryRuleRepoSpy{}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	saved := core.Save(PortalRule{
		Name: "demo.web", EntryId: 4, MatchPathPrefix: "/",
		RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
	})
	assert.Equal(t, 4, saved.EntryId)

	require.PanicsWithError(t, `portal rule "api": the entry it belongs to is required type=APPLICATION code=OPERATION_FAILED`,
		func() {
			core.Save(PortalRule{
				Name: "api", MatchPathPrefix: "/",
				RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site",
			})
		})

	require.PanicsWithError(t, "portal entry 9 not found type=APPLICATION code=OPERATION_FAILED",
		func() {
			core.Save(PortalRule{
				Name: "other", EntryId: 9, MatchPathPrefix: "/",
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
	require.Equal(t, 1, updated.EntryId)
	// A route transition must clear old fields and provide the new required fields.
	updated = service.Update(17, PortalRuleUpdate{RouteType: new(PortalRuleRouteTypePermanentRedirect), RouteSiteName: new(""), RoutePathPrefix: new(""), RouteRedirectionPattern: new("https://example.com{uri}")})
	require.Equal(t, PortalRuleRouteTypePermanentRedirect, updated.RouteType)
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

// Hub reports the requests two rules match once the Web mount paths decide the
// prefixes, because no write can answer that question: Hub applies a seed before
// the applications register their schemas.
func TestPortalRuleCoreConflictsFollowSiteMountPaths(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80, Enabled: true})
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{
		1: {Id: 1, Name: "web", EntryId: 1, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site", Enabled: true},
		2: {Id: 2, Name: "fixed", EntryId: 1, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "fixed-site", Enabled: true},
	}}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	// Two webs that declare no mount path both match the paths their rules
	// declare, so the rules Hub would publish match the same request.
	core.PortalSiteRepo = &portalSiteRepoSpy{entries: map[int]*PortalSite{
		1: {Id: 1, Name: "web-site", Type: PortalSiteTypeWEBGW, Enabled: true},
		2: {Id: 2, Name: "fixed-site", Type: PortalSiteTypeWEBGW, Enabled: true},
	}}
	conflicts := core.Conflicts()
	require.Len(t, conflicts, 1)
	assert.Equal(t, "fixed", conflicts[0].Rule)
	assert.Equal(t, "web", conflicts[0].Conflict)
	assert.Equal(t, "http://*:80", conflicts[0].MatchText())

	// A Web mount path decides the prefix of the rules that target it, so two
	// webs Hub serves at different paths no longer match the same request.
	core.PortalSiteRepo = &portalSiteRepoSpy{entries: map[int]*PortalSite{
		1: {Id: 1, Name: "web-site", Type: PortalSiteTypeWEBGW, WebMountPath: "/", Enabled: true},
		2: {Id: 2, Name: "fixed-site", Type: PortalSiteTypeWEBGW, WebMountPath: "/fixed", Enabled: true},
	}}
	require.Empty(t, core.Conflicts())
}

// A disabled rule, entry, or site is not part of what Portal serves, so it never
// conflicts with a rule Hub publishes.
func TestPortalRuleCoreConflictsSkipDisabledEntities(t *testing.T) {
	disabledRule := &PortalRule{Id: 1, Name: "web", EntryId: 1, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "web-site", Enabled: false}
	enabledRule := &PortalRule{Id: 2, Name: "fixed", EntryId: 1, RouteType: PortalRuleRouteTypeSite, RouteSiteName: "fixed-site", Enabled: true}
	repo := &entryRuleRepoSpy{rules: map[int]*PortalRule{1: disabledRule, 2: enabledRule}}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80, Enabled: true}))
	core.PortalSiteRepo = &portalSiteRepoSpy{entries: map[int]*PortalSite{
		1: {Id: 1, Name: "web-site", Type: PortalSiteTypeWEBGW, Enabled: true},
		2: {Id: 2, Name: "fixed-site", Type: PortalSiteTypeWEBGW, Enabled: false},
	}}
	assert.Empty(t, core.Conflicts())

	disabledRule.Enabled = true
	enabledRule.Enabled = true
	core.PortalSiteRepo = &portalSiteRepoSpy{entries: map[int]*PortalSite{
		1: {Id: 1, Name: "web-site", Type: PortalSiteTypeWEBGW, Enabled: true},
		2: {Id: 2, Name: "fixed-site", Type: PortalSiteTypeWEBGW, Enabled: true},
	}}
	assert.Len(t, core.Conflicts(), 1)
}

func TestPortalRuleCoreCreateKeepsEntry(t *testing.T) {
	entryRepo := newPortalEntryRepoSpy(&PortalEntry{Id: 1, Scheme: "http", Port: 80})
	repo := &entryRuleRepoSpy{}
	core := newPortalRuleCoreWithEntriesForTest(repo, nil, entryRepo)

	created := core.Create(PortalRule{
		Name: "api", EntryId: 1, MatchPathPrefix: "/api",
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
	return newPortalRuleCoreWithEntriesForTest(ruleRepo, certRepo, newTestPortalRuleEntryRepo())
}

func newPortalRuleCoreWithEntriesForTest(ruleRepo PortalRuleRepo, certRepo PortalCertRepo, entryRepo PortalEntryRepo) *PortalRuleCore {
	if certRepo == nil {
		certRepo = newTestPortalCertRepo()
	}
	return &PortalRuleCore{
		PortalRuleRepo:  ruleRepo,
		PortalCertRepo:  certRepo,
		PortalEntryCore: newPortalEntryCoreForTest(ruleRepo, entryRepo, nil),
		PortalSiteRepo:  &portalSiteRepoSpy{},
	}
}
