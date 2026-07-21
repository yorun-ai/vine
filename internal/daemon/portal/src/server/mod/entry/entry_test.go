package entry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

type _TestSite struct {
	name    string
	request *http.Request
	remote  string
}

func (s *_TestSite) Name() string {
	return s.name
}

func (s *_TestSite) Serve(ctx *spec.Context) {
	s.request = ctx.Request
	s.remote = ctx.RemoteAddr
}

func (s *_TestSite) Update(config redised.PortalSite) bool {
	return false
}

func (s *_TestSite) Stop() {
}

func TestEntryRouteMatchesAttachedRules(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTPS,
			host:            "demo.local",
			port:            8443,
			pathPrefix:      "/admin",
			redirectionSite: target,
		}},
	}

	rule, ok := entry.route(newTestRequest("https://demo.local:8443/admin/users"))

	assert.True(t, ok)
	assert.Same(t, target, rule.redirectionSite)
}

func TestEntryRouteMatchesEmptyHostWithIPRequestHost(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTPS,
			host:            "",
			port:            8443,
			pathPrefix:      "/admin",
			redirectionSite: target,
		}},
	}

	rule, ok := entry.route(newTestRequest("https://127.0.0.1:8443/admin/users"))

	assert.True(t, ok)
	assert.Same(t, target, rule.redirectionSite)
}

func TestEntryRouteMatchesEmptyHostWithDomainRequestHost(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTPS,
			host:            "",
			port:            8443,
			pathPrefix:      "/admin",
			redirectionSite: target,
		}},
	}

	rule, ok := entry.route(newTestRequest("https://demo.local:8443/admin/users"))

	assert.True(t, ok)
	assert.Same(t, target, rule.redirectionSite)
}

func TestEntryRouteDoesNotMatchPartialPathPrefix(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTPS,
			host:            "demo.local",
			port:            8443,
			pathPrefix:      "/admin",
			redirectionSite: target,
		}},
	}

	_, ok := entry.route(newTestRequest("https://demo.local:8443/admin2/users"))

	assert.False(t, ok)
}

func TestEntryServesRedirectRule(t *testing.T) {
	entry := &_Entry{
		port: 8080,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTP,
			host:            "demo.local",
			port:            8080,
			pathPrefix:      "/old",
			redirectionSite: site.NewRedirectionSite(true, "https://demo.local/new"),
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local:8080/old/page", nil)
	entry.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusPermanentRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/new", recorder.Header().Get("Location"))
	assert.NotContains(t, recorder.Body.String(), "404")
}

func TestEntryServesRedirectRuleWithoutTrimmedPathPrefix(t *testing.T) {
	entry := &_Entry{
		port: 8080,
		rules: []*_Rule{{
			scheme:          spec.SchemeHTTP,
			host:            "demo.local",
			port:            8080,
			pathPrefix:      "/old",
			redirectionSite: site.NewRedirectionSite(false, "https://demo.local{uri}"),
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local:8080/old/page", nil)
	entry.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/old/page", recorder.Header().Get("Location"))
}

func TestRuleTrimsPathPrefix(t *testing.T) {
	rule := _Rule{pathPrefix: "/admin"}
	request := httptest.NewRequest(http.MethodGet, "https://demo.local:8443/admin/users", nil)

	trimmed := rule.trimPathPrefix(request)

	assert.Equal(t, "/users", trimmed.URL.Path)
	assert.Equal(t, "/admin/users", request.URL.Path)
}

func TestRuleTrimsPathPrefixToRoot(t *testing.T) {
	rule := _Rule{pathPrefix: "/admin"}
	request := httptest.NewRequest(http.MethodGet, "https://demo.local:8443/admin", nil)

	trimmed := rule.trimPathPrefix(request)

	assert.Equal(t, "/", trimmed.URL.Path)
	assert.Equal(t, "/admin", request.URL.Path)
}

func TestEntryResetRulesClearsAttachedRules(t *testing.T) {
	entry := &_Entry{
		rules: []*_Rule{{
			redirectionSite: &_TestSite{name: "admin@demo.app"},
		}},
	}

	entry.SetOrUpdateRules(nil)

	assert.Empty(t, entry.rules)
}

func newTestRequest(target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil)
}
