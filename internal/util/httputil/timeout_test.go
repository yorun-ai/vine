package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestContextWithForwardTimeoutAddsDefaultHTTPTimeout(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ping", nil)
	ctx, cancel := ContextWithForwardTimeout(request)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected HTTP deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > DefaultHttpRequestTimeout {
		t.Fatalf("HTTP timeout = %s, want within %s", remaining, DefaultHttpRequestTimeout)
	}
}

func TestContextWithForwardTimeoutDoesNotLimitEventStreamDuration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/events", nil)
	request.Header.Set("Accept", "text/event-stream")
	ctx, cancel := ContextWithForwardTimeout(request)
	defer cancel()

	if deadline, ok := ctx.Deadline(); ok {
		t.Fatalf("event stream deadline = %s, want none", deadline)
	}
}
