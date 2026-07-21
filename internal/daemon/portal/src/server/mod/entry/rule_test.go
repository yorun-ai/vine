package entry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

func TestNewRuleBuildsSiteRule(t *testing.T) {
	siteManager := newTestSiteManager("admin@demo.app")

	rule, ok := newRule(redised.PortalRule{
		Name:       "admin",
		Scheme:     string(spec.SchemeHTTPS),
		Host:       "demo.local",
		Port:       8443,
		PathPrefix: "/admin",
		TargetType: targetTypeSite,
		SiteName:   "admin@demo.app",
	}, siteManager)

	assert.True(t, ok)
	assert.Equal(t, "admin", rule.name)
	assert.Equal(t, spec.SchemeHTTPS, rule.scheme)
	assert.Equal(t, "demo.local", rule.host)
	assert.Equal(t, 8443, rule.port)
	assert.Equal(t, "/admin", rule.pathPrefix)
	assert.Same(t, siteManager, rule.siteManager)
	assert.Equal(t, "admin@demo.app", rule.targetSiteName)
}

func TestNewRuleBuildsRedirectRule(t *testing.T) {
	rule, ok := newRule(redised.PortalRule{
		Name:               "redirect",
		Scheme:             string(spec.SchemeHTTP),
		Host:               "demo.local",
		Port:               8080,
		PathPrefix:         "/old",
		TargetType:         targetTypePermanentRedirect,
		RedirectionPattern: "https://demo.local/new",
	}, newTestSiteManager())

	assert.True(t, ok)
	assert.Equal(t, "redirect", rule.name)
	assert.Equal(t, spec.SchemeHTTP, rule.scheme)
	assert.Equal(t, "demo.local", rule.host)
	assert.Equal(t, 8080, rule.port)
	assert.Equal(t, "/old", rule.pathPrefix)
	assert.Equal(t, "redirection", rule.redirectionSite.Name())
}

func TestNewRuleBuildsSiteRuleWithMissingSiteName(t *testing.T) {
	rule, ok := newRule(redised.PortalRule{
		Name:       "admin",
		Scheme:     string(spec.SchemeHTTPS),
		TargetType: targetTypeSite,
		SiteName:   "missing@demo.app",
	}, newTestSiteManager())

	assert.True(t, ok)
	assert.Equal(t, "missing@demo.app", rule.targetSiteName)
}

func TestNewRuleSkipsUnknownTargetType(t *testing.T) {
	rule, ok := newRule(redised.PortalRule{
		Name:       "broken",
		Scheme:     "tcp",
		TargetType: "BROKEN",
		SiteName:   "admin@demo.app",
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

func TestPortalRuleHost(t *testing.T) {
	assert.Empty(t, entryRuleHost(""))
	assert.Equal(t, "demo.local", entryRuleHost("demo.local"))
	assert.Equal(t, "127.0.0.1", entryRuleHost("127.0.0.1"))
}
