package httputil

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
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
	var gotPath string
	var gotQuery string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer target.Close()

	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := ForwardUpgrade(w, r, target.URL+"/hmr?from=proxy", nil)
		if err != nil {
			t.Errorf("ForwardUpgrade() error = %v", err)
		}
	}))
	defer frontend.Close()

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
	stream := newUpgradeStreamStub()
	idleStream := newUpgradeIdleReadWriteCloser(stream, 10*time.Millisecond)
	t.Cleanup(func() { _ = idleStream.Close() })

	select {
	case <-stream.closed:
	case <-time.After(time.Second):
		t.Fatal("idle upgrade stream was not closed")
	}
}

func TestUpgradeIdleReadWriteCloserResetsOnTraffic(t *testing.T) {
	stream := newUpgradeStreamStub()
	idleStream := newUpgradeIdleReadWriteCloser(stream, 200*time.Millisecond)
	t.Cleanup(func() { _ = idleStream.Close() })

	time.Sleep(100 * time.Millisecond)
	if _, err := idleStream.Write([]byte("ping")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	select {
	case <-stream.closed:
		t.Fatal("active upgrade stream was closed")
	case <-time.After(150 * time.Millisecond):
	}
	select {
	case <-stream.closed:
	case <-time.After(time.Second):
		t.Fatal("upgrade stream was not closed after becoming idle")
	}
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
