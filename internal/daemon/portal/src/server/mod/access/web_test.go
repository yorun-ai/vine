package access

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestAuthWebUsesAnonymousActorWithoutAuthorization(t *testing.T) {
	access := testManager(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)

	ok := access.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: "missing.Actor"}, request, recorder))

	require.True(t, ok)
	actor, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
	require.NoError(t, err)
	assert.True(t, actor.IsAnonymous())
}

func TestAuthWebParsesAuthorization(t *testing.T) {
	registerTestActorInfo()
	authEndpoint := registerTestAuthService(t, http.StatusOK, "OK", `{"userId":"u1"}`)
	access := testManager(testAuthValues(authEndpoint))
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
}

func TestAuthWebRejectsBadAuthorizationAsUnauthorized(t *testing.T) {
	access := testManager(testAuthValues(""))
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
	access := testManager(testAuthValues(authEndpoint))
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
	access := testManager(testAuthValues(authEndpoint))
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

func TestAuthWebPreservesNativeAuthorizationWithoutActorAuth(t *testing.T) {
	schema := &watched.SchemaActor{SkelName: "demo.NativeActor"}
	manager := testManager(map[string]string{
		watched.FormatSchemaActorKey(schema.SkelName): vcode.MustMarshalJsonS(schema),
	})
	for _, header := range []string{"", "Bearer native-token", "Basic dXNlcjpwdw==", "custom native-credential"} {
		t.Run(header, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
			setTestWebRequestHeaders(t, request)
			request.Header.Set(headerAuthorization, header)
			request.Header.Set(webspec.HeaderWebActor, "untrusted actor metadata")
			response := httptest.NewRecorder()
			require.True(t, manager.AuthWeb(testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: schema.SkelName, ActorVia: "client"}, request, response)))
			require.Equal(t, header, request.Header.Get(headerAuthorization))
			actor, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
			require.NoError(t, err)
			require.True(t, actor.IsAnonymous())
		})
	}
}

func TestAuthWebRejectsUnknownActorWithAuthorization(t *testing.T) {
	manager := testManager(nil)
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
	schema := &watched.SchemaActor{SkelName: "demo.NativeActor"}
	manager := testManager(map[string]string{
		watched.FormatSchemaActorKey(schema.SkelName): vcode.MustMarshalJsonS(schema),
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
		operation := testWebAuthContext(t, watched.PortalActorVia{ActorSkelName: schema.SkelName, ActorVia: "client"}, r, w)
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
