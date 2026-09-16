package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func getAsset(target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "http://hub.local"+target, nil)
	response := httptest.NewRecorder()
	DashboardHandler().ServeHTTP(response, request)
	return response
}
