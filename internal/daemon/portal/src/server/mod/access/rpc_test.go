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
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func TestAccessAllowRpcParsesTargetRpc(t *testing.T) {
	registerTestActorInfo()
	authEndpoint := registerTestAuthService(t, http.StatusOK, "OK", `{"userId":"u1"}`)
	access := newTestAccess(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    watched.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	})

	require.True(t, ok)

	actor, err := meta.DecodeActorFromBase64(request.Header.Get(rpchttp.HeaderRpcActor))
	require.NoError(t, err)
	assert.Equal(t, meta.ActorTypeAuthenticated, actor.Type())
	assert.JSONEq(t, `{"userId":"u1"}`, actor.RawInfo())
}

func TestAccessAllowRpcRejectsOversizedRequestBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.ContentLength = rpchttp.MaxRequestBodyBytes + 1

	ok := new(Access).AllowRpc(&RpcOperation{
		Auther: authOperationForTest(t, request, recorder),
		Server: testServerApp(),
	})

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.InvalidRequest, "rpc request body exceeds")
}

func TestRpcOperationReadRequestBodyClosesAndResetsBody(t *testing.T) {
	originalBody := &requestCloseTrackingBody{Reader: strings.NewReader("request body")}
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	request.Body = originalBody
	request.ContentLength = -1
	operation := &RpcOperation{
		Request:  request,
		Response: httptest.NewRecorder(),
		Server:   testServerApp(),
	}

	if !operation.readRequestBody() {
		t.Fatal("readRequestBody() = false")
	}
	if !originalBody.closed {
		t.Fatal("original request body was not closed")
	}
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	assert.Equal(t, "request body", string(body))
}

type requestCloseTrackingBody struct {
	io.Reader
	closed bool
}

func (b *requestCloseTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestAccessAllowRpcReturnsServiceUnavailableWhenAuthServiceHasNoEndpoint(t *testing.T) {
	access := newTestAccess(t, testAuthValues(""))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    watched.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	})

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ServiceUnavailable, "auth service is unavailable")
}

