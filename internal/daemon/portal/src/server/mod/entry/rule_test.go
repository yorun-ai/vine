package entry

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

func TestNewRuleBuildsSiteRule(t *testing.T) {
	siteManager := newTestSiteManager("admin@demo.app")

	rule, ok := newRule(watched.PortalRule{
		Name:                    "admin",
		MatchScheme:             string(spec.SchemeHTTPS),
		MatchHost:               "demo.local",
		MatchPort:               8443,
		ResolvedMatchPathPrefix: "/admin",
		RouteType:               routeTypeSite,
		RouteSiteName:           "admin@demo.app",
	}, siteManager)

	assert.True(t, ok)
	assert.Equal(t, "admin", rule.name)
	assert.Equal(t, spec.SchemeHTTPS, rule.matchScheme)
	assert.Equal(t, "demo.local", rule.matchHost)
	assert.Equal(t, 8443, rule.matchPort)
	assert.Equal(t, "/admin", rule.matchPathPrefix)
	assert.Same(t, siteManager, rule.siteManager)
	assert.Equal(t, "admin@demo.app", rule.routeSiteName)
}

func TestNewRuleBuildsRedirectRule(t *testing.T) {
	rule, ok := newRule(watched.PortalRule{
		Name:                    "redirect",
		MatchScheme:             string(spec.SchemeHTTP),
		MatchHost:               "demo.local",
		MatchPort:               8080,
		ResolvedMatchPathPrefix: "/old",
		RouteType:               routeTypePermanentRedirect,
		RouteRedirectionPattern: "https://demo.local/new",
	}, newTestSiteManager())

	assert.True(t, ok)
	assert.Equal(t, "redirect", rule.name)
	assert.Equal(t, spec.SchemeHTTP, rule.matchScheme)
	assert.Equal(t, "demo.local", rule.matchHost)
	assert.Equal(t, 8080, rule.matchPort)
	assert.Equal(t, "/old", rule.matchPathPrefix)
	assert.Equal(t, "redirection", rule.redirectionSite.Name())
}

func TestNewRuleBuildsSiteRuleWithMissingSiteName(t *testing.T) {
	rule, ok := newRule(watched.PortalRule{
		Name:          "admin",
		MatchScheme:   string(spec.SchemeHTTPS),
		RouteType:     routeTypeSite,
		RouteSiteName: "missing@demo.app",
	}, newTestSiteManager())

	assert.True(t, ok)
	assert.Equal(t, "missing@demo.app", rule.routeSiteName)
}

func TestNewRuleSkipsUnknownTargetType(t *testing.T) {
	rule, ok := newRule(watched.PortalRule{
		Name:          "broken",
		MatchScheme:   "tcp",
		RouteType:     "BROKEN",
		RouteSiteName: "admin@demo.app",
	}, newTestSiteManager("admin@demo.app"))

	assert.False(t, ok)
	assert.Nil(t, rule)
}

func TestPortalRulePortDefaultsByScheme(t *testing.T) {
	assert.Equal(t, defaultHTTPEntryPort, entryRulePort(spec.SchemeHTTP, 0))
	assert.Equal(t, defaultHTTPSEntryPort, entryRulePort(spec.SchemeHTTPS, 0))
}

func TestPortalRulePortPanicsOnUnsupportedScheme(t *testing.T) {
	assert.Panics(t, func() {
		entryRulePort(spec.Scheme("tcp"), 0)
	})
}

func TestRuleRewritePath(t *testing.T) {
	for _, test := range []struct{ prefix, target, request, want string }{
		{"/api", "", "/api/users?x=1", "/users?x=1"},
		{"/api", "/internal", "/api/users?x=%2F", "/internal/users?x=%2F"},
		{"/api", "/api", "/api/users", "/api/users"},
		{"/", "/internal", "/users", "/internal/users"},
		{"", "/internal", "/users", "/internal/users"},
		{"/api", "/internal", "/api", "/internal"},
		{"/api", "/internal", "/api/", "/internal/"},
		{"/api", "", "/api", "/"},
		{"/api", "/internal", "/%61pi/a%2Fb/%25/%E4%B8%AD/?q=a+b", "/internal/a%2Fb/%25/%E4%B8%AD/?q=a+b"},
		{"/api", "/base%20path", "/api//a/../b", "/base%20path//a/../b"},
		{"/", "", "/a%2Fb/", "/a%2Fb/"},
	} {
		t.Run(test.prefix+"->"+test.target+test.request, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.request, strings.NewReader("body"))
			request.Header.Set("X-Test", "keep")
			rule := _Rule{matchPathPrefix: test.prefix, routePathPrefix: test.target}
			require.True(t, rule.Matches(request))
			next := rule.rewritePath(request)
			assert.Equal(t, test.want, next.URL.RequestURI())
			assert.Equal(t, test.request, request.URL.RequestURI())
			assert.Equal(t, request.Context(), next.Context())
			assert.Equal(t, http.MethodPost, next.Method)
			assert.Equal(t, "keep", next.Header.Get("X-Test"))
			assert.Equal(t, request.Body, next.Body)
		})
	}
}

func TestWildcardHostMatchesOneLabel(t *testing.T) {
	rule := _Rule{matchHost: "*.example.com"}
	for _, host := range []string{"a.example.com", "b.example.com:8080", "A.EXAMPLE.COM"} {
		assert.True(t, rule.Matches(httptest.NewRequest("GET", "http://"+host+"/", nil)), host)
	}
	for _, host := range []string{"example.com", "a.b.example.com", ".example.com", "evil-example.com", "a.example.com.evil"} {
		assert.False(t, rule.matchesHost(host), host)
	}
}

func TestWildcardRuleRejectsRpcAndRedirect(t *testing.T) {
	rule, ok := newRule(watched.PortalRule{Name: "wildcard", MatchScheme: "http", MatchHost: "*.example.com", RouteType: "SITE", RouteSiteName: "rpc"}, newTestSiteManager("rpc"))
	require.True(t, ok)
	w := httptest.NewRecorder()
	rule.Serve(&spec.Context{Request: httptest.NewRequest("GET", "http://a.example.com/inspect", nil), ResponseWriter: w})
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "WEBGW")
	_, ok = newRule(watched.PortalRule{Name: "redirect", MatchScheme: "http", MatchHost: "*.example.com", RouteType: "TEMPORARY_REDIRECT"}, nil)
	assert.False(t, ok)
}
