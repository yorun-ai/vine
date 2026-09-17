package entry

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vcode"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/util/httputil"
)

type _TestSite struct {
	name string
}

func (s *_TestSite) Name() string {
	return s.name
}

func (s *_TestSite) Serve(ctx *spec.Context) {
}

func (s *_TestSite) Update(config watched.PortalSite) bool {
	return false
}

func (s *_TestSite) Stop() {
}

func TestEntryRouteMatchesAttachedRules(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTPS,
			matchHost:       "demo.local",
			matchPort:       8443,
			matchPathPrefix: "/admin",
			redirectionSite: target,
		}},
	}

	rule, ok := entry.route(newTestRequest("https://demo.local:8443/admin/users"))

	assert.True(t, ok)
	assert.Same(t, target, rule.redirectionSite)
}

func TestEntryRouteMatchesEmptyHostWithAnyRequestHost(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTPS,
			matchHost:       "",
			matchPort:       8443,
			matchPathPrefix: "/admin",
			redirectionSite: target,
		}},
	}

	for _, requestHost := range []string{"127.0.0.1", "demo.local"} {
		t.Run(requestHost, func(t *testing.T) {
			rule, ok := entry.route(newTestRequest("https://" + requestHost + ":8443/admin/users"))

			assert.True(t, ok)
			assert.Same(t, target, rule.redirectionSite)
		})
	}
}

func TestEntryRouteDoesNotMatchPartialPathPrefix(t *testing.T) {
	target := &_TestSite{name: "admin@demo.app"}
	entry := &_Entry{
		port: 8443,
		rules: []*_Rule{{
			matchScheme:     spec.SchemeHTTPS,
			matchHost:       "demo.local",
			matchPort:       8443,
			matchPathPrefix: "/admin",
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
			matchScheme:     spec.SchemeHTTP,
			matchHost:       "demo.local",
			matchPort:       8080,
			matchPathPrefix: "/old",
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
			matchScheme:     spec.SchemeHTTP,
			matchHost:       "demo.local",
			matchPort:       8080,
			matchPathPrefix: "/old",
			redirectionSite: site.NewRedirectionSite(false, "https://demo.local{uri}"),
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local:8080/old/page", nil)
	entry.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/old/page", recorder.Header().Get("Location"))
}

func TestNewEntryHTTPServerAppliesConnectionLimits(t *testing.T) {
	entry := newEntry(spec.SchemeHTTP, 8080, nil)

	server := newEntryHTTPServer("0.0.0.0:8080", entry)

	assert.Equal(t, entryReadHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, entryIdleTimeout, server.IdleTimeout)
	assert.Equal(t, entryMaxHeaderBytes, server.MaxHeaderBytes)
	assert.Equal(t, httputil.DefaultMaxHeaderValueCount, server.MaxHeaderValueCount)
	assert.Zero(t, server.ReadTimeout)
	assert.Zero(t, server.WriteTimeout)
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

func TestEntryTargetPathForwardingAndUpdate(t *testing.T) {
	for _, inproc := range []bool{false, true} {
		t.Run(fmt.Sprintf("inproc=%t", inproc), func(t *testing.T) {
			backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("X-Received-Method", r.Method)
				w.Header().Set("X-Received-Body", string(body))
				_, _ = io.WriteString(w, r.URL.RequestURI())
			})
			var endpoint string
			if inproc {
				endpoint = "link+inproc://vine/entry-target-path-test"
				ingressinproc.Register(endpoint, backend)
				t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
			} else {
				server := httptest.NewUnstartedServer(backend)
				server.Config.Protocols = httputil.UnencryptedHTTP2Protocols()
				server.Start()
				t.Cleanup(server.Close)
				endpoint = server.URL
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			values := map[string]string{
				watched.FormatPortalSiteKey("web"): vcode.MustMarshalJsonS(watched.PortalSite{
					Name: "web", Type: "WEBGW", WebgwConfig: &watched.PortalWebgwConfig{WebName: "demo.Web"},
				}),
				watched.FormatWebRegistrationKey("demo.Web", "demo", "instance"): vcode.MustMarshalJsonS(watched.WebRegistration{
					Endpoint: endpoint + "/web/proxy/in/instance/demo.Web", WebSkelName: "demo.Web", AppName: "demo", AppInstanceId: "instance",
				}),
			}
			endpoints := &epmgr.Manager{Context: ctx, Watch: hubwatch.NewTestClient(values)}
			endpoints.DIInit()
			sites := &site.Manager{Context: ctx, Watch: hubwatch.NewTestClient(values), Epmgr: endpoints, Access: new(access.Access)}
			sites.DIInit()
			entry := newEntry(spec.SchemeHTTP, 80, nil)
			public := httptest.NewServer(entry)
			t.Cleanup(public.Close)
			for _, targetPath := range []string{"/internal", "/v2", ""} {
				rule, ok := newRule(watched.PortalRule{Name: "rule", MatchScheme: "http", ResolvedMatchPathPrefix: "/api", ResolvedRoutePathPrefix: targetPath, RouteType: "SITE", RouteSiteName: "web"}, sites)
				require.True(t, ok)
				entry.SetOrUpdateRules([]*_Rule{rule})
				request, err := http.NewRequest(http.MethodPost, public.URL+"/api/a%2Fb/?q=%2F", strings.NewReader("payload"))
				require.NoError(t, err)
				response, err := public.Client().Do(request)
				require.NoError(t, err)
				body, err := io.ReadAll(response.Body)
				_ = response.Body.Close()
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode, string(body))
				assert.Equal(t, "/web/proxy/in/instance/demo.Web"+targetPath+"/a%2Fb/?q=%2F", string(body))
				assert.Equal(t, "POST", response.Header.Get("X-Received-Method"))
				assert.Equal(t, "payload", response.Header.Get("X-Received-Body"))
			}
		})
	}
}

func TestEntryTargetPathDispatchesWithinRpcGateway(t *testing.T) {
	rule, ok := newRule(watched.PortalRule{Name: "rpc", MatchScheme: "http", ResolvedMatchPathPrefix: "/api", ResolvedRoutePathPrefix: "/inspect", RouteType: "SITE", RouteSiteName: "rpc"}, newTestSiteManager("rpc"))
	require.True(t, ok)
	entry := newEntry(spec.SchemeHTTP, 80, nil)
	entry.SetOrUpdateRules([]*_Rule{rule})
	response := httptest.NewRecorder()
	entry.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api", nil))
	assert.Contains(t, response.Body.String(), "rpcgw inspect is not implemented")
	assert.NotContains(t, response.Body.String(), "rpcgw path is not found")
}
