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

func TestJoinPath(t *testing.T) {
	for _, tc := range []struct {
		parts []string
		want  string
	}{
		{[]string{}, ""},
		{[]string{"/orders", "/:id"}, "/orders/:id"},
		{[]string{"/orders/", "*path"}, "/orders/*path"},
		{[]string{"", "/health"}, "/health"},
		{[]string{"/orders", "../users"}, "/users"},
		{[]string{"/orders", "./items"}, "/orders/items"},
		{[]string{"/orders//", "//items/"}, "/orders/items"},
		{[]string{"/orders", "/"}, "/orders"},
		{[]string{"/", "/"}, "/"},
	} {
		if got := JoinPath(tc.parts...); got != tc.want {
			t.Fatalf("JoinPath(%q) = %q, want %q", tc.parts, got, tc.want)
		}
	}
}

func TestStripPathPrefix(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/invoke/demo.Service/Get?debug=1", nil)

	next := StripPathPrefix(request, "/invoke")

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

func TestValidatePathPrefix(t *testing.T) {
	for _, value := range []string{"", "/", "/app", "/app/", "/app/%2Fdocs"} {
		if err := ValidatePathPrefix(value); err != nil {
			t.Fatalf("expected %q to be valid: %v", value, err)
		}
	}
	for _, value := range []string{"//app", "https://example.com/app", "/app?x=1", "/app#section", "/./app", "/app/../admin", "/app path", "/app\\data"} {
		if err := ValidatePathPrefix(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
