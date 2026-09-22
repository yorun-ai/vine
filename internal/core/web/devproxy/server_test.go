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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/web/spec"
)

func TestServerForwardsTheRequestToTheDevelopmentServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	developmentServer := newDevelopmentServer(t, "from source")
	server := newServerForTest(t, developmentServer)

	recorder := serveForTest(server, "/admin/assets/app.js?v=1")

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "from source" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if developmentServer.path != "/admin/assets/app.js" {
		t.Fatalf("unexpected target path: %s", developmentServer.path)
	}
	if developmentServer.query != "v=1" {
		t.Fatalf("unexpected target query: %s", developmentServer.query)
	}
}

// The request reaches the development server as the client sent it: a path the
// client escaped keeps its escaping instead of being re-encoded from the path the
// Web handler read.
func TestServerForwardsAnEscapedRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	developmentServer := newDevelopmentServer(t, "from source")
	server := newServerForTest(t, developmentServer)

	recorder := serveForTest(server, "/admin/a%2Fb")

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if developmentServer.requestURI != "/admin/a%2Fb" {
		t.Fatalf("unexpected target request URI: %s", developmentServer.requestURI)
	}
}

func TestServerFollowsTheDevelopmentServerToAnotherPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	first := newDevelopmentServer(t, "first")
	second := newDevelopmentServer(t, "second")
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
func TestServerKeepsTheDevelopmentServerItHas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	developmentServer := newDevelopmentServer(t, "from source")
	developmentServer.writeStateFile(t, stateFilePath)
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
func TestServersServingTheSameStateFileFollowOneDevelopmentServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, time.Hour)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	developmentServer := newDevelopmentServer(t, "from source")
	developmentServer.writeStateFile(t, stateFilePath)
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

func TestServerReportsAnUnreachableDevelopmentServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setStateReloadInterval(t, 0)
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	host, port := closedDevelopmentServerAddress(t)
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

func newServerForTest(t *testing.T, developmentServer *developmentServer) *Server {
	t.Helper()
	stateFilePath := filepath.Join(t.TempDir(), "dev-state.json")
	developmentServer.writeStateFile(t, stateFilePath)
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

// developmentServer stands in for the frontend a development run starts: it
// records the request it was given and answers with one body.
type developmentServer struct {
	server     *httptest.Server
	path       string
	query      string
	requestURI string
}

func newDevelopmentServer(t *testing.T, body string) *developmentServer {
	t.Helper()
	development := new(developmentServer)
	development.server = httptest.NewTestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		development.path = request.URL.Path
		development.query = request.URL.RawQuery
		development.requestURI = request.RequestURI
		_, _ = writer.Write([]byte(body))
	}))
	development.server.Start()
	return development
}

func (d *developmentServer) writeStateFile(t *testing.T, stateFilePath string) {
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

// closedDevelopmentServerAddress returns an address nothing listens on, the way
// a frontend that stopped leaves its state file behind.
func closedDevelopmentServerAddress(t *testing.T) (string, int) {
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
