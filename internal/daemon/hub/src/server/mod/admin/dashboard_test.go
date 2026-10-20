package admin

import (
	"bytes"

	"net/http"
	"net/http/httptest"

	"strings"
	"sync"
	"testing"
)

func TestDashboardHandlerReturnsNotFoundForMissingFile(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local/assets/missing.js", nil)
	request.Header.Set("Accept", "*/*")
	response := httptest.NewRecorder()
	dashboardHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing asset status code: %d", response.Code)
	}
}

func TestDashboardServesEmbeddedBuildOutput(t *testing.T) {
	content, err := dashboardFS.ReadFile("dashboard/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	handler := dashboardHandler()
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(method, "http://hub.local/status/task-queue", nil)
		request.Header.Set("Accept", "text/html")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "" ||
			response.Header().Get("Vary") != "Accept-Encoding" || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("unexpected %s response: %d %v", method, response.Code, response.Header())
		}
		if method == http.MethodGet && !bytes.Equal(response.Body.Bytes(), content) {
			t.Fatal("response must contain the exact embedded HTML bytes")
		}
		if method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatal("HEAD response must have no body")
		}
	}

	// The same handler must keep independent state for concurrent requests.
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			request := httptest.NewRequest(http.MethodGet, "http://hub.local/brand/vinehub.png", nil)
			request.Header.Set("Accept-Encoding", "br")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "" ||
				!bytes.HasPrefix(response.Body.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
				t.Errorf("unexpected PNG response: %d %v", response.Code, response.Header())
			}
		})
	}
	wg.Wait()
}

func TestDashboardDevProxyRequiresEnvironment(t *testing.T) {
	t.Setenv(dashboardDevProxyEnv, "")
	if proxy := dashboardDevProxy(); proxy != nil {
		proxy.Close()
		t.Fatal("empty environment must disable development server probing")
	}
}

func TestDashboardDevProxyPrefersAvailableServer(t *testing.T) {
	t.Setenv(dashboardDevProxyEnv, "1")
	devServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("development " + request.URL.Path))
	}))
	t.Cleanup(devServer.Close)
	originalURL := dashboardDevServerURL
	dashboardDevServerURL = devServer.URL
	t.Cleanup(func() { dashboardDevServerURL = originalURL })
	proxy := dashboardDevProxy()
	if proxy == nil {
		t.Fatal("environment must enable probing even with embedded assets")
	}
	t.Cleanup(proxy.Close)
	server := &Server{
		rpcHTTPHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte("admin api"))
		}),
		dashboardHandler:  dashboardHandler(),
		dashboardDevProxy: proxy,
	}
	get := func(path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "http://hub.local"+path, nil)
		request.Header.Set("Accept", "text/html")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		return response
	}
	if response := get("/app/config"); response.Code != http.StatusOK || response.Body.String() != "development /app/config" {
		t.Fatalf("expected development server priority: %d %s", response.Code, response.Body.String())
	}
	if response := get("/api/invoke/InfoService/GetInfo"); response.Body.String() != "admin api" {
		t.Fatalf("Admin API must stay on Hub: %s", response.Body.String())
	}
	devServer.Close()
	proxy.refreshAvailable()
	if response := get("/app/config"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded fallback: %d %s", response.Code, response.Body.String())
	}
}
