package access

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vcode"
)

func TestAccessAllowRpcParsesTargetRpc(t *testing.T) {
	registerTestActorInfo()
	authEndpoint := registerTestAuthService(t, http.StatusOK, "OK", `{"userId":"u1"}`)
	access := testManager(testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    redised.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	})

	require.True(t, ok)

	actor, err := meta.DecodeActorFromBase64(request.Header.Get(rpchttp.HeaderRpcActor))
	require.NoError(t, err)
	assert.Equal(t, meta.ActorTypeAuthenticated, actor.Type())
	assert.JSONEq(t, `{"userId":"u1"}`, actor.RawInfo())
}

func TestAccessAllowRpcReturnsServiceUnavailableWhenAuthServiceHasNoEndpoint(t *testing.T) {
	access := testManager(testAuthValues(""))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    redised.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	})

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ServiceUnavailable, "auth service is unavailable")
}

func TestAccessAllowRpcMapsAuthServiceStatus(t *testing.T) {
	authEndpoint := registerTestAuthService(t, http.StatusOK, "UNAUTHORIZED", `null`)
	access := testManager(testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.Unauthorized, "auth failed")
}

func TestAccessAllowRpcSendsCredentialToAuthService(t *testing.T) {
	authEndpoint := "link+inproc://vine/auth-rpc-credential-test"
	ingressinproc.Register(authEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"key1":"token123"`) || !strings.Contains(string(body), `"key2":"dXNlcjpwd2Q="`) {
			t.Fatalf("unexpected auth request body: %s", string(body))
		}
		writeTestAuthResponse(w, r, http.StatusOK, "OK", `{"userId":"u1"}`)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(authEndpoint) })
	access := testManager(testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
}

func TestAccessAllowRpcForwardsTimeoutToAuthService(t *testing.T) {
	authEndpoint := "link+inproc://vine/auth-rpc-options-test"
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
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil).WithContext(ctx)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	require.True(t, ok)
}

func TestAccessAllowRpcCreatesTraceChildForAuthService(t *testing.T) {
	var gotTrace meta.Trace
	authEndpoint := "link+inproc://vine/auth-rpc-trace-test"
	ingressinproc.Register(authEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotTrace, err = rpchttp.DecodeTraceFromHeader(r.Header)
		require.NoError(t, err)
		writeTestAuthResponse(w, r, http.StatusOK, "OK", `{"userId":"u1"}`)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(authEndpoint) })
	access := testManager(testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")
	baseTrace, err := rpchttp.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    redised.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	})

	require.True(t, ok)
	require.NotNil(t, gotTrace)
	assert.Equal(t, baseTrace.Id(), gotTrace.Id())
	assert.NotEqual(t, baseTrace.Span(), gotTrace.Span())
}

func testAuthValues(authEndpoint string) map[string]string {
	values := map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorSchema()),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skel.MethodSchema{
				{SkelName: "Get"},
			},
		}),
	}
	if authEndpoint != "" {
		values[redised.FormatRpcServiceRegistrationKey("demo.UserActorAuthService", "demo.app", "123e4567-e89b-12d3-a456-426614174011")] = vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      authEndpoint,
			ServiceName:   "demo.UserActorAuthService",
			AppName:       "demo.app",
			AppInstanceId: "123e4567-e89b-12d3-a456-426614174011",
		})
	}
	return values
}

func testAuthActorSchema() redised.SchemaActor {
	return redised.SchemaActor{
		SkelName:       "demo.UserActor",
		AuthEnabled:    true,
		AuthCredential: testCredentialSchema(),
		AuthInfo:       &skel.DataSchema{SkelName: "demo.UserInfo"},
		AuthService:    testAuthServiceSchema(),
		AuthMethod:     testAuthMethodSchema(),
	}
}

func testUserActorAudiences() []*skel.ActorAudienceSchema {
	return []*skel.ActorAudienceSchema{{SkelName: "demo.UserActor"}}
}

func testAuthServiceSchema() *skel.ServiceSchema {
	return &skel.ServiceSchema{
		SkelName: "demo.UserActorAuthService",
		Methods: []*skel.MethodSchema{
			testAuthMethodSchema(),
		},
	}
}

func testAuthMethodSchema() *skel.MethodSchema {
	return &skel.MethodSchema{
		SkelName: "auth",
		Arguments: []*skel.MemberSchema{
			{Name: "credential"},
		},
		ResultType: &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "demo.UserInfo"},
	}
}

func registerTestAuthService(t *testing.T, status int, rpcStatus string, result string) string {
	t.Helper()

	endpoint := "link+inproc://vine/auth-rpc-test-" + strings.ToLower(rpcStatus)
	ingressinproc.Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestAuthResponse(w, r, status, rpcStatus, result)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
	return endpoint
}

func writeTestAuthResponse(w http.ResponseWriter, r *http.Request, status int, rpcStatus string, result string) {
	w.Header().Set(rpchttp.HeaderContentType, "application/vrpc+json")
	w.Header().Set(rpchttp.HeaderRpcStatus, rpcStatus)
	w.Header().Set(rpchttp.HeaderRpcServer, "name=demo.auth,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174012")
	w.WriteHeader(status)
	if rpcStatus == "OK" {
		_, _ = w.Write([]byte(`{"result":` + result + `}`))
		return
	}
	_, _ = w.Write(vcode.MustMarshalJson(map[string]any{
		"result": nil,
		"error": map[string]string{
			"code":    rpcStatus,
			"message": "auth failed",
		},
	}))
}

type _TestUserInfo struct {
	UserId string `json:"userId"`
}

var registerTestActorInfoOnce sync.Once

func registerTestActorInfo() {
	registerTestActorInfoOnce.Do(func() {
		meta.RegisterActor(meta.ActorSpec{
			Name:         "UserActor",
			SkelName:     "demo.UserActor",
			InfoSkelName: "demo.UserInfo",
			InfoType:     reflect.TypeFor[*_TestUserInfo](),
		})
	})
}

func TestAccessAllowRpcRejectsMissingServiceSchema(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorSchema()),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ServiceUnavailable, "rpc service schema is not found")
}

func TestAccessAllowRpcRejectsMissingMethodSchema(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorSchema()),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skel.MethodSchema{
				{SkelName: "List"},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.NotFound, "rpc method schema is not found")
}

func TestAccessAllowRpcRejectsMissingActorSchema(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skel.MethodSchema{
				{SkelName: "Get"},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ClientForbidden, "not allowed")
}

func TestAccessAllowRpcRejectsActorWithoutCredentialSchema(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName:    "demo.UserActor",
			AuthEnabled: true,
			AuthInfo:    &skel.DataSchema{SkelName: "demo.UserInfo"},
		}),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skel.MethodSchema{
				{SkelName: "Get"},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	assert.Panics(t, func() {
		access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
	})
}

func TestAccessAllowRpcRejectsActorWithoutInfoSchema(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName:       "demo.UserActor",
			AuthEnabled:    true,
			AuthCredential: &skel.DataSchema{SkelName: "demo.UserCredential"},
		}),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skel.MethodSchema{
				{SkelName: "Get"},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	assert.Panics(t, func() {
		access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
	})
}

func TestRpcAccessOperationParseAuthModeUsesMethodMode(t *testing.T) {
	ctx := &RpcOperation{
		serviceSchema: &skel.ServiceSchema{AuthMode: skel.AuthModeNoAuth},
		methodSchema:  &skel.MethodSchema{AuthMode: skel.AuthModeAuth},
	}

	assert.Equal(t, skel.AuthModeAuth, ctx.authMode())
}

func TestRpcAccessOperationParseAuthModeFallsBackToServiceMode(t *testing.T) {
	ctx := &RpcOperation{
		serviceSchema: &skel.ServiceSchema{AuthMode: skel.AuthModeNoAuth},
		methodSchema:  &skel.MethodSchema{AuthMode: skel.AuthModeUnset},
	}

	assert.Equal(t, skel.AuthModeNoAuth, ctx.authMode())
}

func TestRpcAccessOperationParseAuthModeDefaultsToAuth(t *testing.T) {
	ctx := &RpcOperation{
		serviceSchema: &skel.ServiceSchema{AuthMode: skel.AuthModeUnset},
		methodSchema:  &skel.MethodSchema{AuthMode: skel.AuthModeUnset},
	}

	assert.Equal(t, skel.AuthModeAuth, ctx.authMode())
}

func TestAccessAllowRpcInjectsActorAndServiceSchemas(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorSchema()),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			AuthMode:  skel.AuthModeNoAuth,
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeNoAuth},
			},
		}),
	})
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	ctx := &RpcOperation{
		Auther:      authOperationForTest(t, request, httptest.NewRecorder()),
		ActorVia:    redised.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	}

	require.True(t, access.AllowRpc(ctx))

	assert.Equal(t, "demo.UserActor", ctx.actorSchema.SkelName)
	assert.Equal(t, "demo.UserService", ctx.serviceSchema.SkelName)
}

func TestAccessAllowRpcRejectsDifferentActorVia(t *testing.T) {
	access := testManager(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorSchema()),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName: "demo.UserService",
			AuthMode: skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{SkelName: "demo.UserActor", Via: skel.ActorViaAgent},
			},
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeNoAuth},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{
		ActorSkelName: "demo.UserActor",
		ActorVia:      "client",
	}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ClientForbidden, "rpc service does not allow actor via")
}

func TestRpcAccessOperationParseCredentialWritesBadCredentialError(t *testing.T) {
	response := httptest.NewRecorder()
	ctx := &RpcOperation{
		Auther: Auther{
			actorSchema: &skel.ActorSchema{
				AuthCredential: testCredentialSchema(),
			},
			Request:  httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil),
			Response: response,
		},
		Server: testServerApp(),
	}
	ctx.Request.Header.Set("Authorization", "Key1 token123, unknown value")

	succeed := ctx.parseCredential(ctx.writeError)
	assert.False(t, succeed)
	assertRpcAuthError(t, response, ex.ClientForbidden, "bad credential")
}

func testRpcAuthContext(t *testing.T, actorVia redised.PortalActorVia, request *http.Request, response http.ResponseWriter) *RpcOperation {
	t.Helper()

	setTestRequestHeaders(t, request)
	return &RpcOperation{
		Auther:      authOperationForTest(t, request, response),
		Server:      testServerApp(),
		ActorVia:    actorVia,
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	}
}

func authOperationForTest(t *testing.T, request *http.Request, response http.ResponseWriter) Auther {
	t.Helper()

	trace, err := rpchttp.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)
	initiator, err := meta.DecodeInitiatorFromBase64(request.Header.Get(rpchttp.HeaderRpcInitiator))
	require.NoError(t, err)
	return Auther{
		Request:   request,
		Response:  response,
		Trace:     trace,
		Initiator: initiator,
	}
}

func setTestRequestHeaders(t *testing.T, request *http.Request) {
	t.Helper()

	rpchttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	request.Header.Set(rpchttp.HeaderRpcClient, "name=demo.client,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174001")
	initiator, err := meta.NewInitiator("demo.client", "0.0.0", "123e4567-e89b-12d3-a456-426614174001", "curl/8.0", "192.0.2.1")
	require.NoError(t, err)
	request.Header.Set(rpchttp.HeaderRpcInitiator, meta.EncodeInitiatorToBase64(initiator))
}

func assertRpcAuthError(t *testing.T, recorder *httptest.ResponseRecorder, code ex.Code, message string) {
	t.Helper()

	assert.Equal(t, rpchttp.ResponseStatusCode, recorder.Code)
	assert.Equal(t, string(code), recorder.Header().Get(rpchttp.HeaderRpcStatus))
	assert.Contains(t, recorder.Body.String(), message)
}

func testServerApp() meta.App {
	return meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099")
}
