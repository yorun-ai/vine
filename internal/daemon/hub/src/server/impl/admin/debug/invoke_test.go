package debug

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type _DebugRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f _DebugRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDoServiceDebugInvokeRequestKeepsH2CContextUntilBodyRead(t *testing.T) {
	releaseBody := make(chan struct{})
	var releaseBodyOnce sync.Once
	release := func() { releaseBodyOnce.Do(func() { close(releaseBody) }) }
	t.Cleanup(release)
	server := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-releaseBody
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), &http2.Server{}))
	t.Cleanup(server.Close)

	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network string, addr string, _ *tls.Config) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
	}
	t.Cleanup(transport.CloseIdleConnections)

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	response, err := doServiceDebugInvokeRequest(request, transport)
	if err != nil {
		t.Fatalf("doServiceDebugInvokeRequest() error = %v", err)
	}
	release()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q, want %q", body, `{"ok":true}`)
	}
}

func TestDoServiceDebugInvokeRequestCancelsContextOnRoundTripError(t *testing.T) {
	expectedErr := errors.New("round trip failed")
	var forwardedContext context.Context
	transport := _DebugRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		forwardedContext = request.Context()
		return nil, expectedErr
	})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://link.invalid", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	_, err = doServiceDebugInvokeRequest(request, transport)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("doServiceDebugInvokeRequest() error = %v, want %v", err, expectedErr)
	}
	select {
	case <-forwardedContext.Done():
	default:
		t.Fatal("forwarded context was not canceled")
	}
}
