package rpcgw

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	portalhubredis "go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/computil"
	"go.yorun.ai/vine/util/vcode"
)

func TestRpcGatewayForwardsConfiguredServiceToRegistrationEndpoint(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Test", "ok")
		w.Header().Set("X-Rpcgw-Path", r.URL.Path)
		w.Header().Set("X-Rpcgw-Trace", r.Header.Get(rpchttp.HeaderRpcTrace))
		w.Header().Set("X-Rpcgw-Initiator", r.Header.Get(rpchttp.HeaderRpcInitiator))
		w.Header().Set("X-Rpcgw-Actor", r.Header.Get(rpchttp.HeaderRpcActor))
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://api.example.com/invoke/demo.UserService/Get?debug=1", strings.NewReader("request"))
	setTestAuthHeaders(request)
	request.Header.Set("Origin", "https://console.example.com")
	request.Header.Set("User-Agent", "curl/8.0")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowOrigin); got != "https://console.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowCredentials); got != "true" {
		t.Fatalf("allow credentials = %q", got)
	}
	if got := recorder.Header().Values(spec.HeaderAccessControlExposeHeaders); !headerValuesContain(got, spec.HeaderPortalTraceId) {
		t.Fatalf("expose headers = %v, want %s", got, spec.HeaderPortalTraceId)
	}
	if got := recorder.Header().Get(spec.HeaderPortalTraceId); !meta.IsValidId(got) {
		t.Fatalf("portal trace id = %q, want valid trace id", got)
	}
	if recorder.Header().Get("X-Rpcgw-Test") != "ok" {
		t.Fatalf("expected response header")
	}
	if got := recorder.Header().Get("X-Rpcgw-Path"); got != "/rpc/proxy/in/instance-1/demo.UserService/Get" {
		t.Fatalf("unexpected forwarded path: %s", got)
	}
	traceHeader := http.Header{}
	traceHeader.Set(rpchttp.HeaderRpcTrace, recorder.Header().Get("X-Rpcgw-Trace"))
	gotTrace, err := rpchttp.DecodeTraceFromHeader(traceHeader)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if !meta.IsValidSpan(gotTrace.Span()) {
		t.Fatalf("expected generated rpc span, got %s", gotTrace.Span())
	}
	initiator, err := meta.DecodeInitiatorFromBase64(recorder.Header().Get("X-Rpcgw-Initiator"))
	if err != nil {
		t.Fatalf("DecodeInitiatorFromBase64() error = %v", err)
	}
	if initiator.Name() != "demo.client" || initiator.Version() != "0.0.0" || initiator.InstanceId() != "123e4567-e89b-12d3-a456-426614174001" {
		t.Fatalf("unexpected initiator app: %#v", initiator)
	}
	if initiator.Dialer() != "curl/8.0" {
		t.Fatalf("unexpected initiator dialer: %s", initiator.Dialer())
	}
	if initiator.IpAddr() != "192.0.2.1" {
		t.Fatalf("unexpected initiator ip: %v", initiator.IpAddr())
	}
	if recorder.Body.String() != "forwarded" {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestRpcGatewayClearsAcceptEncodingBeforeForward(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-accept-encoding-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Accept-Encoding", r.Header.Get(rpchttp.HeaderAcceptEncoding))
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderAcceptEncoding, "gzip, zstd")

	target.Serve(testContext(recorder, request))

	if got := recorder.Header().Get("X-Rpcgw-Accept-Encoding"); got != "" {
		t.Fatalf("forwarded accept-encoding = %q, want empty", got)
	}
}

func TestRpcGatewayAddsDefaultRpcOptionsTimeoutBeforeForward(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-default-options-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Options", r.Header.Get(rpchttp.HeaderRpcOptions))
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)

	target.Serve(testContext(recorder, request))

	options := mustDecodeRpcOptions(t, recorder.Header().Get("X-Rpcgw-Options"))
	if options.Timeout <= 0 || options.Timeout > defaultRpcTimeout {
		t.Fatalf("forwarded timeout = %s, want within %s", options.Timeout, defaultRpcTimeout)
	}
}

