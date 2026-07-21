package entry

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPPrefersFirstForwardedForIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local", nil)
	request.Header.Set("X-Forwarded-For", " 203.0.113.10, 203.0.113.11")
	request.Header.Set("X-Real-IP", "203.0.113.20")
	request.RemoteAddr = "203.0.113.30:1234"

	if got := clientIP(request); got != "203.0.113.10" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPFallsBackToRealIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local", nil)
	request.Header.Set("X-Real-IP", "203.0.113.20")
	request.RemoteAddr = "203.0.113.30:1234"

	if got := clientIP(request); got != "203.0.113.20" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPFallsBackToRemoteAddr(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local", nil)
	request.RemoteAddr = "203.0.113.30:1234"

	if got := clientIP(request); got != "203.0.113.30" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}
