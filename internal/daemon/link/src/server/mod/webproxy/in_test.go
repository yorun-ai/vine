package webproxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreapp "go.yorun.ai/vine/internal/core/app"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
)

func TestResolveInboundRouteRejectsUnknownWebName(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	localApp := mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	registerLocalWebApp(proxy, localApp, "http://127.0.0.1:8080"+testPathWebAccess, "http://127.0.0.1:8080", []string{"admin@demo.app"})

	_, _, err := proxy.resolveInboundRoute("/" + localApp.InstanceId() + "/other@demo.app/ping")
	if err != errWebEndpointUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleInForwardsToRegisteredWebEndpoint(t *testing.T) {
	localApp := mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	target := newH2CTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web/access/admin@demo.app/ping" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "a=1" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte("ok"))
	}))

	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()
	proxy.transport = target.Transport
	registerLocalWebApp(proxy, localApp, target.URL+testPathWebAccess, target.URL, []string{"admin@demo.app"})

	req := httptest.NewRequest(http.MethodGet, "/"+localApp.InstanceId()+"/admin@demo.app/ping?a=1", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.StatusCode, string(body))
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestHandleInRejectsInvalidInstanceScopedPath(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	req := httptest.NewRequest(http.MethodGet, "/admin@demo.app/ping", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestHandleInRejectsInvalidAppInstanceID(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid/admin@demo.app/ping", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestHandleInRejectsMissingWebName(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	req := httptest.NewRequest(http.MethodGet, "/11111111-1111-1111-1111-111111111111/", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestHandleInReturnsUnavailableWhenNoEndpoint(t *testing.T) {
	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()

	req := httptest.NewRequest(http.MethodGet, "/11111111-1111-1111-1111-111111111111/admin@demo.app/ping", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestHandleInForwardsToInprocLocalApp(t *testing.T) {
	localApp := mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")
	hostPath := coreapp.InprocHostPath(localApp.InstanceId())
	targetEndpoint := webinproc.Endpoint(hostPath, testPathWebAccess)
	webinproc.Register(targetEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin@demo.app/ping" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "a=1" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(func() {
		webinproc.Unregister(targetEndpoint)
	})

	proxy := &WebProxy{
		Context:   context.Background(),
		AppMinder: newTestAppMinder(),
	}
	proxy.DIInit()
	registerLocalWebApp(proxy, localApp, targetEndpoint, hostPath, []string{"admin@demo.app"})

	req := httptest.NewRequest(http.MethodGet, "/"+localApp.InstanceId()+"/admin@demo.app/ping?a=1", nil)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.StatusCode, string(body))
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestHandleInWebRequestInheritsContextDeadline(t *testing.T) {
	tests := []struct {
		name   string
		accept string
	}{
		{name: "ordinary request"},
		{name: "event stream", accept: "text/event-stream"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var targetDeadline time.Time
			localApp := mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111112")
			if index == 1 {
				localApp = mustWebProxyMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111113")
			}
			target := newH2CTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targetDeadline, _ = r.Context().Deadline()
				_, _ = w.Write([]byte("ok"))
			}))

			proxy := &WebProxy{
				Context:   context.Background(),
				AppMinder: newTestAppMinder(),
			}
			proxy.DIInit()
			proxy.transport = target.Transport
			registerLocalWebApp(proxy, localApp, target.URL+testPathWebAccess, target.URL, []string{"admin@demo.app"})

			req := httptest.NewRequest(http.MethodGet, "/"+localApp.InstanceId()+"/admin@demo.app/ping", nil)
			if test.accept != "" {
				req.Header.Set("Accept", test.accept)
			}
			requestDeadline := time.Now().Add(time.Hour)
			ctx, cancel := context.WithDeadline(req.Context(), requestDeadline)
			defer cancel()
			req = req.WithContext(ctx)
			recorder := httptest.NewRecorder()

			proxy.handleIn(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("unexpected status code: %d", recorder.Code)
			}
			if !targetDeadline.Equal(requestDeadline) {
				t.Fatalf("target deadline = %s, want %s", targetDeadline, requestDeadline)
			}
		})
	}
}
