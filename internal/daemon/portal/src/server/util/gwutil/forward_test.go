package gwutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/util/httputil"
)

func TestForwardRequest(t *testing.T) {
	t.Run("http endpoint", func(t *testing.T) {
		var gotURL string
		responseBody := &closeTrackingReadCloser{Reader: strings.NewReader("forwarded")}
		prevDo := do
		do = func(req *http.Request) (*http.Response, error) {
			gotURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Header:     http.Header{"X-Gwutil-Test": []string{"ok"}},
				Body:       responseBody,
			}, nil
		}
		defer func() {
			do = prevDo
		}()

		request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.Service/Get?debug=1", strings.NewReader("request"))
		response, err := ForwardRequest(request, "http://127.0.0.1:23001/rpc/proxy/in/instance-1")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if gotURL != "http://127.0.0.1:23001/rpc/proxy/in/instance-1/demo.Service/Get?debug=1" {
			t.Fatalf("unexpected target url: %s", gotURL)
		}
		if response.StatusCode != http.StatusAccepted {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if response.Header.Get("X-Gwutil-Test") != "ok" {
			t.Fatalf("expected response header")
		}
		if string(body) != "forwarded" {
			t.Fatalf("unexpected response body: %s", body)
		}
		if !responseBody.closed {
			t.Fatal("expected original response body to be closed")
		}
	})

	t.Run("inproc ingress endpoint", func(t *testing.T) {
		ingressEndpoint := "link+inproc://vine/link"
		ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Gwutil-Inproc-Test", r.URL.Path+"?"+r.URL.RawQuery)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("forwarded-inproc"))
		}))

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping?a=1", nil)
		response, err := ForwardRequest(request, ingressEndpoint+"/web/proxy/in/instance-1/admin@demo.app")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if response.StatusCode != http.StatusAccepted {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if got := response.Header.Get("X-Gwutil-Inproc-Test"); got != "/web/proxy/in/instance-1/admin@demo.app/ping?a=1" {
			t.Fatalf("unexpected inproc path: %s", got)
		}
		if string(body) != "forwarded-inproc" {
			t.Fatalf("unexpected response body: %s", body)
		}
	})

	t.Run("rpc inproc endpoint", func(t *testing.T) {
		rpcEndpoint := "rpc+inproc://vine/gwutil-rpc-test/rpc/invoke"
		rpcinproc.Register(rpcEndpoint, _TestRpcHandler{})
		t.Cleanup(func() {
			rpcinproc.Unregister(rpcEndpoint)
		})

		request := httptest.NewRequest(http.MethodPost, "http://demo.local/vine.hub.AppConfigService/list", nil)
		request.Header.Set("accept", "application/vrpc+json")
		request.Header.Set("content-type", "application/vrpc+json")
		request.Header.Set("vrpc-trace", "id=123e4567e89b12d3a456426614174000,span=1234567890abcdef")
		request.Header.Set("vrpc-client", "name=demo.client,version=0.0.0,instanceId=123e4567-e89b-12d3-a456-426614174001")

		response, err := ForwardRequest(request, rpcEndpoint)
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if response.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if response.Header.Get("vrpc-status") != string(ex.OK) {
			t.Fatalf("unexpected rpc status: %s", response.Header.Get("vrpc-status"))
		}
		if !strings.Contains(string(body), "demo.Config") {
			t.Fatalf("unexpected response body: %s", body)
		}
	})

	t.Run("web inproc endpoint", func(t *testing.T) {
		webEndpoint := "web+inproc://vine/gwutil-web-test/web/access"
		webinproc.Register(webEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Gwutil-Web-Inproc-Test", r.URL.Path+"?"+r.URL.RawQuery)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("forwarded-web-inproc"))
		}))
		t.Cleanup(func() {
			webinproc.Unregister(webEndpoint)
		})

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/assets/app.js?v=1", nil)
		response, err := ForwardRequest(request, webEndpoint)
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if response.StatusCode != http.StatusCreated {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if got := response.Header.Get("X-Gwutil-Web-Inproc-Test"); got != "/web/access/vine.hub.DashboardWeb/assets/app.js?v=1" {
			t.Fatalf("unexpected web inproc path: %s", got)
		}
		if string(body) != "forwarded-web-inproc" {
			t.Fatalf("unexpected response body: %s", body)
		}
	})

	t.Run("web inproc endpoint with web name suffix", func(t *testing.T) {
		webEndpoint := "web+inproc://vine/gwutil-web-suffix-test/web/access"
		webinproc.Register(webEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Gwutil-Web-Inproc-Suffix-Test", r.URL.Path+"?"+r.URL.RawQuery)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("forwarded-web-inproc-suffix"))
		}))
		t.Cleanup(func() {
			webinproc.Unregister(webEndpoint)
		})

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/assets/app.js?v=1", nil)
		response, err := ForwardRequest(request, webEndpoint+"/vine.hub.DashboardWeb")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if response.StatusCode != http.StatusCreated {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if got := response.Header.Get("X-Gwutil-Web-Inproc-Suffix-Test"); got != "/web/access/vine.hub.DashboardWeb/assets/app.js?v=1" {
			t.Fatalf("unexpected web inproc path: %s", got)
		}
		if string(body) != "forwarded-web-inproc-suffix" {
			t.Fatalf("unexpected response body: %s", body)
		}
	})

	t.Run("web inproc endpoint with web name suffix keeps root slash", func(t *testing.T) {
		webEndpoint := "web+inproc://vine/gwutil-web-root-test/web/access"
		webinproc.Register(webEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Gwutil-Web-Inproc-Root-Test", r.URL.Path)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("forwarded-web-inproc-root"))
		}))
		t.Cleanup(func() {
			webinproc.Unregister(webEndpoint)
		})

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/", nil)
		response, err := ForwardRequest(request, webEndpoint+"/vine.hub.DashboardWeb")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		defer response.Body.Close()
		body := mustReadResponseBody(t, response)

		if response.StatusCode != http.StatusCreated {
			t.Fatalf("unexpected status code: %d", response.StatusCode)
		}
		if got := response.Header.Get("X-Gwutil-Web-Inproc-Root-Test"); got != "/web/access/vine.hub.DashboardWeb/" {
			t.Fatalf("unexpected web inproc path: %s", got)
		}
		if string(body) != "forwarded-web-inproc-root" {
			t.Fatalf("unexpected response body: %s", body)
		}
	})
}