func TestRpcGatewayIgnoresClientCancelAfterRequestIsAccepted(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-client-cancel-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("forwarded"))
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil).WithContext(ctx)
	setTestAuthHeaders(request)
	cancel()

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "forwarded" {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestRpcGatewayForwardsRemainingRpcOptionsTimeout(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-remaining-options-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Options", r.Header.Get(rpchttp.HeaderRpcOptions))
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderRpcOptions, "timeout=60s")

	target.Serve(testContext(recorder, request))

	options := mustDecodeRpcOptions(t, recorder.Header().Get("X-Rpcgw-Options"))
	if options.Timeout <= 0 || options.Timeout > 60*time.Second {
		t.Fatalf("forwarded timeout = %s, want remaining timeout within 60s", options.Timeout)
	}
}

func TestRpcGatewayRejectsRpcOptionsTimeoutOverMax(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "http://127.0.0.1:23001/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderRpcOptions, "timeout=121s")

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, rpchttp.HeaderRpcOptions)
}

func TestRpcGatewayGeneratesMissingRpcSpanBeforeForward(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-missing-span-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Trace", r.Header.Get(rpchttp.HeaderRpcTrace))
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderRpcTrace, "id=123e4567e89b12d3a456426614174000")

	target.Serve(testContext(recorder, request))

	traceHeader := http.Header{}
	traceHeader.Set(rpchttp.HeaderRpcTrace, recorder.Header().Get("X-Rpcgw-Trace"))
	trace, err := rpchttp.DecodeTraceFromHeader(traceHeader)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if trace.Id() != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("unexpected trace id: %s", trace.Id())
	}
	if !meta.IsValidSpan(trace.Span()) {
		t.Fatalf("expected generated span, got %s", trace.Span())
	}
}

func TestRpcGatewayMapsForwardedRpcErrorToHTTPStatusCode(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-error-status-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rpchttp.WriteRequestErrorResponse(w, r, testServerApp(), ex.New(ex.NotFound, "missing user")); err != nil {
			t.Fatalf("WriteRequestErrorResponse() error = %v", err)
		}
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get(rpchttp.HeaderRpcStatus) != string(ex.NotFound) {
		t.Fatalf("unexpected rpc status: %s", recorder.Header().Get(rpchttp.HeaderRpcStatus))
	}
}

func TestRpcGatewayCompressesLargeResponseWithZstdFirst(t *testing.T) {
	body := strings.Repeat("zstd response body", computil.CompressionThreshold)
	recorder := serveTestRpcGatewayCompressedResponse(t, body, "gzip, zstd")

	if got := recorder.Header().Get(rpchttp.HeaderContentEncoding); got != computil.EncodingZstd {
		t.Fatalf("content-encoding = %q, want zstd", got)
	}
	if got := recorder.Header().Values("Vary"); !headerValuesContain(got, rpchttp.HeaderAcceptEncoding) {
		t.Fatalf("vary = %v, want accept-encoding", got)
	}
	decoded := mustDecodeZstd(t, recorder.Body.Bytes())
	if string(decoded) != body {
		t.Fatalf("unexpected decoded body")
	}
}

func TestRpcGatewayCompressesLargeResponseWithGzipFallback(t *testing.T) {
	body := strings.Repeat("gzip response body", computil.CompressionThreshold)
	recorder := serveTestRpcGatewayCompressedResponse(t, body, "gzip")

	if got := recorder.Header().Get(rpchttp.HeaderContentEncoding); got != computil.EncodingGzip {
		t.Fatalf("content-encoding = %q, want gzip", got)
	}
	decoded := mustDecodeGzip(t, recorder.Body.Bytes())
	if string(decoded) != body {
		t.Fatalf("unexpected decoded body")
	}
}

