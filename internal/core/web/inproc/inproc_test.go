package inproc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
)

func resetRegistryForTest(t *testing.T) {
	t.Helper()

	registryMutex.Lock()
	prev := handlerByEndpoint
	handlerByEndpoint = map[string]*_Registration{}
	registryMutex.Unlock()
	t.Cleanup(func() {
		registryMutex.Lock()
		defer registryMutex.Unlock()
		handlerByEndpoint = prev
	})
}

func TestRegistrationCleanupIsIdempotentAndDoesNotRemoveReplacement(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/default@demo.app"
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	cleanupFirst := Register(endpoint, handler)
	Unregister(endpoint)
	cleanupSecond := Register(endpoint, handler)

	cleanupFirst()
	cleanupFirst()
	if _, ok := getHandler(endpoint); !ok {
		t.Fatal("stale cleanup removed replacement handler")
	}

	cleanupSecond()
	cleanupSecond()
	if registered, ok := getHandler(endpoint); ok {
		t.Fatalf("cleanup did not remove handler: %#v", registered)
	}
}

func TestRegistrySupportsConcurrentRegistrationsAndLookups(t *testing.T) {
	resetRegistryForTest(t)

	const count = 64
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	results := make(chan bool, count)
	var wg sync.WaitGroup
	for i := range count {
		endpoint := fmt.Sprintf("web+inproc://concurrent-%d", i)
		wg.Go(func() {
			cleanup := Register(endpoint, handler)
			_, registered := getHandler(endpoint)
			cleanup()
			_, remains := getHandler(endpoint)
			results <- registered && !remains
		})
	}
	wg.Wait()
	close(results)
	for result := range results {
		if !result {
			t.Fatal("concurrent registration lifecycle failed")
		}
	}
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

func TestRoundTripRejectsCanceledRequestBeforeCallingHandler(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/default@demo.app"
	handlerCalled := make(chan struct{}, 1)
	Register(endpoint, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled <- struct{}{}
	}))

	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	cancel()

	_, err := RoundTrip(endpoint, req)
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-handlerCalled:
		t.Fatal("handler was called for an already canceled request")
	default:
	}
}

func TestRoundTripPrefersCancellationOverConcurrentResponse(t *testing.T) {
	resetRegistryForTest(t)

	endpoint := "web+inproc://app/demo/web/access/default@demo.app"
	handlerStarted := make(chan struct{})
	Register(endpoint, http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		close(handlerStarted)
		<-req.Context().Done()
	}))

	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	errCh := make(chan error, 1)
	go func() {
		_, err := RoundTrip(endpoint, req)
		errCh <- err
	}()

	<-handlerStarted
	cancel()
	if err := <-errCh; err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoundTripStreamsResponseBeforeHandlerReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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

		buffer := make([]byte, len("data: ready\n\n"))
		_, _ = io.ReadFull(resp.Body, buffer)
		body := string(buffer)
		if body != "data: ready\n\n" {
			t.Fatalf("unexpected body: %s", body)
		}
	})
}