func TestForwardRequestContextTimeout(t *testing.T) {
	t.Run("ordinary request keeps timeout", func(t *testing.T) {
		var hasDeadline bool
		prevDo := do
		do = func(req *http.Request) (*http.Response, error) {
			_, hasDeadline = req.Context().Deadline()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}
		defer func() {
			do = prevDo
		}()

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
		response, err := ForwardRequest(request, "http://127.0.0.1:23001")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		_ = response.Body.Close()
		if !hasDeadline {
			t.Fatal("expected ordinary web request deadline")
		}
	})

	t.Run("event stream has no total timeout", func(t *testing.T) {
		var hasDeadline bool
		prevDo := do
		do = func(req *http.Request) (*http.Response, error) {
			_, hasDeadline = req.Context().Deadline()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}
		defer func() {
			do = prevDo
		}()

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/events", nil)
		request.Header.Set("Accept", "text/event-stream")
		response, err := ForwardRequest(request, "http://127.0.0.1:23001")
		if err != nil {
			t.Fatalf("ForwardRequest() error = %v", err)
		}
		_ = response.Body.Close()
		if hasDeadline {
			t.Fatal("did not expect event stream request deadline")
		}
	})

	t.Run("event stream still follows parent cancel", func(t *testing.T) {
		prevDo := do
		do = func(req *http.Request) (*http.Response, error) {
			return nil, req.Context().Err()
		}
		defer func() {
			do = prevDo
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		request := httptest.NewRequest(http.MethodGet, "http://demo.local/events", nil).WithContext(ctx)
		request.Header.Set("Accept", "text/event-stream")
		_, err := ForwardRequest(request, "http://127.0.0.1:23001")
		if err == nil {
			t.Fatal("expected canceled context error")
		}
	})
}

func mustReadResponseBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	body, err := httputil.ReadResponseBody(response)
	if err != nil {
		t.Fatalf("ReadResponseBody() error = %v", err)
	}
	return body
}

type _TestRpcHandler struct{}

func (_TestRpcHandler) ServeRpc(rpcRequest rpcspec.Request) rpcspec.Response {
	return &rpcspec.ResponseImpl{
		ServerValue: meta.MustNewApp("vine.hub", "0.0.0", "123e4567-e89b-12d3-a456-426614174002"),
		MethodValue: rpcRequest.MethodInfo(),
		ResultValue: []skeled.AppConfigItem{{
			Key:       "demo.Config",
			Lifecycle: "ETERNAL",
			Value:     "{}",
		}},
		ErrorValue: ex.NewOK(),
	}
}

type closeTrackingReadCloser struct {
	*strings.Reader
	closed bool
}

func (r *closeTrackingReadCloser) Close() error {
	r.closed = true
	return nil
}

var _ io.ReadCloser = (*closeTrackingReadCloser)(nil)
