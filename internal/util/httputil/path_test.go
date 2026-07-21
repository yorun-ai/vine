package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathPrefix(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"invoke/demo.Service/Get", "/invoke"},
		{"/invoke/demo.Service/Get", "/invoke"},
		{"/inspect", "/inspect"},
	} {
		if got := PathPrefix(tc.path); got != tc.want {
			t.Fatalf("PathPrefix(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestTrimPathPrefix(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/invoke/demo.Service/Get?debug=1", nil)

	next := TrimPathPrefix(request, "/invoke")

	if next.URL.Path != "/demo.Service/Get" {
		t.Fatalf("unexpected path: %s", next.URL.Path)
	}
	if next.URL.RawQuery != "debug=1" {
		t.Fatalf("unexpected raw query: %s", next.URL.RawQuery)
	}
	if request.URL.Path != "/invoke/demo.Service/Get" {
		t.Fatalf("original request was mutated: %s", request.URL.Path)
	}
}