func TestAccessAllowRpcMapsAuthServiceStatus(t *testing.T) {
	authEndpoint := registerTestAuthService(t, http.StatusOK, "UNAUTHORIZED", `null`)
	access := newTestAccess(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

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
	access := newTestAccess(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

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
	access := newTestAccess(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil).WithContext(ctx)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

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
	access := newTestAccess(t, testAuthValues(authEndpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")
	baseTrace, err := rpchttp.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)

	ok := access.AllowRpc(&RpcOperation{
		Auther:      authOperationForTest(t, request, recorder),
		Server:      testServerApp(),
		ActorVia:    watched.PortalActorVia{ActorSkelName: "demo.UserActor"},
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
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorDescriptor()),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skeldesc.Method{
				{SkelName: "Get", Name: "Get", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	}
	if authEndpoint != "" {
		values[watched.FormatRpcServiceRegistrationKey("demo.UserActorAuthService", "demo.app", "123e4567-e89b-12d3-a456-426614174011")] = vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
			Endpoint:      authEndpoint,
			ServiceName:   "demo.UserActorAuthService",
			AppName:       "demo.app",
			AppInstanceId: "123e4567-e89b-12d3-a456-426614174011",
		})
	}
	return values
}

func testAuthActorDescriptor() watched.DescriptorActor {
	return watched.DescriptorActor{
		SkelName: "demo.UserActor", Auth: &skeldesc.ActorAuth{Info: &skeldesc.Data{SkelName: "demo.UserInfo"}, Service: testAuthServiceDescriptor(), MethodName: testAuthMethodDescriptor().Name, Credential: testCredentialDescriptor()},
	}
}

func testUserActorAudiences() []*skeldesc.ActorAudience {
	return []*skeldesc.ActorAudience{{SkelName: "demo.UserActor"}}
}

func testAuthServiceDescriptor() *skeldesc.Service {
	return &skeldesc.Service{
		SkelName: "demo.UserActorAuthService",
		Methods: []*skeldesc.Method{
			testAuthMethodDescriptor(),
		}, AuthMode: skeldesc.AuthModeRequired,
	}
}

func testAuthMethodDescriptor() *skeldesc.Method {
	return &skeldesc.Method{
		SkelName: "auth",
		Arguments: []*skeldesc.Member{
			{Name: "credential"},
		},
		ResultType: &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.UserInfo"}, Name: "auth", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired,
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

func TestAccessAllowRpcRejectsMissingServiceDescriptor(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorDescriptor()),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ServiceUnavailable, "rpc service descriptor is not found")
}

func TestAccessAllowRpcRejectsMissingMethodDescriptor(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorDescriptor()),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skeldesc.Method{
				{SkelName: "List", Name: "List", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.NotFound, "rpc method descriptor is not found")
}

func TestAccessAllowRpcRejectsMissingActorDescriptor(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skeldesc.Method{
				{SkelName: "Get", Name: "Get", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ClientForbidden, "not allowed")
}

func TestAccessAllowRpcRejectsActorWithoutCredentialDescriptor(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(watched.DescriptorActor{
			SkelName: "demo.UserActor", Auth: &skeldesc.ActorAuth{Info: &skeldesc.Data{SkelName: "demo.UserInfo"}, Service: testAuthServiceDescriptor(), MethodName: testAuthMethodDescriptor().Name},
		}),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skeldesc.Method{
				{SkelName: "Get", Name: "Get", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	assert.Panics(t, func() {
		access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
	})
}

func TestAccessAllowRpcRejectsActorWithoutInfoDescriptor(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(watched.DescriptorActor{
			SkelName: "demo.UserActor", Auth: &skeldesc.ActorAuth{Credential: &skeldesc.Data{SkelName: "demo.UserCredential"}, Service: testAuthServiceDescriptor(), MethodName: testAuthMethodDescriptor().Name},
		}),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			Methods: []*skeldesc.Method{
				{SkelName: "Get", Name: "Get", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired},
			}, AuthMode: skeldesc.AuthModeRequired,
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	assert.Panics(t, func() {
		access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
	})
}

func TestRpcOperationConsumesEffectiveAuthMode(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	operation := &RpcOperation{Auther: Auther{Request: request, Response: httptest.NewRecorder(), actorDescriptor: &skeldesc.Actor{}},
		serviceDescriptor: &skeldesc.Service{AuthMode: skeldesc.AuthModeRequired},
		methodDescriptor:  &skeldesc.Method{AuthMode: skeldesc.AuthModeRequired, EffectiveAuthMode: skeldesc.AuthModeAnonymous}}
	require.True(t, operation.Auth())
	require.Equal(t, meta.ActorTypeAnonymous, operation.actor.Type())
}

func TestAccessAllowRpcInjectsActorAndServiceDescriptors(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorDescriptor()),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName:  "demo.UserService",
			Audiences: testUserActorAudiences(),
			AuthMode:  skeldesc.AuthModeOptional,
			Methods: []*skeldesc.Method{
				{SkelName: "Get", AuthMode: skeldesc.AuthModeOptional, Name: "Get", EffectiveAuthMode: skeldesc.AuthModeOptional},
			},
		}),
	})
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	ctx := &RpcOperation{
		Auther:      authOperationForTest(t, request, httptest.NewRecorder()),
		ActorVia:    watched.PortalActorVia{ActorSkelName: "demo.UserActor"},
		ServiceName: "demo.UserService",
		MethodName:  "Get",
	}

	require.True(t, access.AllowRpc(ctx))

	assert.Equal(t, "demo.UserActor", ctx.actorDescriptor.SkelName)
	assert.Equal(t, "demo.UserService", ctx.serviceDescriptor.SkelName)
}

func TestAccessAllowRpcRejectsDifferentActorVia(t *testing.T) {
	access := newTestAccess(t, map[string]string{
		watched.FormatDescriptorActorKey("demo.UserActor"): vcode.MustMarshalJsonS(testAuthActorDescriptor()),
		watched.FormatDescriptorServiceKey("demo.UserService"): vcode.MustMarshalJsonS(watched.DescriptorService{
			SkelName: "demo.UserService",
			AuthMode: skeldesc.AuthModeOptional,
			Audiences: []*skeldesc.ActorAudience{
				{SkelName: "demo.UserActor", Via: skeldesc.ActorViaAgent},
			},
			Methods: []*skeldesc.Method{
				{SkelName: "Get", AuthMode: skeldesc.AuthModeOptional, Name: "Get", EffectiveAuthMode: skeldesc.AuthModeOptional},
			},
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	ok := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{
		ActorSkelName: "demo.UserActor",
		ActorVia:      "client",
	}, request, recorder))

	assert.False(t, ok)
	assertRpcAuthError(t, recorder, ex.ClientForbidden, "rpc service does not allow actor via")
}

func TestRpcAccessOperationParseCredentialWritesUnauthorized(t *testing.T) {
	for name, authorization := range map[string]string{
		"missing":   "",
		"malformed": "Key1 token123, unknown value",
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx := &RpcOperation{
				actorDescriptor: &skeldesc.Actor{Auth: &skeldesc.ActorAuth{Credential: testCredentialDescriptor()}},
				Request:         httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil),
				Response:        response,
				Server:          testServerApp(),
			}
			if authorization != "" {
				ctx.Request.Header.Set(headerAuthorization, authorization)
			}

			succeed := ctx.parseCredential(ctx.writeError)
			assert.False(t, succeed)
			assertRpcAuthError(t, response, ex.Unauthorized, "bad credential")
		})
	}
}

func testRpcOperation(t *testing.T, actorVia watched.PortalActorVia, request *http.Request, response http.ResponseWriter) *RpcOperation {
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

func TestAccessAllowRpcPreservesAuthErrorReason(t *testing.T) {
	endpoint := "link+inproc://vine/auth-reason-test"
	ingressinproc.Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(rpchttp.HeaderContentType, "application/vrpc+json")
		rpchttp.EncodeStatusCodeToHeader(w.Header(), ex.PermissionDenied)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"message":"pending review","reason":"USER_PENDING_REVIEW"}}`))
	}))
	t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
	access := newTestAccess(t, testAuthValues(endpoint))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
	setTestRequestHeaders(t, request)
	request.Header.Set("Authorization", "Key1 token123, key2 dXNlcjpwd2Q=")
	allowed := access.AllowRpc(testRpcOperation(t, watched.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
	require.False(t, allowed)
	assertRpcAuthError(t, recorder, ex.PermissionDenied, "pending review")
	assert.Contains(t, recorder.Body.String(), `"reason":"USER_PENDING_REVIEW"`)
}
