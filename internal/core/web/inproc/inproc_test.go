package inproc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func resetRegistryForTest(t *testing.T) {
	t.Helper()

	prev := handlerByEndpoint
	handlerByEndpoint = map[string]http.Handler{}
	t.Cleanup(func() {
		handlerByEndpoint = prev
	})
}

func TestRegisterAndRoundTrip(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/default@demo.app"
	Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-path", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/web/access/default@demo.app/ping", nil)
	resp, err := RoundTrip(endpoint, req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if got := resp.Header.Get("x-path"); got != "/web/access/default@demo.app/ping" {
		t.Fatalf("unexpected path header: %s", got)
	}
}

func TestRoundTripReturnsErrorForMissingEndpoint(t *testing.T) {
	resetRegistryForTest(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := RoundTrip("web+inproc://app/missing/web/access/demo", req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoundTripRespectsRequestContext(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/default@demo.app"
	Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	cancel()

	_, err := RoundTrip(endpoint, req)
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoundTripStreamsResponseBeforeHandlerReturns(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/stream@demo.app"
	handlerDone := make(chan struct{})
	Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: ready\n\n"))
		w.(http.Flusher).Flush()
		<-handlerDone
	}))
	t.Cleanup(func() {
		close(handlerDone)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := RoundTrip(endpoint, req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	bodyCh := make(chan string, 1)
	go func() {
		buffer := make([]byte, len("data: ready\n\n"))
		_, _ = io.ReadFull(resp.Body, buffer)
		bodyCh <- string(buffer)
	}()

	select {
	case body := <-bodyCh:
		if body != "data: ready\n\n" {
			t.Fatalf("unexpected body: %s", body)
		}
	case <-time.After(time.Second):
		t.Fatal("stream response was not available before handler returned")
	}
}
