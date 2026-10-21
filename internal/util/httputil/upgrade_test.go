package httputil

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestIsUpgradeRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/ws", nil)
	request.Header.Set("Connection", "keep-alive, Upgrade")
	request.Header.Set("Upgrade", "websocket")

	if !IsUpgradeRequest(request) {
		t.Fatal("expected upgrade request")
	}
}

func TestForwardUpgrade(t *testing.T) {
	log, _ := newTestProxyLogger(t, "daemon:link:webproxy")
	var gotPath string
	var gotQuery string
	target := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		hijacker := w.(http.Hijacker)
		conn, rw, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("Hijack() error = %v", err)
			return
		}
		defer conn.Close()

		_, _ = fmt.Fprint(rw, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		_ = rw.Flush()

		line, err := rw.ReadString('\n')
		if err != nil {
			t.Errorf("ReadString() error = %v", err)
			return
		}
		_, _ = fmt.Fprintf(rw, "echo:%s", line)
		_ = rw.Flush()
	}))
	target.Start()

	frontend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := ForwardUpgrade(w, r, target.URL+"/hmr?from=proxy", nil, log)
		if err != nil {
			t.Errorf("ForwardUpgrade() error = %v", err)
		}
	}))
	frontend.Start()

	conn, err := net.Dial("tcp", strings.TrimPrefix(frontend.URL, "http://"))
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	_, _ = fmt.Fprint(conn, "GET /client HTTP/1.1\r\nHost: demo.local\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}

	_, _ = fmt.Fprint(conn, "ping\n")
	got, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	if got != "echo:ping\n" {
		t.Fatalf("unexpected tunnel response: %s", got)
	}
	if gotPath != "/hmr" {
		t.Fatalf("unexpected target path: %s", gotPath)
	}
	if gotQuery != "from=proxy" {
		t.Fatalf("unexpected target query: %s", gotQuery)
	}
}

func TestUpgradeIdleReadWriteCloserClosesIdleStream(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newUpgradeStreamStub()
		idleStream := newUpgradeIdleReadWriteCloser(stream, 10*time.Millisecond)
		t.Cleanup(func() { _ = idleStream.Close() })

		synctest.Sleep(10 * time.Millisecond)
		select {
		case <-stream.closed:
		default:
			t.Fatal("idle upgrade stream was not closed")
		}
	})
}

func TestUpgradeIdleReadWriteCloserResetsOnTraffic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newUpgradeStreamStub()
		idleStream := newUpgradeIdleReadWriteCloser(stream, 200*time.Millisecond)
		t.Cleanup(func() { _ = idleStream.Close() })

		synctest.Sleep(100 * time.Millisecond)
		if _, err := idleStream.Write([]byte("ping")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		synctest.Sleep(199 * time.Millisecond)
		select {
		case <-stream.closed:
			t.Fatal("active upgrade stream was closed")
		default:
		}

		synctest.Sleep(time.Millisecond)
		select {
		case <-stream.closed:
		default:
			t.Fatal("upgrade stream was not closed after becoming idle")
		}
	})
}

func TestUpgradeIdleTransportWrapsSwitchingProtocolStream(t *testing.T) {
	stream := newUpgradeStreamStub()
	transport := NewUpgradeIdleTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusSwitchingProtocols,
			Body:       stream,
		}, nil
	}), time.Second)

	response, err := transport.RoundTrip(httptest.NewRequest(http.MethodGet, "http://demo.local/hmr", nil))
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer response.Body.Close()
	if _, ok := response.Body.(*_UpgradeIdleReadWriteCloser); !ok {
		t.Fatalf("response body type = %T, want idle upgrade stream", response.Body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type upgradeStreamStub struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func newUpgradeStreamStub() *upgradeStreamStub {
	return &upgradeStreamStub{closed: make(chan struct{})}
}

func (s *upgradeStreamStub) Read([]byte) (int, error) {
	<-s.closed
	return 0, io.EOF
}

func (*upgradeStreamStub) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (s *upgradeStreamStub) Close() error {
	s.closeOnce.Do(func() { close(s.closed) })
	return nil
}

func TestForwardUpgradeFailureLogging(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel bool
		err    error
		level  string
	}{
		{"client cancellation", true, context.Canceled, "DEBUG"},
		{"canceled EOF", true, io.EOF, "DEBUG"},
		{"upstream cancellation", false, context.Canceled, "ERROR"},
		{"upstream EOF", false, io.EOF, "ERROR"},
		{"deadline", false, context.DeadlineExceeded, "ERROR"},
		{"unrelated failure after cancellation", true, errors.New("dial failed"), "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log, logPath := newTestProxyLogger(t, "daemon:link:webproxy")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			request := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://demo.local/ws", nil)
			request.Header.Set("Connection", "Upgrade")
			request.Header.Set("Upgrade", "websocket")
			response := httptest.NewRecorder()
			transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if tc.cancel {
					cancel()
				}
				return nil, tc.err
			})
			if err := ForwardUpgrade(response, request, "http://backend.local/ws", transport, log); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want 502", response.Code)
			}
			record := readProxyLog(t, logPath)
			if record["logger"] != log.Name() || record["level"] != tc.level || record["error"] != tc.err.Error() {
				t.Fatalf("unexpected failure log: %v", record)
			}
		})
	}
}

func TestForwardUpgradeClientDisconnectCancelsPendingHandshake(t *testing.T) {
	log, logPath := newTestProxyLogger(t, "daemon:portal:webgw")
	backendEntered := make(chan struct{})
	backendCanceled := make(chan error, 1)
	backend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(backendEntered)
		// Keep the handshake pending until the actual network request is canceled.
		select {
		case <-r.Context().Done():
			backendCanceled <- r.Context().Err()
		case <-t.Context().Done():
		}
	}))
	backend.Start()

	proxyFinished := make(chan error, 1)
	frontend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyFinished <- ForwardUpgrade(w, r, backend.URL+"/ws", backend.Client().Transport, log)
	}))
	frontend.Start()

	connection, err := net.DialTimeout("tcp", strings.TrimPrefix(frontend.URL, "http://"), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if err := connection.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(connection, "GET /ws HTTP/1.1\r\nHost: demo.local\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-backendEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("backend did not receive the upgrade request")
	}

	// Closing the client socket must cancel both proxy hops without a manual cancel.
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-backendCanceled:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("backend context error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client disconnect did not cancel the backend handshake")
	}
	select {
	case err := <-proxyFinished:
		if err != nil {
			t.Fatalf("proxy handler returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("proxy handler did not exit after client disconnect")
	}
	record := readProxyLog(t, logPath)
	if record["level"] != "DEBUG" || record["msg"] != "web upgrade canceled" || record["logger"] != log.Name() {
		t.Fatalf("unexpected cancellation log: %v", record)
	}
}
