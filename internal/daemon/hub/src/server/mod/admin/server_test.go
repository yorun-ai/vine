package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServerServesAdminAPIAndDashboardBuild pins the routing of the admin
// listener: the Admin API answers its own path, and every other path serves the
// embedded Dashboard build.
func TestServerServesAdminAPIAndDashboardBuild(t *testing.T) {
	apiPath := ""
	server := &Server{
		rpcHTTPHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			apiPath = request.URL.Path
			writer.WriteHeader(http.StatusOK)
		}),
		dashboardHandler: DashboardHandler(),
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://hub.local/rpc/invoke/InfoService/GetInfo", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected api status code: %d", response.Code)
	}
	if apiPath != "/InfoService/GetInfo" {
		t.Fatalf("unexpected api path: %q", apiPath)
	}

	response = httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hub.local/portal/site", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected dashboard status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected the Dashboard build, got: %s", response.Body.String())
	}
}
