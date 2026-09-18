package admin

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	coreapp "go.yorun.ai/vine/internal/core/app"
)

func TestDashboardHandlerServesEmbeddedIndex(t *testing.T) {
	response := getAsset("/")
	if dashboardAssets == nil {
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "script/dev-hub-dashboard.sh") {
			t.Fatalf("expected development hint, got %d: %s", response.Code, response.Body.String())
		}
		return
	}
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", response.Body.String())
	}
}

func TestDashboardHandlerServesEmbeddedAsset(t *testing.T) {
	response := getAsset("/brand/vinehub.png")
	if dashboardAssets == nil {
		if response.Code != http.StatusNotFound {
			t.Fatalf("unexpected status code: %d", response.Code)
		}
		return
	}

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
	if dashboardAssets == nil {
		if response.Code != http.StatusNotFound {
			t.Fatalf("unexpected status code: %d", response.Code)
		}
		return
	}

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected embedded dashboard index, got: %s", response.Body.String())
	}
}

func TestDashboardHandlerReturnsNotFoundForMissingFile(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local/assets/missing.js", nil)
	request.Header.Set("Accept", "*/*")
	response := httptest.NewRecorder()
	dashboardHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing asset status code: %d", response.Code)
	}
}

// dashboardDevProxy serves a development server beside Hub when a developer runs
// the local Dashboard script, so a Dashboard source change needs no build.
func TestDashboardDevProxyServesTheDevelopmentServer(t *testing.T) {
	if dashboardAssets != nil {
		t.Skip("embedded builds do not probe a development server")
	}
	devServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("development build " + request.URL.Path))
	}))
	t.Cleanup(devServer.Close)
	t.Cleanup(func() { dashboardDevServerURL = "http://localhost:7098" })
	dashboardDevServerURL = devServer.URL

	devProxy := dashboardDevProxy()
	t.Cleanup(devProxy.Close)
	server := &Server{rpcHTTPHandler: http.NotFoundHandler(), dashboardHandler: dashboardHandler(), dashboardDevProxy: devProxy}

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

func TestDashboardDevProxyReturnsDevelopmentHintWhenUnavailable(t *testing.T) {
	if dashboardAssets != nil {
		t.Skip("embedded builds do not probe a development server")
	}
	t.Cleanup(func() { dashboardDevServerURL = "http://localhost:7098" })
	// Nothing listens on the port the development server script would use.
	dashboardDevServerURL = "http://127.0.0.1:1"

	devProxy := dashboardDevProxy()
	t.Cleanup(devProxy.Close)
	server := &Server{rpcHTTPHandler: http.NotFoundHandler(), dashboardHandler: dashboardHandler(), dashboardDevProxy: devProxy}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hub.local/app/config", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "script/dev-hub-dashboard.sh") {
		t.Fatalf("expected the development hint, got: %d %s", response.Code, response.Body.String())
	}
}

func TestDashboardDevProxyIsAutomaticWithoutEmbeddedBuild(t *testing.T) {
	if dashboardAssets != nil {
		if dashboardDevProxy() != nil {
			t.Fatal("embedded builds must not probe the development server")
		}
		return
	}
	devProxy := dashboardDevProxy()
	if devProxy == nil {
		t.Fatal("expected automatic development proxy")
	}
	t.Cleanup(devProxy.Close)
}

// TestDashboardBuildCallsTheAdminApiPath guards the path the listener serves: the
// Admin API answers one path there and the Dashboard build on every other, so the
// build Hub hands the browser has to call the API path. A build that called
// another path would reach the entry document instead of the API.
func TestDashboardBuildCallsTheAdminApiPath(t *testing.T) {
	if dashboardAssets == nil {
		t.Skip("Dashboard assets are generated for embedded builds")
	}
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
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	dashboardHandler().ServeHTTP(response, request)
	return response
}

func TestDashboardAssetsRequireAnEntryDocument(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
		mode fs.FileMode
		want bool
	}{
		{name: "placeholder only", file: ".gitkeep"},
		{name: "leftover asset", file: "assets/old.js"},
		{name: "directory is not an entry document", file: "index.html", mode: fs.ModeDir},
		{name: "plain entry", file: "index.html", want: true},
		{name: "Brotli entry", file: "index.html.br", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := fstest.MapFS{
				"assets/dashboard/.gitkeep": &fstest.MapFile{},
			}
			files["assets/dashboard/"+tc.file] = &fstest.MapFile{Mode: tc.mode}
			if got := newDashboardAssets(files) != nil; got != tc.want {
				t.Fatalf("embedded Dashboard detected = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDashboardDoesNotServePlaceholder(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local/.gitkeep", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	dashboardHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("placeholder status = %d, want 404", response.Code)
	}
}

func TestDashboardServesEmbeddedBrotliWithoutRecompression(t *testing.T) {
	if dashboardAssets == nil {
		t.Skip("Dashboard assets have not been built")
	}
	compressed, err := dashboardFS.ReadFile("assets/dashboard/index.html.br")
	if err != nil {
		t.Fatal(err)
	}
	handler := dashboardHandler()
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(method, "http://hub.local/status/task-queue", nil)
		request.Header.Set("Accept", "text/html")
		request.Header.Set("Accept-Encoding", "br")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "br" ||
			response.Header().Get("Vary") != "Accept-Encoding" || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("unexpected %s response: %d %v", method, response.Code, response.Header())
		}
		if method == http.MethodGet && !bytes.Equal(response.Body.Bytes(), compressed) {
			t.Fatal("response must contain the exact embedded Brotli bytes")
		}
		if method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatal("HEAD response must have no body")
		}
	}

	// The same handler must keep independent state for concurrent requests.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
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

func TestDashboardBundleContainsOneRepresentationPerFile(t *testing.T) {
	err := fs.WalkDir(dashboardFS, "assets/dashboard", func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.Contains(filename, "THIRD_PARTY_LICENSES") {
			t.Errorf("unexpected generated license file: %s", filename)
		}
		if strings.HasSuffix(filename, ".br") {
			if _, err := fs.Stat(dashboardFS, strings.TrimSuffix(filename, ".br")); err == nil {
				t.Errorf("both original and Brotli embedded: %s", filename)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
