package httputil

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
