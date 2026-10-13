package devproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/web/spec"
)

// A module graph arrives in a burst. The proxy must queue excess requests
// instead of overrunning the development server with upstream connections.
func TestDevProxyBoundsConcurrentRequests(t *testing.T) {
	var active atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		if current > devMaxConnsPerHost {
			http.Error(writer, "development server overloaded", http.StatusServiceUnavailable)
			return
		}
		time.Sleep(30 * time.Millisecond)
		_, _ = writer.Write([]byte("module source"))
	}))
	t.Cleanup(backend.Close)
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := devProxy(target)
	if transport, ok := proxy.Transport.(*http.Transport); ok {
		t.Cleanup(transport.CloseIdleConnections)
	}

	start := make(chan struct{})
	results := make(chan int, 128)
	for range cap(results) {
		go func() {
			<-start
			response := httptest.NewRecorder()
			proxy.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/src/module.ts", nil))
			results <- response.Code
		}()
	}
	close(start)
	failures := 0
	for range cap(results) {
		if status := <-results; status != http.StatusOK {
			failures++
		}
	}
	if failures > 0 {
		t.Fatalf("%d module requests failed under concurrent load", failures)
	}
}

func TestServerForwardsTheRequestToTheDevServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	devServer := newDevServer(t, "from source")
	server := newServerForTest(t, devServer)

	recorder := serveForTest(server, "/admin/assets/app.js?v=1")

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "from source" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if devServer.path != "/admin/assets/app.js" {
		t.Fatalf("unexpected target path: %s", devServer.path)
	}
	if devServer.query != "v=1" {
		t.Fatalf("unexpected target query: %s", devServer.query)
	}
}

// The request reaches the development server as the client sent it: a path the
// client escaped keeps its escaping instead of being re-encoded from the path the
// Web handler read.
func TestServerForwardsAnEscapedRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	devServer := newDevServer(t, "from source")
	server := newServerForTest(t, devServer)

	recorder := serveForTest(server, "/admin/a%2Fb")

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if devServer.requestURI != "/admin/a%2Fb" {
		t.Fatalf("unexpected target request URI: %s", devServer.requestURI)
	}
}

func TestServerFollowsTheDevServerToAnotherPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	first := newDevServer(t, "first")
	second := newDevServer(t, "second")
	first.writeStateFile(t, stateFilePath)
	server := serverForStateFile(stateFilePath)

	if body := serveForTest(server, "/").Body.String(); body != "first" {
		t.Fatalf("unexpected body: %s", body)
	}

	second.writeStateFile(t, stateFilePath)
	if body := serveForTest(server, "/").Body.String(); body != "second" {
		t.Fatalf("unexpected body after the port changed: %s", body)
	}
}

// A development server keeps answering a state file that is being rewritten, and
// one that was removed after the frontend stopped: the proxy that is running is
// the last server the file named.
func TestServerKeepsTheDevServerItHas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	devServer := newDevServer(t, "from source")
	devServer.writeStateFile(t, stateFilePath)
	server := serverForStateFile(stateFilePath)
	if body := serveForTest(server, "/").Body.String(); body != "from source" {
		t.Fatalf("unexpected body: %s", body)
	}

	if err := os.Remove(stateFilePath); err != nil {
		t.Fatalf("remove state file: %v", err)
	}

	recorder := serveForTest(server, "/")

	if recorder.Code != http.StatusOK || recorder.Body.String() != "from source" {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

// The Web handler is built for every request, so two servers serving one state
// file follow one development server: the second request uses the file the first
// read, not a state file of its own.
func TestServersServingTheSameStateFileFollowOneDevServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, time.Hour)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	devServer := newDevServer(t, "from source")
	devServer.writeStateFile(t, stateFilePath)
	first := serverForStateFile(stateFilePath)
	if body := serveForTest(first, "/").Body.String(); body != "from source" {
		t.Fatalf("unexpected body: %s", body)
	}
	if err := os.Remove(stateFilePath); err != nil {
		t.Fatalf("remove state file: %v", err)
	}
	second := serverForStateFile(stateFilePath)

	recorder := serveForTest(second, "/")

	if recorder.Code != http.StatusOK || recorder.Body.String() != "from source" {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestServerReportsAStateFileItCannotRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	server := serverForStateFile(stateFilePath)

	recorder := serveForTest(server, "/")

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "read development server state "+stateFilePath) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestServerReportsAnInvalidStateFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	for _, test := range []struct {
		name    string
		content string
		message string
	}{
		{name: "no port", content: `{"host": "127.0.0.1"}`, message: "invalid development server port 0"},
		{name: "broken json", content: `{"port":`, message: "parse development server state"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
			if err := os.WriteFile(stateFilePath, []byte(test.content), 0o644); err != nil {
				t.Fatalf("write state file: %v", err)
			}
			server := serverForStateFile(stateFilePath)

			recorder := serveForTest(server, "/")

			if recorder.Code != http.StatusBadGateway {
				t.Fatalf("unexpected status code: %d", recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), test.message) {
				t.Fatalf("unexpected body: %s", recorder.Body.String())
			}
		})
	}
}

func TestServerReportsAnUnreachableDevServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	host, port := closedDevServerAddress(t)
	writeStateFile(t, stateFilePath, host, port)
	server := serverForStateFile(stateFilePath)

	recorder := serveForTest(server, "/")

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), fmt.Sprintf("development server %s:%d is not reachable", host, port)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestServerRoutesServeTheMountRootAndEveryPath(t *testing.T) {
	for _, test := range []struct {
		basePath string
		paths    []string
	}{
		{basePath: "/", paths: []string{"/*path"}},
		{basePath: "/admin", paths: []string{"", "/*path"}},
	} {
		t.Run(test.basePath, func(t *testing.T) {
			router := spec.NewRouter(reflect.TypeFor[*Server](), test.basePath)
			server := new(Server)

			server.Routes(router)

			paths := []string{}
			for _, route := range router.Routes() {
				if !strings.HasSuffix(route.HandlerName(), ".Serve") {
					t.Fatalf("unexpected handler: %s", route.HandlerName())
				}
				if !slices.Contains(paths, route.Path()) {
					paths = append(paths, route.Path())
				}
			}
			if !reflect.DeepEqual(paths, test.paths) {
				t.Fatalf("unexpected paths: %#v", paths)
			}
		})
	}
}

// setStateReloadInterval lets a test read the state file for every request, so it
// does not wait for the interval a development run uses.
func setStateReloadInterval(t *testing.T, interval time.Duration) {
	t.Helper()
	previous := stateReloadInterval
	stateReloadInterval = interval
	t.Cleanup(func() { stateReloadInterval = previous })
}

func newServerForTest(t *testing.T, devServer *devServer) *Server {
	t.Helper()
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	devServer.writeStateFile(t, stateFilePath)
	return serverForStateFile(stateFilePath)
}

// serverForStateFile is what a Web handler does in DIInit: the state file is the
// only thing an instance carries, and the development server it names is kept per
// state file for the life of the process.
func serverForStateFile(stateFilePath string) *Server {
	server := new(Server)
	server.SetStateFile(stateFilePath)
	return server
}

func serveForTest(server *Server, target string) *httptest.ResponseRecorder {
	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local"+target, nil)
	server.GinCtx = ginCtx
	server.Serve()
	return recorder.ResponseRecorder
}

// closeNotifyRecorder gives a forwarding request the CloseNotifier a server's
// ResponseWriter carries, so the request context ends with the client.
type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
}

func (*closeNotifyRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

// devServer stands in for the frontend a development run starts: it
// records the request it was given and answers with one body.
type devServer struct {
	server     *httptest.Server
	path       string
	query      string
	requestURI string
}

func newDevServer(t *testing.T, body string) *devServer {
	t.Helper()
	dev := new(devServer)
	dev.server = httptest.NewTestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		dev.path = request.URL.Path
		dev.query = request.URL.RawQuery
		dev.requestURI = request.RequestURI
		_, _ = writer.Write([]byte(body))
	}))
	dev.server.Start()
	return dev
}

func (d *devServer) writeStateFile(t *testing.T, stateFilePath string) {
	t.Helper()
	parsed, err := url.Parse(d.server.URL)
	if err != nil {
		t.Fatalf("parse development server URL: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse development server port: %v", err)
	}
	writeStateFile(t, stateFilePath, parsed.Hostname(), port)
}

func writeStateFile(t *testing.T, stateFilePath string, host string, port int) {
	t.Helper()
	content := fmt.Sprintf(`{"host": %q, "port": %d}`, host, port)
	if err := os.WriteFile(stateFilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write state file: %v", err)
	}
}

// closedDevServerAddress returns an address nothing listens on, the way
// a frontend that stopped leaves its state file behind.
func closedDevServerAddress(t *testing.T) (string, int) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return "127.0.0.1", address.Port
}
