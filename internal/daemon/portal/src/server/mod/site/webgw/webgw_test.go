package webgw

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"

	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	portalhubredis "go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/computil"
	"go.yorun.ai/vine/util/vcode"
)

func TestWebGatewayForwardsToRegistrationEndpoint(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Webgw-Test", "ok")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("forwarded"))
	}))

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://api.example.com/ping?a=1", nil)
	request.Header.Set("Origin", "https://console.example.com")

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
	if recorder.Header().Get("X-Webgw-Test") != "ok" {
		t.Fatalf("expected response header")
	}
	if recorder.Body.String() != "forwarded" {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestWebGatewayForwardsAnonymousActorWithoutAuthorization(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-anonymous-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := meta.DecodeActorFromBase64(r.Header.Get(webspec.HeaderWebActor))
		if err != nil {
			t.Fatalf("DecodeActorFromBase64() error = %v", err)
		}
		if !actor.IsAnonymous() {
			t.Fatalf("unexpected actor type: %s", actor.Type())
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestWebGatewayCreatesForwardTrace(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-trace-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Webgw-Trace", r.Header.Get(webspec.HeaderWebTrace))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=123e4567e89b12d3a456426614174000,span=1234567890abcdef")

	target.Serve(testContext(recorder, request))

	trace := mustDecodeWebTrace(t, recorder.Header().Get("X-Webgw-Trace"))
	if trace.Id() != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("unexpected trace id: %s", trace.Id())
	}
	if trace.Span() == "1234567890abcdef" {
		t.Fatalf("expected forwarded span to differ from incoming span")
	}
}

func TestWebGatewayAddsDefaultOptionsTimeoutBeforeForward(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-default-options-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Webgw-Options", r.Header.Get(webspec.HeaderWebOptions))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)

	target.Serve(testContext(recorder, request))

	options := mustDecodeWebOptions(t, recorder.Header().Get("X-Webgw-Options"))
	if options.Timeout <= 0 || options.Timeout > defaultWebTimeout {
		t.Fatalf("forwarded timeout = %s, want within %s", options.Timeout, defaultWebTimeout)
	}
}

func TestWebGatewayIgnoresClientCancelAfterRequestIsAccepted(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-client-cancel-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("forwarded"))
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil).WithContext(ctx)
	cancel()

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "forwarded" {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestWebGatewayForwardsRemainingOptionsTimeout(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-remaining-options-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Webgw-Options", r.Header.Get(webspec.HeaderWebOptions))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set(webspec.HeaderWebOptions, "timeout=60s")

	target.Serve(testContext(recorder, request))

	options := mustDecodeWebOptions(t, recorder.Header().Get("X-Webgw-Options"))
	if options.Timeout <= 0 || options.Timeout > 60*time.Second {
		t.Fatalf("forwarded timeout = %s, want remaining timeout within 60s", options.Timeout)
	}
}

func TestWebGatewayRejectsOptionsTimeoutOverMax(t *testing.T) {
	target := newTestWebGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set(webspec.HeaderWebOptions, "timeout=121s")

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), webspec.HeaderWebOptions) {
		t.Fatalf("expected error body to mention %s, got %q", webspec.HeaderWebOptions, recorder.Body.String())
	}
}

func TestWebGatewayReturnsRequestTraceId(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/portal-webgw-trace-id-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=123e4567e89b12d3a456426614174000,span=1234567890abcdef")

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if got := recorder.Header().Get(spec.HeaderPortalTraceId); got != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("portal trace id = %q", got)
	}
}

func TestWebGatewayReturnsGeneratedTraceIdWhenRequestTraceIsInvalid(t *testing.T) {
	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      "http://127.0.0.1:23001/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=bad-id,span=1234567890abcdef")

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if got := recorder.Header().Get(spec.HeaderPortalTraceId); !meta.IsValidId(got) || got == "bad-id" {
		t.Fatalf("portal trace id = %q, want generated valid trace id", got)
	}
}

func TestWebGatewayCompressesLargeTextResponse(t *testing.T) {
	body := strings.Repeat("web response body", computil.CompressionThreshold)
	ingressEndpoint := "link+inproc://vine/portal-webgw-compression-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))

	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      ingressEndpoint + "/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	request.Header.Set("Accept-Encoding", "gzip, zstd")

	target.Serve(testContext(recorder, request))

	if got := recorder.Header().Get("Content-Encoding"); got != computil.EncodingZstd {
		t.Fatalf("content-encoding = %q, want zstd", got)
	}
	if got := recorder.Header().Values("Vary"); !headerValuesContain(got, "Accept-Encoding") {
		t.Fatalf("vary = %v, want accept-encoding", got)
	}
	decoded := mustDecodeZstd(t, recorder.Body.Bytes())
	if string(decoded) != body {
		t.Fatalf("unexpected decoded body")
	}
}

