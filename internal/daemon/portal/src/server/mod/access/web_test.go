package access

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestAuthWebRequiresAuthorizationByDefault(t *testing.T) {
	access := testManager(t, testAuthValues(""))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	require.False(t, access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder)))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuthWebParsesAuthorization(t *testing.T) {
	registerTestActorInfo()
	authEndpoint := registerTestAuthService(t, http.StatusOK, "OK", `{"userId":"u1"}`)
	access := testManager(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
	actor, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
	require.NoError(t, err)
	assert.True(t, actor.IsAuthenticated())
	assert.JSONEq(t, `{"userId":"u1"}`, actor.RawInfo())
	require.NotContains(t, request.Header, headerAuthorization)
}

func TestAuthWebRejectsBadAuthorizationAsUnauthorized(t *testing.T) {
	access := testManager(t, testAuthValues(""))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, unknown value")

	ok := access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "bad credential")
}

func TestAuthWebMapsAuthServiceStatus(t *testing.T) {
	authEndpoint := registerTestAuthService(t, http.StatusOK, "UNAUTHORIZED", `null`)
	access := testManager(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "auth failed")
}

func TestAuthWebForwardsTimeoutToAuthService(t *testing.T) {
	authEndpoint := "link+inproc://vine/web-auth-options-test"
	ingressinproc.Register(authEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options, err := rpchttp.DecodeOptionsFromHeader(r.Header)
		require.NoError(t, err)
		require.Positive(t, options.Timeout)
		require.LessOrEqual(t, options.Timeout, 10*time.Second)
		writeTestAuthResponse(w, r, http.StatusOK, "OK", `{"userId":"u1"}`)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(authEndpoint) })
	access := testManager(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil).WithContext(ctx)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
}

func setTestWebRequestHeaders(t *testing.T, request *http.Request) {
	t.Helper()

	webspec.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	initiator, err := meta.NewInitiator("demo.client", "0.0.0", "123e4567-e89b-12d3-a456-426614174001", "curl/8.0", "192.0.2.1")
	require.NoError(t, err)
	request.Header.Set(webspec.HeaderWebInitiator, meta.EncodeInitiatorToBase64(initiator))
}

func testWebAuthContext(t *testing.T, actorVia watched.PortalActorVia, request *http.Request, response http.ResponseWriter) *WebOperation {
	t.Helper()

	trace, err := webspec.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)
	initiator, err := meta.DecodeInitiatorFromBase64(request.Header.Get(webspec.HeaderWebInitiator))
	require.NoError(t, err)
	return &WebOperation{
		Request:   request,
		Response:  response,
		Trace:     trace,
		Initiator: initiator,
		ActorVia:  actorVia,
	}
}

func TestAuthWebOffPreservesNativeAuthorizationWithoutActorAuth(t *testing.T) {
	descriptor := &watched.DescriptorActor{SkelName: "demo.NativeActor"}
	manager := testManager(t, map[string]string{
		watched.FormatDescriptorActorKey(descriptor.SkelName): vcode.MustMarshalJsonS(descriptor),
		watched.FormatDescriptorWebKey("demo.NativeWeb"):      vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.NativeWeb", AuthMode: skeldesc.AuthModeOff}),
	})
	for _, header := range []string{"", "Bearer native-token", "Basic dXNlcjpwdw==", "custom native-credential"} {
		t.Run(header, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
			setTestWebRequestHeaders(t, request)
			request.Header.Set(headerAuthorization, header)
			request.Header.Set(webspec.HeaderWebActor, "untrusted actor metadata")
			response := httptest.NewRecorder()
			operation := testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: descriptor.SkelName, ActorVia: "client"}, request, response)
			operation.WebName = "demo.NativeWeb"
			require.True(t, manager.AuthWeb(operation))
			require.Equal(t, header, request.Header.Get(headerAuthorization))
			actor, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
			require.NoError(t, err)
			require.True(t, actor.IsAnonymous())
		})
	}
}

func TestAuthWebRejectsUnknownActorWithAuthorization(t *testing.T) {
	manager := testManager(t, nil)
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Bearer native-token")
	response := httptest.NewRecorder()
	require.False(t, manager.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "missing.Actor", ActorVia: "client"}, request, response)))
	require.Equal(t, http.StatusForbidden, response.Code)
}