func TestRpcGatewayCreatesForwardSpan(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-span-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Trace", r.Header.Get(rpchttp.HeaderRpcTrace))
		w.WriteHeader(http.StatusAccepted)
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	trace, err := meta.NewTrace(meta.NewId(), "1234567890abcdef")
	if err != nil {
		t.Fatalf("NewTrace() error = %v", err)
	}
	rpchttp.EncodeTraceToHeader(request.Header, trace)

	target.Serve(testContext(recorder, request))

	got := mustDecodeRpcTrace(t, recorder.Header().Get("X-Rpcgw-Trace"))
	if got.Id() != trace.Id() {
		t.Fatalf("unexpected trace id: %s", got.Id())
	}
	if got.Span() == trace.Span() {
		t.Fatalf("expected forwarded span to differ from incoming span")
	}
}

func TestRpcGatewayOverwritesExistingRpcInitiator(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-initiator-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Initiator", r.Header.Get(rpchttp.HeaderRpcInitiator))
		w.WriteHeader(http.StatusAccepted)
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	expected, err := meta.NewInitiator("other.client", "1.2.3", "123e4567-e89b-12d3-a456-426614174010", "browser", "203.0.113.9")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderRpcInitiator, meta.EncodeInitiatorToBase64(expected))
	request.Header.Set("User-Agent", "curl/8.0")

	target.Serve(testContext(recorder, request))

	got, err := meta.DecodeInitiatorFromBase64(recorder.Header().Get("X-Rpcgw-Initiator"))
	if err != nil {
		t.Fatalf("DecodeInitiatorFromBase64() error = %v", err)
	}
	if got.Name() != "demo.client" || got.Dialer() != "curl/8.0" || got.IpAddr() != "192.0.2.1" {
		t.Fatalf("unexpected initiator: %#v", got)
	}
}

func TestRpcGatewayRejectsMissingRpcClient(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "http://127.0.0.1:23001/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	rpchttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, rpchttp.HeaderRpcClient)
}

func TestRpcGatewayRejectsMissingRpcTrace(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "http://127.0.0.1:23001/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	request.Header.Set(rpchttp.HeaderRpcClient, "name=demo.client,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174001")

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, rpchttp.HeaderRpcTrace)
}

func TestRpcGatewayRejectsInvalidContentType(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "http://127.0.0.1:23001/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderContentType, "application/json")

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, rpchttp.HeaderContentType)
}

func TestRpcGatewayRejectsInvalidRpcOptionsForInprocEndpoint(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "rpc+inproc://vine/portal-rpcgw-options-test/rpc/invoke",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderAccept, rpchttp.ContentTypeJson)
	request.Header.Set(rpchttp.HeaderRpcOptions, "timeout=soon")

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, rpchttp.HeaderRpcOptions)
}

func TestRpcGatewayOverwritesExistingRpcActor(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-rpcgw-actor-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rpcgw-Actor", r.Header.Get(rpchttp.HeaderRpcActor))
		w.WriteHeader(http.StatusAccepted)
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	actor := meta.NewAuthenticatedActorForTest()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderRpcActor, meta.EncodeActorToBase64(actor))

	target.Serve(testContext(recorder, request))

	got, err := meta.DecodeActorFromBase64(recorder.Header().Get("X-Rpcgw-Actor"))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != meta.ActorTypeAnonymous {
		t.Fatalf("unexpected actor type: %s", got.Type())
	}
}

func TestRpcGatewayReturnsNotFoundOutsideInvokePath(t *testing.T) {
	target := newTestRpcGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.NotFound, "rpcgw path is not found")
}

func TestRpcGatewayDispatchesInspectPath(t *testing.T) {
	target := newTestRpcGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/inspect", nil)

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.InvalidRequest, "rpcgw inspect is not implemented")
}

func TestRpcGatewayAllowsOptionsFromSameEntryDomain(t *testing.T) {
	target := newTestRpcGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/invoke/demo.UserService/Get", nil)
	request.Header.Set("Origin", "https://console.example.com")
	request.Header.Set(spec.HeaderAccessControlReqHeaders, "content-type, authorization")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))

	assertOptionsAllowed(t, recorder, "https://console.example.com")
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowHeaders); got != "content-type, authorization" {
		t.Fatalf("allow headers = %q", got)
	}
}

