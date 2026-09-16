package admin

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	coreapp "go.yorun.ai/vine/internal/core/app"
)

func TestDashboardHandlerServesEmbeddedIndex(t *testing.T) {
	response := getAsset("/")

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", response.Body.String())
	}
}

func TestDashboardHandlerServesEmbeddedAsset(t *testing.T) {
	response := getAsset("/brand/vinehub.png")

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if body := response.Body.Bytes(); len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("expected embedded png asset")
	}
}

func TestDashboardHandlerFallsBackToIndexForSpaRoute(t *testing.T) {
	response := getAsset("/settings/dashboard-port")

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", response.Body.String())
	}
}

func TestDashboardHandlerFallsBackToIndexForMissingFile(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local/assets/missing.js", nil)
	response := httptest.NewRecorder()

	DashboardHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", response.Body.String())
	}
}

// DashboardDevProxy serves a development server beside Hub when a developer asks
// for one, so a Dashboard source change needs no build, and Hub serves the
// embedded build while that server is not running.
func TestDashboardDevProxyServesTheDevelopmentServer(t *testing.T) {
	devServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("development build " + request.URL.Path))
	}))
	t.Cleanup(devServer.Close)
	t.Cleanup(func() { dashboardDevServerURL = "http://localhost:7098" })
	dashboardDevServerURL = devServer.URL

	t.Setenv(dashboardDevProxyEnv, "1")
	devProxy := DashboardDevProxy()
	t.Cleanup(devProxy.Close)
	server := &Server{rpcHTTPHandler: http.NotFoundHandler(), dashboardHandler: DashboardHandler(), dashboardDevProxy: devProxy}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hub.local/app/config", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if body := response.Body.String(); body != "development build /app/config" {
		t.Fatalf("expected the development server response, got: %s", body)
	}

	// The Admin API keeps answering Hub, not the development server.
	response = httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://hub.local/api/invoke/InfoService/GetInfo", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected api status code: %d", response.Code)
	}
}

func TestDashboardDevProxyFallsBackToTheEmbeddedBuild(t *testing.T) {
	t.Cleanup(func() { dashboardDevServerURL = "http://localhost:7098" })
	// Nothing listens on the port the development server script would use.
	dashboardDevServerURL = "http://127.0.0.1:1"

	t.Setenv(dashboardDevProxyEnv, "1")
	devProxy := DashboardDevProxy()
	t.Cleanup(devProxy.Close)
	server := &Server{rpcHTTPHandler: http.NotFoundHandler(), dashboardHandler: DashboardHandler(), dashboardDevProxy: devProxy}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hub.local/app/config", nil))
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected the embedded Dashboard build, got: %s", response.Body.String())
	}
}

// Without the environment variable Hub serves only the embedded build, whatever
// runs on the development port.
func TestDashboardDevProxyIsOffByDefault(t *testing.T) {
	t.Setenv(dashboardDevProxyEnv, "")
	if DashboardDevProxy() != nil {
		t.Fatal("expected no development proxy")
	}
}

// TestDashboardBuildCallsTheAdminApiPath guards the path the listener serves: the
// Admin API answers one path there and the Dashboard build on every other, so the
// build Hub hands the browser has to call the API path. A build that called
// another path would reach the entry document instead of the API.
func TestDashboardBuildCallsTheAdminApiPath(t *testing.T) {
	script := dashboardScriptPath(getAsset("/").Body.String())
	if script == "" {
		t.Fatal("expected the index to load a build script")
	}
	body := getAsset(script).Body.String()
	if !strings.Contains(body, apiPath) {
		t.Fatalf("expected the Dashboard build to call %s", apiPath)
	}
	if strings.Contains(body, coreapp.PathRpcInvoke) {
		t.Fatalf("expected the Dashboard build to call %s, not the runtime path %s", apiPath, coreapp.PathRpcInvoke)
	}
}

// dashboardScriptPath returns the build script the entry document loads.
func dashboardScriptPath(index string) string {
	matches := regexp.MustCompile(`src="([^"]+\.js)"`).FindStringSubmatch(index)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func getAsset(target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local"+target, nil)
	response := httptest.NewRecorder()
	DashboardHandler().ServeHTTP(response, request)
	return response
}