// Exercise the admission decision and forwarded headers through real HTTP
// listeners, including a Web handler that owns its native authentication.
func TestAuthWebNativeCredentialsReachBackend(t *testing.T) {
	descriptor := &watched.DescriptorActor{SkelName: "demo.NativeActor"}
	manager := testManager(t, map[string]string{
		watched.FormatDescriptorActorKey(descriptor.SkelName): vcode.MustMarshalJsonS(descriptor),
		watched.FormatDescriptorWebKey("demo.NativeWeb"):      vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.NativeWeb", AuthMode: skeldesc.AuthModeOff}),
	})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := meta.DecodeActorFromBase64(r.Header.Get(webspec.HeaderWebActor))
		if err != nil || !actor.IsAnonymous() {
			http.Error(w, "unexpected actor metadata", http.StatusInternalServerError)
			return
		}
		if r.Header.Get(headerAuthorization) != "Bearer native-token" {
			http.Error(w, "native authentication failed", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(backend.Close)
	target, err := url.Parse(backend.URL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setTestWebRequestHeaders(t, r)
		operation := testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: descriptor.SkelName, ActorVia: "client"}, r, w)
		operation.WebName = "demo.NativeWeb"
		if manager.AuthWeb(operation) {
			proxy.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(gateway.Close)
	for _, tc := range []struct {
		credential string
		status     int
	}{
		{"Bearer native-token", http.StatusNoContent},
		{"Bearer wrong-token", http.StatusUnauthorized},
	} {
		request, err := http.NewRequest(http.MethodGet, gateway.URL, nil)
		require.NoError(t, err)
		request.Header.Set(headerAuthorization, tc.credential)
		request.Header.Set(webspec.HeaderWebActor, "spoofed actor")
		response, err := gateway.Client().Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, tc.status, response.StatusCode)
	}
}

func TestAuthWebRejectsMissingNamedDescriptor(t *testing.T) {
	manager := testManager(t, nil)
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/", nil)
	response := httptest.NewRecorder()
	operation := &WebOperation{Auther: Auther{Request: request, Response: response}, WebName: "demo.Missing"}
	require.False(t, manager.AuthWeb(operation))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestAuthWebLegacyDescriptorDefaultsToRequired(t *testing.T) {
	registerTestActorInfo()
	endpoint := registerTestAuthService(t, http.StatusOK, "OK", `{"userId":"u1"}`)
	for _, mode := range []skeldesc.AuthMode{skeldesc.AuthModeRequired} {
		for _, enabled := range []bool{false, true} {
			for _, credential := range []string{"", "Bearer malformed", "Key1 token, key2 token"} {
				t.Run(fmt.Sprintf("%s/enabled=%t/%s", mode, enabled, credential), func(t *testing.T) {
					values := testAuthValues(endpoint)
					actor := testAuthActorDescriptor()
					if !enabled {
						actor.Auth = nil
					}
					values[watched.FormatDescriptorActorKey(actor.SkelName)] = vcode.MustMarshalJsonS(actor)
					values[watched.FormatDescriptorWebKey("demo.Web")] = vcode.MustMarshalJsonS(watched.DescriptorWeb{SkelName: "demo.Web", AuthMode: mode})
					manager := testManager(t, values)
					request := httptest.NewRequest(http.MethodGet, "http://demo.local/", nil)
					setTestWebRequestHeaders(t, request)
					if credential != "" {
						request.Header.Set(headerAuthorization, credential)
					}
					response := httptest.NewRecorder()
					operation := testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: actor.SkelName}, request, response)
					operation.WebName = "demo.Web"
					want := enabled && credential == "Key1 token, key2 token"
					require.Equal(t, want, manager.AuthWeb(operation))
					if want {
						require.NotContains(t, request.Header, headerAuthorization)
						admitted, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
						require.NoError(t, err)
						require.True(t, admitted.IsAuthenticated())
					} else if !enabled {
						require.Equal(t, http.StatusForbidden, response.Code)
						require.Contains(t, response.Body.String(), "actor auth not enabled")
					} else {
						require.Equal(t, http.StatusUnauthorized, response.Code)
					}
				})
			}
		}
	}
}