func TestRpcGatewayDoesNotAllowOptionsForWildcardEntryOrigin(t *testing.T) {
	target := newTestRpcGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/invoke/demo.UserService/Get", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, ""))

	assertOptionsNotAllowed(t, recorder)
}

func TestRpcGatewayAllowsOptionsFromStrictAllowedOrigin(t *testing.T) {
	target := newTestRpcGatewayWithCors(nil, redised.PortalCors{
		Mode: redised.PortalCorsModeStrict,
		AllowedOrigins: []string{
			"https://console.example.com",
		},
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.net/invoke/demo.UserService/Get", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.net"))

	assertOptionsAllowed(t, recorder, "https://console.example.com")
}

func TestRpcGatewayDoesNotAllowOptionsWhenCorsDisabled(t *testing.T) {
	target := newTestRpcGatewayWithCors(nil, redised.PortalCors{
		Mode: redised.PortalCorsModeDisabled,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/invoke/demo.UserService/Get", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))

	assertOptionsNotAllowed(t, recorder)
}

func TestRpcGatewayReturnsUnavailableWhenServiceHasNoEndpoint(t *testing.T) {
	target := newTestRpcGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)

	target.Serve(testContext(recorder, request))

	assertRpcGatewayError(t, recorder, ex.ServiceUnavailable, "rpcgw service endpoint is unavailable")
}

func TestRpcGatewayRejectsServiceOutsideConfiguredList(t *testing.T) {
	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.OrderService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      "http://127.0.0.1:23001/rpc/proxy/in/instance-1",
			ServiceName:   "demo.OrderService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})

	target.Update(redised.PortalSite{
		Name: "demo-api",
		Type: "RPCGW",
		RpcgwConfig: &redised.PortalRpcgwConfig{
			Services: []redised.PortalRpcgwService{{SkelName: "demo.OrderService"}},
		},
	})

	target.Update(redised.PortalSite{
		Name:        "demo-api",
		Type:        "RPCGW",
		RpcgwConfig: &redised.PortalRpcgwConfig{},
	})

	if registration, configured := target.routeService("demo.OrderService"); configured || registration != nil {
		t.Fatal("expected service to be outside configured list")
	}
}

func testContext(recorder http.ResponseWriter, request *http.Request) *spec.Context {
	return testContextWithEntryOrigin(recorder, request, "")
}

func testContextWithEntryOrigin(recorder http.ResponseWriter, request *http.Request, entryHost string) *spec.Context {
	return &spec.Context{
		Request:        request,
		ResponseWriter: recorder,
		RemoteAddr:     "192.0.2.1",
		EntryOrigin: spec.EntryOrigin{
			Scheme: spec.SchemeHTTP,
			Host:   entryHost,
			Port:   80,
		},
	}
}

func setTestAuthHeaders(request *http.Request) {
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	rpchttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	request.Header.Set(rpchttp.HeaderRpcClient, "name=demo.client,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174001")
	request.Header.Set("Authorization", "key token")
}

func testCredentialSchema() *skel.DataSchema {
	return &skel.DataSchema{
		SkelName: "demo.UserCredential",
		Members: []*skel.MemberSchema{
			{Name: "key"},
		},
	}
}

func newTestRpcGateway(valuesByKey map[string]string) *RpcGateway {
	return newTestRpcGatewayWithCors(valuesByKey, redised.PortalCors{
		Mode: redised.PortalCorsModeSameDomain,
	})
}

func newTestRpcGatewayWithCors(valuesByKey map[string]string, cors redised.PortalCors) *RpcGateway {
	return New(context.Background(), testServerApp(), newTestAccess(), newTestEpmgr(valuesByKey), redised.PortalSite{
		Name: "demo-api",
		Type: "RPCGW",
		ActorVia: redised.PortalActorVia{
			ActorSkelName: "demo.UserActor",
		},
		Cors: cors,
		RpcgwConfig: &redised.PortalRpcgwConfig{
			Services: []redised.PortalRpcgwService{{SkelName: "demo.UserService"}},
		},
	})
}

func assertOptionsAllowed(t *testing.T, recorder *httptest.ResponseRecorder, origin string) {
	t.Helper()

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowOrigin); got != origin {
		t.Fatalf("allow origin = %q, want %q", got, origin)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowCredentials); got != "true" {
		t.Fatalf("allow credentials = %q", got)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowMethods); got != "POST, OPTIONS" {
		t.Fatalf("allow methods = %q", got)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlMaxAge); got != "600" {
		t.Fatalf("max age = %q", got)
	}
}