func TestWebGatewayReturnsUnavailableWhenNoEndpoint(t *testing.T) {
	target := newTestWebGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)

	target.Serve(testContext(recorder, request))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestWebGatewayAllowsOptionsFromSameEntryDomain(t *testing.T) {
	target := newTestWebGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/ping", nil)
	request.Header.Set("Origin", "https://console.example.com")
	request.Header.Set(spec.HeaderAccessControlReqHeaders, "content-type, authorization")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))

	assertOptionsAllowed(t, recorder, "https://console.example.com")
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowHeaders); got != "content-type, authorization" {
		t.Fatalf("allow headers = %q", got)
	}
}

func TestWebGatewayDoesNotAllowOptionsForWildcardEntryOrigin(t *testing.T) {
	target := newTestWebGateway(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/ping", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, ""))

	assertOptionsNotAllowed(t, recorder)
}

func TestWebGatewayAllowsOptionsFromStrictAllowedOrigin(t *testing.T) {
	target := newTestWebGatewayWithCors(nil, redised.PortalCors{
		Mode: redised.PortalCorsModeStrict,
		AllowedOrigins: []string{
			"https://console.example.com",
		},
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.net/ping", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.net"))

	assertOptionsAllowed(t, recorder, "https://console.example.com")
}

func TestWebGatewayDoesNotAllowOptionsWhenCorsDisabled(t *testing.T) {
	target := newTestWebGatewayWithCors(nil, redised.PortalCors{
		Mode: redised.PortalCorsModeDisabled,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "http://api.example.com/ping", nil)
	request.Header.Set("Origin", "https://console.example.com")

	target.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))

	assertOptionsNotAllowed(t, recorder)
}

func TestWebGatewayUpdateKeepsRegistrationWhenWebNameDoesNotChange(t *testing.T) {
	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      "http://127.0.0.1:23001/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})

	target.Update(redised.PortalSite{
		Name: "demo-web",
		Type: "WEBGW",
		WebgwConfig: &redised.PortalWebgwConfig{
			WebName: "admin@demo.app",
		},
	})

	if _, ok := target.routeWeb(); !ok {
		t.Fatal("expected endpoint after update")
	}
}

func TestWebGatewayUpdateSwitchesWebName(t *testing.T) {
	target := newTestWebGateway(map[string]string{
		redised.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      "http://127.0.0.1:23001/web/proxy/in/instance-1/admin@demo.app",
			WebSkelName:   "admin@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
		redised.FormatWebRegistrationKey("home@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(redised.WebRegistration{
			Endpoint:      "http://127.0.0.1:23002/web/proxy/in/instance-1/home@demo.app",
			WebSkelName:   "home@demo.app",
			AppName:       "demo.app",
			AppInstanceId: "instance-1",
		}),
	})

	target.Update(redised.PortalSite{
		Name: "demo-web",
		Type: "WEBGW",
		WebgwConfig: &redised.PortalWebgwConfig{
			WebName: "home@demo.app",
		},
	})

	registration, ok := target.routeWeb()
	if !ok {
		t.Fatal("expected endpoint after web name update")
	}
	if registration.WebSkelName != "home@demo.app" {
		t.Fatalf("unexpected web registration: %+v", registration)
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

func newTestWebGateway(valuesByKey map[string]string) *WebGateway {
	return newTestWebGatewayWithCors(valuesByKey, redised.PortalCors{
		Mode: redised.PortalCorsModeSameDomain,
	})
}

func newTestWebGatewayWithCors(valuesByKey map[string]string, cors redised.PortalCors) *WebGateway {
	return New(context.Background(), new(access.Access), newTestEpmgr(valuesByKey), redised.PortalSite{
		Name: "demo-web",
		Type: "WEBGW",
		Cors: cors,
		WebgwConfig: &redised.PortalWebgwConfig{
			WebName: "admin@demo.app",
		},
	})
}

func newTestEpmgr(valuesByKey map[string]string) *epmgr.Manager {
	manager := &epmgr.Manager{
		Context: context.Background(),
		Redis:   portalhubredis.NewTestClient(valuesByKey),
	}
	manager.DIInit()
	return manager
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
	if got := recorder.Header().Get(spec.HeaderAccessControlAllowMethods); got != "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS" {
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

func mustDecodeWebOptions(t *testing.T, value string) *webspec.Options {
	t.Helper()

	header := http.Header{}
	header.Set(webspec.HeaderWebOptions, value)
	options, err := webspec.DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	return options
}

func mustDecodeWebTrace(t *testing.T, value string) meta.Trace {
	t.Helper()

	header := http.Header{}
	header.Set(webspec.HeaderWebTrace, value)
	trace, err := webspec.DecodeTraceFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	return trace
}
