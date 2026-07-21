package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/web/proxy"
)

func TestDashboardWebProxyForwardsRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(envDashboardDevProxy, "1")

	var gotPath string
	var gotQuery string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte("dashboard"))
	}))
	t.Cleanup(target.Close)

	restoreDashboardProxy := setDashboardProxyForTest(t, target.URL)
	t.Cleanup(restoreDashboardProxy)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/assets/app.js?v=1", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "dashboard" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if gotPath != "/assets/app.js" {
		t.Fatalf("unexpected proxy path: %s", gotPath)
	}
	if gotQuery != "v=1" {
		t.Fatalf("unexpected proxy query: %s", gotQuery)
	}
}

func TestDashboardWebSkipsProxyWhenDevProxyDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	t.Cleanup(target.Close)

	restoreDashboardProxy := setDashboardProxyForTest(t, target.URL)
	t.Cleanup(restoreDashboardProxy)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if called {
		t.Fatal("unexpected dashboard proxy call")
	}
	if !strings.Contains(recorder.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", recorder.Body.String())
	}
}

func TestDashboardWebServesEmbeddedIndexWhenProxyUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUnavailableDashboardProxy(t)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", recorder.Body.String())
	}
}

func TestDashboardWebServesEmbeddedAssetWhenProxyUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUnavailableDashboardProxy(t)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/brand/vinehub.png", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/brand/vinehub.png"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if body := recorder.Body.Bytes(); len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("expected embedded png asset")
	}
}

func TestDashboardWebFallsBackToEmbeddedIndexForSpaRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUnavailableDashboardProxy(t)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/portal/site", nil)
	ginCtx.Request.Header.Set("Accept", "text/html")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/portal/site"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", recorder.Body.String())
	}
}

func TestDashboardWebDoesNotFallbackToIndexForMissingAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withUnavailableDashboardProxy(t)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/vine.hub.DashboardWeb/assets/missing.js", nil)
	ginCtx.Request.Header.Set("Accept", "*/*")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/missing.js"}}

	handler := newDashboardWebServerTestHandler(ginCtx)
	handler.ProxyDashboard()

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
}

func (*closeNotifyRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

func mustParseDashboardProxyTestURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return parsed
}

func newDashboardWebServerTestHandler(ginCtx *gin.Context) *WebServerImpl {
	handler := &WebServerImpl{}
	handler.GinCtx = ginCtx
	return handler
}

func withUnavailableDashboardProxy(t *testing.T) {
	t.Helper()

	t.Setenv(envDashboardDevProxy, "1")
	restoreDashboardProxy := setDashboardProxyForTest(t, "http://127.0.0.1:1")
	t.Cleanup(restoreDashboardProxy)
}

func setDashboardProxyForTest(t *testing.T, rawURL string) func() {
	t.Helper()

	previousProxy := dashboardProxy
	dashboardProxy = proxy.NewReverseProxy(proxy.Option{
		Target:            mustParseDashboardProxyTestURL(t, rawURL),
		DialTimeout:       dashboardProxyDialTimeout,
		DetectionInterval: time.Hour,
	})
	return func() {
		dashboardProxy = previousProxy
	}
}