func assertOptionsNotAllowed(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowOrigin); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}

func assertRpcGatewayError(t *testing.T, recorder *httptest.ResponseRecorder, code ex.Code, message string) {
	t.Helper()

	if recorder.Code != ex.HTTPStatusCode(code) {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get(rpchttp.HeaderRpcStatus) != string(code) {
		t.Fatalf("unexpected rpc status: %s", recorder.Header().Get(rpchttp.HeaderRpcStatus))
	}
	if !strings.Contains(recorder.Body.String(), message) {
		t.Fatalf("expected error message %q, got %q", message, recorder.Body.String())
	}
}

func mustDecodeRpcOptions(t *testing.T, value string) *rpchttp.Options {
	t.Helper()

	header := http.Header{}
	header.Set(rpchttp.HeaderRpcOptions, value)
	options, err := rpchttp.DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	return options
}

func mustDecodeRpcTrace(t *testing.T, value string) meta.Trace {
	t.Helper()

	header := http.Header{}
	header.Set(rpchttp.HeaderRpcTrace, value)
	trace, err := rpchttp.DecodeTraceFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	return trace
}

func testServerApp() meta.App {
	return meta.MustNewApp("vine.portal", "0.0.0", "123e4567-e89b-12d3-a456-426614174099")
}

func newTestEpmgr(valuesByKey map[string]string) *epmgr.Manager {
	manager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   portalhubredis.NewTestClient(valuesByKey),
	}
	manager.DIInit()
	return manager
}

func newTestAccess() *access.Access {
	redisClient := newTestSchemaRedis()
	epmgrManager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   redisClient,
	}
	epmgrManager.DIInit()
	manager := &access.Access{
		Context: context.Background(),
		Redis:   redisClient,
		Epmgr:   epmgrManager,
	}
	manager.DIInit()
	return manager
}

func newTestSchemaRedis() *portalhubredis.Client {
	redisClient := portalhubredis.NewTestClient(map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName:       "demo.UserActor",
			AuthCredential: testCredentialSchema(),
			AuthInfo:       &skel.DataSchema{SkelName: "demo.UserInfo"},
		}),
		redised.FormatSchemaServiceKey("demo.UserService"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName: "demo.UserService",
			AuthMode: skel.AuthModeNoAuth,
			Audiences: []*skel.ActorAudienceSchema{
				{SkelName: "demo.UserActor"},
			},
			Methods: []*skel.MethodSchema{
				{SkelName: "Get", AuthMode: skel.AuthModeNoAuth},
			},
		}),
	})
	return redisClient
}

func serveTestRpcGatewayCompressedResponse(t *testing.T, body string, acceptEncoding string) *httptest.ResponseRecorder {
	t.Helper()

	ingressEndpoint := "link+inproc://vine/portal-rpcgw-compression-test-" + meta.NewId()
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	target := newTestRpcGateway(map[string]string{
		redised.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      ingressEndpoint + "/rpc/proxy/in/instance-1",
			ServiceName:   "demo.UserService",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", nil)
	setTestAuthHeaders(request)
	request.Header.Set(rpchttp.HeaderAcceptEncoding, acceptEncoding)

	target.Serve(testContext(recorder, request))
	return recorder
}

func mustDecodeZstd(t *testing.T, body []byte) []byte {
	t.Helper()

	reader, err := zstd.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("zstd.NewReader() error = %v", err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("zstd ReadAll() error = %v", err)
	}
	return decoded
}

func mustDecodeGzip(t *testing.T, body []byte) []byte {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = reader.Close() }()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("gzip ReadAll() error = %v", err)
	}
	return decoded
}

func headerValuesContain(values []string, target string) bool {
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), target) {
				return true
			}
		}
	}
	return false
}
