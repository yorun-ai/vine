package access

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func TestAuthWebUsesAnonymousActorWithoutAuthorization(t *testing.T) {
	access := testManager(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)

	ok := access.AuthWeb(testWebAuthContext(t, redised.PortalActorVia{ActorSkelName: "missing.Actor"}, request, recorder))

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

	ok := access.AuthWeb(testWebAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
	actor, err := meta.DecodeActorFromBase64(request.Header.Get(webspec.HeaderWebActor))
	require.NoError(t, err)
	assert.True(t, actor.IsAuthenticated())
	assert.JSONEq(t, `{"userId":"u1"}`, actor.RawInfo())
}

func TestAuthWebRejectsBadAuthorization(t *testing.T) {
	access := testManager(testAuthValues(""))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, unknown value")

	ok := access.AuthWeb(testWebAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.False(t, ok)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "bad credential")
}

func TestAuthWebMapsAuthServiceStatus(t *testing.T) {
	authEndpoint := registerTestAuthService(t, http.StatusOK, "UNAUTHORIZED", `null`)
	access := testManager(testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	setTestWebRequestHeaders(t, request)
	request.Header.Set(headerAuthorization, "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AuthWeb(testWebAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

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

	ok := access.AuthWeb(testWebAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
}

func setTestWebRequestHeaders(t *testing.T, request *http.Request) {
	t.Helper()

	webspec.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	initiator, err := meta.NewInitiator("demo.client", "0.0.0", "123e4567-e89b-12d3-a456-426614174001", "curl/8.0", "192.0.2.1")
	require.NoError(t, err)
	request.Header.Set(webspec.HeaderWebInitiator, meta.EncodeInitiatorToBase64(initiator))
}

func testWebAuthContext(t *testing.T, actorVia redised.PortalActorVia, request *http.Request, response http.ResponseWriter) *WebOperation {
	t.Helper()

	trace, err := webspec.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)
	initiator, err := meta.DecodeInitiatorFromBase64(request.Header.Get(webspec.HeaderWebInitiator))
	require.NoError(t, err)
	return &WebOperation{
		Auther: Auther{
			Request:   request,
			Response:  response,
			Trace:     trace,
			Initiator: initiator,
		},
		ActorVia: actorVia,
	}
}
