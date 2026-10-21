package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/web/assets"
	"go.yorun.ai/vine/internal/core/web/devproxy"
	"go.yorun.ai/vine/internal/core/web/spec"
	vinehttp "go.yorun.ai/vine/internal/util/httputil"
	"golang.org/x/net/websocket"
)

type _ServerTestHandler struct {
	_DefaultServerTestWebServer
}

func (h *_ServerTestHandler) Routes(r *spec.Router) {
	r.GET("/ping", h.Ping)
}

func (*_ServerTestHandler) Ping() {}

type _ServerTestWebServer interface {
	spec.Handler

	mustBeServerTestWebServer()
}

type _DefaultServerTestWebServer struct {
}

func (*_DefaultServerTestWebServer) Routes(*spec.Router) {
	panic("method routes is not implemented")
}

func (*_DefaultServerTestWebServer) mustBeServerTestWebServer() {}

type testAssetsWeb struct {
	_DefaultServerTestWebServer
	assets.Server
}

func (h *testAssetsWeb) DIInit() {
	h.Server.SetAccessor(assets.NewEmbedAccessor(fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("index")},
		"dist/app.js":     &fstest.MapFile{Data: []byte("script")},
	}, "dist"))
}

func (h *testAssetsWeb) Routes(r *spec.Router) {
	h.Server.Routes(r)
}

// Exercise method promotion, execution-scoped injection and DIInit together.
// Calling ServeAsset directly would miss a broken generated handler's Routes.
func TestServerServesEmbeddedAssetsThroughContainer(t *testing.T) {
	for _, mount := range []string{"", "/", "/main", "/nested/main"} {
		t.Run("mount="+mount, func(t *testing.T) {
			spec.ResetRegistryForTest()
			t.Cleanup(spec.ResetRegistryForTest)
			spec.Register(&spec.WebSpec{
				Name: "ServerTestWeb", SkelName: "demo.user.ServerTestWeb", MountPath: mount,
				ServerType:        reflect.TypeFor[_ServerTestWebServer](),
				DefaultServerType: reflect.TypeFor[*_DefaultServerTestWebServer](),
			})
			server := NewServer(Option{HandlerTypes: []reflect.Type{reflect.TypeFor[*testAssetsWeb]()}})
			base := strings.TrimSuffix(mount, "/")
			for _, tc := range []struct {
				method string
				path   string
				accept string
				status int
				body   string
			}{
				{http.MethodGet, base + "/", "", 200, "index"},
				{http.MethodGet, base + "/index.html", "", 200, "index"},
				{http.MethodGet, base + "/app.js?v=1", "", 200, "script"},
				{http.MethodHead, base + "/app.js", "", 200, ""},
				{http.MethodGet, base + "/missing.js", "", 404, ""},
				{http.MethodGet, base + "/page", "text/html", 200, "index"},
				{http.MethodPost, base + "/", "", 405, ""},
			} {
				request := newAssetsWebRequest(t, tc.method, "/demo.user.ServerTestWeb"+tc.path)
				request.Header.Set("Accept", tc.accept)
				recorder := httptest.NewRecorder()
				server.HTTPHandler().ServeHTTP(recorder, request)
				if recorder.Code != tc.status || recorder.Body.String() != tc.body {
					t.Fatalf("%s %s: status=%d body=%q", tc.method, tc.path, recorder.Code, recorder.Body.String())
				}
			}
			if base != "" {
				for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
					recorder := httptest.NewRecorder()
					server.HTTPHandler().ServeHTTP(recorder, newAssetsWebRequest(t, method, "/demo.user.ServerTestWeb"+base))
					want := http.StatusOK
					if method == http.MethodPost {
						want = http.StatusMethodNotAllowed
					}
					if recorder.Code != want || recorder.Header().Get("Location") != "" {
						t.Fatalf("mount root %s: status=%d location=%q", method, recorder.Code, recorder.Header().Get("Location"))
					}
				}
			}
		})
	}
}

func newAssetsWebRequest(t *testing.T, method string, path string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	initiator, err := meta.NewInitiator("portal.app", "1.0.0", "123e4567-e89b-12d3-a456-426614174000", "http", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	encodeTestTraceToHeader(request.Header, meta.InitialTrace())
	encodeTestInitiatorToHeader(request.Header, initiator)
	encodeTestActorToHeader(request.Header, meta.NewAnonymousActor())
	return request
}

func TestStripWebNamePreservesRequestURI(t *testing.T) {
	for _, tc := range []struct {
		input string
		path  string
		uri   string
	}{
		{"/demo.Web/main/a%2Fb?q=a%2Fb", "/main/a/b", "/main/a%2Fb?q=a%2Fb"},
		{"/demo.Web/main/a%20b", "/main/a b", "/main/a%20b"},
		{"/demo.Web/main?", "/main", "/main?"},
		{"/demo.Web", "/", "/"},
		{"/%64emo.Web/main/a%2Fb", "/main/a/b", "/main/a/b"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, tc.input, nil)
			stripWebName("/demo.Web")(ctx)
			if ctx.Request.URL.Path != tc.path || ctx.Request.RequestURI != tc.uri || ctx.Request.URL.RequestURI() != tc.uri {
				t.Fatalf("path=%q rawPath=%q URI=%q", ctx.Request.URL.Path, ctx.Request.URL.RawPath, ctx.Request.RequestURI)
			}
		})
	}
}

// _SecondServerTestHandler is a second Web of the same application: the server
// serves every Web through its own Gin group, so a test can pin that one Web
// answers without reaching the routes of the other.
type _SecondServerTestHandler struct {
	_DefaultSecondServerTestWebServer
}

func (h *_SecondServerTestHandler) Routes(r *spec.Router) {
	r.GET("/ping", h.Ping)
}

func (*_SecondServerTestHandler) Ping() {}

type _SecondServerTestWebServer interface {
	spec.Handler

	mustBeSecondServerTestWebServer()
}

type _DefaultSecondServerTestWebServer struct {
}

func (*_DefaultSecondServerTestWebServer) Routes(*spec.Router) {
	panic("method routes is not implemented")
}

func (*_DefaultSecondServerTestWebServer) mustBeSecondServerTestWebServer() {}

func registerServerTestWeb() {
	spec.Register(&spec.WebSpec{
		Name:              "ServerTestWeb",
		SkelName:          "demo.user.ServerTestWeb",
		ServerType:        reflect.TypeFor[_ServerTestWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultServerTestWebServer](),
	})
	spec.Register(&spec.WebSpec{
		Name:              "SecondServerTestWeb",
		SkelName:          "demo.user.SecondServerTestWeb",
		MountPath:         "/console",
		ServerType:        reflect.TypeFor[_SecondServerTestWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultSecondServerTestWebServer](),
	})
}

type _ServerTestInvalidHandler struct{}

func (*_ServerTestInvalidHandler) Routes(*spec.Router) {}

func (*_ServerTestInvalidHandler) mustBeServerTestWebServer() {}

type _ServerTestExecutor struct {
	initRoutes []spec.Route
	executed   []spec.Route
	contexts   []*gin.Context
	execute    func(route spec.Route, ginCtx *gin.Context)
}

func (e *_ServerTestExecutor) Init(routes []spec.Route) {
	e.initRoutes = append([]spec.Route{}, routes...)
}

func (e *_ServerTestExecutor) Execute(route spec.Route, ginCtx *gin.Context) {
	if e.execute != nil {
		e.execute(route, ginCtx)
		return
	}
	e.executed = append(e.executed, route)
	e.contexts = append(e.contexts, ginCtx)
	ginCtx.Status(http.StatusNoContent)
}

func newServerTestServer(executor Executor) *Server {
	spec.ResetRegistryForTest()
	registerServerTestWeb()

	return NewServer(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_ServerTestHandler]()},
		Executor:     executor,
	})
}

func TestServerDelegatesToExecutor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	executor := &_ServerTestExecutor{}
	server := newServerTestServer(executor)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	if len(executor.initRoutes) != 1 {
		t.Fatalf("expected executor init with 1 route, got %d", len(executor.initRoutes))
	}
	if len(executor.executed) != 1 {
		t.Fatalf("expected executor execute once, got %d", len(executor.executed))
	}
	if executor.executed[0].Method() != http.MethodGet || executor.executed[0].Path() != "/demo.user.ServerTestWeb/ping" {
		t.Fatalf("unexpected route: %#v", executor.executed[0])
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestServerRoutesExposeRealHandlerName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newServerTestServer(&_ServerTestExecutor{})

	routes := server.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !strings.Contains(routes[0].HandlerName(), "_ServerTestHandler") ||
		!strings.HasSuffix(routes[0].HandlerName(), ".Ping") {
		t.Fatalf("unexpected handler name: %s", routes[0].HandlerName())
	}
}

// TestServerServesEveryWebOfTheApplication pins the boundary between the Webs of
// one application: they share the Gin instance, and each one answers on its own
// group without reaching the routes of the other. A handler reads the path its
// Web serves: the Web name Link and the application dispatch on is gone, and the
// mount the Web declares stays.
func TestServerServesEveryWebOfTheApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spec.ResetRegistryForTest()
	registerServerTestWeb()

	var executed []string
	var served []string
	server := NewServer(Option{
		HandlerTypes: []reflect.Type{
			reflect.TypeFor[*_ServerTestHandler](),
			reflect.TypeFor[*_SecondServerTestHandler](),
		},
		Executor: &_ServerTestExecutor{
			execute: func(route spec.Route, ginCtx *gin.Context) {
				executed = append(executed, route.Path())
				served = append(served, ginCtx.Request.URL.Path)
				ginCtx.Status(http.StatusNoContent)
			},
		},
	})

	for _, path := range []string{"/demo.user.ServerTestWeb/ping", "/demo.user.SecondServerTestWeb/console/ping"} {
		recorder := httptest.NewRecorder()
		server.HTTPHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %s: unexpected status code: %d", path, recorder.Code)
		}
	}
	assert.Equal(t, []string{"/demo.user.ServerTestWeb/ping", "/demo.user.SecondServerTestWeb/console/ping"}, executed)
	assert.Equal(t, []string{"/ping", "/console/ping"}, served)

	// The route of one Web is not reachable under the group of the other.
	recorder := httptest.NewRecorder()
	server.HTTPHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/demo.user.SecondServerTestWeb/console/status", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestServerRejectsHandlerWithoutEmbeddedDefaultServer(t *testing.T) {
	spec.ResetRegistryForTest()
	registerServerTestWeb()

	assert.PanicsWithError(t, "no embedded default web server type found on *server._ServerTestInvalidHandler", func() {
		NewServer(Option{
			HandlerTypes: []reflect.Type{reflect.TypeFor[*_ServerTestInvalidHandler]()},
			Executor:     &_ServerTestExecutor{},
		})
	})
}

func TestServerRecoveryPropagatesAbortHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ginCtx.Status(http.StatusOK)
			panic(http.ErrAbortHandler)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() { server.HTTPHandler().ServeHTTP(recorder, request) })

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestServerRecoveryReturnsInternalServerErrorForPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			panic("boom")
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
}

func TestServerRecoveryMapsStructuredErrorToHTTPStatus(t *testing.T) {
	testCases := []struct {
		name   string
		code   ex.Code
		status int
	}{
		{name: "application error", code: ex.NotFound, status: http.StatusNotFound},
		{name: "client system error", code: ex.InvalidRequest, status: http.StatusBadRequest},
		{name: "server system error", code: ex.ServiceUnavailable, status: http.StatusServiceUnavailable},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			server := newServerTestServer(&_ServerTestExecutor{
				execute: func(route spec.Route, ginCtx *gin.Context) {
					ex.PanicNew(testCase.code, "request failed")
				},
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
			server.HTTPHandler().ServeHTTP(recorder, request)

			if recorder.Code != testCase.status {
				t.Fatalf("unexpected status code: got %d want %d", recorder.Code, testCase.status)
			}
			if recorder.Body.Len() != 0 {
				t.Fatalf("structured recovery should not impose a response body: %q", recorder.Body.String())
			}
		})
	}
}

func TestServerRecoveryPreservesStartedWebResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ginCtx.String(http.StatusAccepted, "partial response")
			ex.PanicNew(ex.NotFound, "missing resource")
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "partial response" {
		t.Fatalf("unexpected response body: %q", recorder.Body.String())
	}
}

func TestServerRecoveryLogsSystemErrorRaiseStack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := setServerTestDefaultLogger(t)
	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ex.PanicNew(ex.InvalidRequest, "bad request")
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	logOutput, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	if !strings.Contains(string(logOutput), "web request system error recovered") {
		t.Fatalf("system error recovery was not logged: %s", logOutput)
	}
	if !strings.Contains(string(logOutput), "TestServerRecoveryLogsSystemErrorRaiseStack") {
		t.Fatalf("system error log does not contain the raise call site: %s", logOutput)
	}
}

func TestServerRequestCompletionLog(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		reportError bool
		level       string
	}{
		{"success", http.StatusOK, false, "DEBUG"},
		{"not modified", http.StatusNotModified, false, "DEBUG"},
		{"bad request", http.StatusBadRequest, false, "DEBUG"},
		{"unauthorized", http.StatusUnauthorized, false, "DEBUG"},
		{"forbidden", http.StatusForbidden, false, "DEBUG"},
		{"not found", http.StatusNotFound, false, "DEBUG"},
		{"internal error", http.StatusInternalServerError, false, "ERROR"},
		{"bad gateway", http.StatusBadGateway, false, "ERROR"},
		{"explicit error with success", http.StatusOK, true, "WARN"},
		{"explicit error with client error", http.StatusBadRequest, true, "WARN"},
		{"explicit error with server error", http.StatusBadGateway, true, "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			logPath := setServerTestDefaultLogger(t)
			server := newServerTestServer(&_ServerTestExecutor{
				execute: func(route spec.Route, ctx *gin.Context) {
					if tc.reportError {
						_ = ctx.Error(errors.New("handler failure"))
					}
					ctx.Status(tc.status)
				},
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
			server.HTTPHandler().ServeHTTP(recorder, request)
			record := readServerTestLastLogRecord(t, logPath)
			if record.Level != tc.level || record.Message != "web request finished" || record.Status != tc.status || recorder.Code != tc.status {
				t.Fatalf("unexpected completion log: %#v, response status=%d", record, recorder.Code)
			}
			if tc.reportError {
				if !strings.Contains(record.Errors, "handler failure") {
					t.Fatalf("missing error details: %#v", record)
				}
			} else if record.Errors != "" {
				t.Fatalf("unexpected error details: %#v", record)
			}
		})
	}
}

func TestServerRequestCompletionPreservesOrdinaryRequests(t *testing.T) {
	setServerTestDefaultLogger(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		request := httptest.NewRequestWithContext(ctx, "GET", "/demo.user.ServerTestWeb/ping", nil)
		server := newServerTestServer(&_ServerTestExecutor{execute: func(route spec.Route, ginCtx *gin.Context) {
			if ginCtx.Request != request || ginCtx.Request.Context() != ctx {
				t.Fatal("completion logging replaced the request or its context")
			}
			if ginCtx.Keys != nil {
				t.Fatal("ordinary request allocated tracking state")
			}
			ginCtx.Status(status)
		}})
		server.HTTPHandler().ServeHTTP(httptest.NewRecorder(), request)
	}
}

type _ServerTestLogRecord struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
	Status  int    `json:"status"`
	Errors  string `json:"errors"`
}

func setServerTestDefaultLogger(t *testing.T) string {
	t.Helper()

	logPath := filepath.Join(t.TempDir(), "server.jsonl")
	original := logger.New("vine:test")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelDebug,
		OutputPath: logPath,
	}))
	t.Cleanup(func() {
		logger.SetDefault(original)
	})
	return logPath
}

func readServerTestLastLogRecord(t *testing.T, path string) _ServerTestLogRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected at least one log line")
	}
	var record _ServerTestLogRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &record); err != nil {
		t.Fatalf("unmarshal log record: %v", err)
	}
	return record
}

func TestServerReverseProxyAbortReachesTransport(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	defer upstream.Close()
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	proxy.ErrorLog = log.New(io.Discard, "", 0)
	server := newServerTestServer(&_ServerTestExecutor{execute: func(route spec.Route, ctx *gin.Context) {
		proxy.ServeHTTP(ctx.Writer, ctx.Request)
	}})
	for _, protocol := range []string{"http1", "http2", "inproc"} {
		t.Run(protocol, func(t *testing.T) {
			var response *http.Response
			var err error
			if protocol == "inproc" {
				response, err = vinehttp.InprocRoundTrip(server.HTTPHandler(), httptest.NewRequest("GET", "/demo.user.ServerTestWeb/ping", nil))
			} else {
				listener := httptest.NewUnstartedServer(server.HTTPHandler())
				listener.EnableHTTP2 = protocol == "http2"
				listener.StartTLS()
				defer listener.Close()
				response, err = listener.Client().Get(listener.URL + "/demo.user.ServerTestWeb/ping")
			}
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err == nil || string(data) != "partial" {
				t.Fatalf("body=%q err=%v", data, err)
			}
		})
	}
}

func TestDevProxyCompletionLogDistinguishesUpstreamFromLocalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := setServerTestDefaultLogger(t)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("success") == "true" {
			_, _ = io.WriteString(w, "upstream success")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	t.Cleanup(backend.Close)
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	// The proxy follows the same development state file as a normal Vine Web.
	state := []byte(`{"host":"` + target.Hostname() + `","port":` + target.Port() + `}`)
	statePath := filepath.Join(t.TempDir(), "dev-state.json")
	if err := os.WriteFile(statePath, state, 0o600); err != nil {
		t.Fatal(err)
	}
	proxy := new(devproxy.Server)
	proxy.SetStateFile(statePath)
	var reportError bool
	var panicAfter bool
	server := newServerTestServer(&_ServerTestExecutor{execute: func(route spec.Route, ctx *gin.Context) {
		if ctx.Query("local") == "true" {
			if spec.IsUpstream5xx(ctx) {
				t.Fatal("upstream marker leaked into another request")
			}
			ctx.Status(http.StatusServiceUnavailable)
			return
		}
		proxy.GinCtx = ctx
		proxy.Serve()
		if ctx.Query("success") == "true" && ctx.Keys != nil {
			t.Fatal("successful proxy response allocated tracking state")
		}
		if panicAfter {
			panic("local handler panic")
		}
		if reportError {
			_ = ctx.Error(errors.New("local handler failure"))
		}
	}})
	for _, tc := range []struct {
		name          string
		query         string
		explicitError bool
		panicAfter    bool
		stopBackend   bool
		status        int
		level         string
	}{
		{"upstream 200", "success=true", false, false, false, 200, "DEBUG"},
		{"upstream 503", "", false, false, false, 503, "DEBUG"},
		{"local 503 after upstream request", "local=true", false, false, false, 503, "ERROR"},
		{"upstream with explicit error", "", true, false, false, 503, "ERROR"},
		{"local panic after upstream", "", false, true, false, 503, "ERROR"},
		{"connection failure", "", false, false, true, 502, "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reportError = tc.explicitError
			panicAfter = tc.panicAfter
			if tc.stopBackend {
				backend.Close()
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), "GET", "/demo.user.ServerTestWeb/ping", nil)
			request.URL.RawQuery = tc.query
			server.HTTPHandler().ServeHTTP(response, request)
			record := readServerTestLastLogRecord(t, logPath)
			if record.Level != tc.level || response.Code != tc.status {
				t.Fatalf("status=%d, log=%#v", response.Code, record)
			}
			if tc.explicitError && !strings.Contains(record.Errors, "local handler failure") {
				t.Fatalf("missing explicit error: %#v", record)
			}
		})
	}
}

func TestDevProxyFlushFailureThroughGin(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		level   string
	}{
		{"success", nil, "DEBUG"},
		{"flush failure", io.ErrClosedPipe, "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logPath := setServerTestDefaultLogger(t)
			headersFlushed := make(chan struct{})
			backend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = http.NewResponseController(w).Flush()
				// Delay the body until the proxy's header flush, exercising Gin's
				// pending status and the ReverseProxy flush timer together.
				select {
				case <-headersFlushed:
					_, _ = io.WriteString(w, "upstream unavailable")
				case <-time.After(5 * time.Second):
					t.Error("proxy did not flush upstream headers")
				}
			}))
			backend.Start()
			server := newServerTestDevProxy(t, backend.URL)
			writer := new(serverTestFlushWriter{ResponseRecorder: httptest.NewRecorder(), failure: tc.failure, headersFlushed: headersFlushed})
			server.HTTPHandler().ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), "GET", "/demo.user.ServerTestWeb/ping", nil))
			if writer.flushes.Load() == 0 || writer.Code != http.StatusServiceUnavailable {
				t.Fatalf("flushes=%d status=%d", writer.flushes.Load(), writer.Code)
			}
			record := readServerTestLastLogRecord(t, logPath)
			if record.Level != tc.level || record.Status != http.StatusServiceUnavailable {
				t.Fatalf("completion log=%#v", record)
			}
		})
	}
}

type serverTestFlushWriter struct {
	*httptest.ResponseRecorder
	failure        error
	flushes        atomic.Int32
	headersFlushed chan struct{}
}

func (w *serverTestFlushWriter) FlushError() error {
	w.ResponseRecorder.Flush()
	if w.flushes.Add(1) == 1 {
		close(w.headersFlushed)
	}
	return w.failure
}

// Like net/http's writer, legacy Flush discards the error from FlushError.
func (w *serverTestFlushWriter) Flush() { _ = w.FlushError() }

func TestDevProxyClientDisconnectThroughGin(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		name := "HTTP response"
		if upgrade {
			name = "WebSocket handshake"
		}
		t.Run(name, func(t *testing.T) {
			logPath := setServerTestDefaultLogger(t)
			entered := make(chan struct{})
			backend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				select {
				case <-r.Context().Done():
				case <-t.Context().Done():
				}
			}))
			backend.Start()
			server := newServerTestDevProxy(t, backend.URL)
			finished := make(chan any, 1)
			frontend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					recovered := recover()
					finished <- recovered
					if recovered != nil {
						panic(recovered)
					}
				}()
				server.HTTPHandler().ServeHTTP(w, r)
			}))
			frontend.Start()
			connection, err := net.DialTimeout("tcp", strings.TrimPrefix(frontend.URL, "http://"), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = connection.Close() })
			if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			headers := ""
			if upgrade {
				headers = "Connection: Upgrade\r\nUpgrade: websocket\r\n"
			}
			if _, err := io.WriteString(connection, "GET /demo.user.ServerTestWeb/ping HTTP/1.1\r\nHost: test\r\n"+headers+"\r\n"); err != nil {
				t.Fatal(err)
			}
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("backend was not reached")
			}
			if err := connection.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case recovered := <-finished:
				if recovered != http.ErrAbortHandler {
					t.Fatalf("canceled request did not abort: panic=%v", recovered)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("proxy did not exit after cancellation")
			}
			data, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var record _ServerTestLogRecord
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record.Level == "ERROR" || record.Level == "WARN" || record.Message == "web request finished" {
					t.Fatalf("canceled request produced a failure/completion log: %#v", record)
				}
			}
		})
	}
}

func newServerTestDevProxy(t *testing.T, backendURL string) *Server {
	t.Helper()
	target, err := url.Parse(backendURL)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(t.TempDir(), "dev-state.json")
	state := []byte(`{"host":"` + target.Hostname() + `","port":` + target.Port() + `}`)
	if err := os.WriteFile(statePath, state, 0o600); err != nil {
		t.Fatal(err)
	}
	return newServerTestServer(&_ServerTestExecutor{execute: func(route spec.Route, ctx *gin.Context) {
		proxy := new(devproxy.Server{GinCtx: ctx})
		proxy.SetStateFile(statePath)
		proxy.Serve()
	}})
}

func TestDevProxyWebSocketHMR(t *testing.T) {
	logPath := setServerTestDefaultLogger(t)
	const connected = `{"type":"connected"}`
	const ping = `{"type":"custom","event":"client:ping"}`
	const update = `{"type":"update","updates":[{"type":"js-update","path":"/src/app.ts"}]}`
	backendClosed := make(chan struct{})
	backend := httptest.NewTestServer(t, websocket.Server{
		Handshake: func(config *websocket.Config, request *http.Request) error {
			if request.URL.RequestURI() != "/ping?token=dev" || request.Header.Get("Sec-WebSocket-Protocol") != "vite-hmr" {
				t.Errorf("HMR handshake: uri=%s protocol=%s", request.URL.RequestURI(), request.Header.Get("Sec-WebSocket-Protocol"))
			}
			config.Protocol = []string{"vite-hmr"}
			return nil
		},
		Handler: func(connection *websocket.Conn) {
			defer close(backendClosed)
			if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Error(err)
				return
			}
			if err := websocket.Message.Send(connection, connected); err != nil {
				t.Error(err)
				return
			}
			var message string
			if err := websocket.Message.Receive(connection, &message); err != nil || message != ping {
				t.Errorf("HMR client message=%q error=%v", message, err)
				return
			}
			if err := websocket.Message.Send(connection, update); err != nil {
				t.Error(err)
				return
			}
			if err := websocket.Message.Receive(connection, &message); !errors.Is(err, io.EOF) {
				t.Errorf("HMR connection did not close cleanly: %v", err)
			}
		},
	})
	backend.Start()
	server := newServerTestDevProxy(t, backend.URL)
	proxyFinished := make(chan struct{})
	frontend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(proxyFinished)
		server.HTTPHandler().ServeHTTP(w, r)
	}))
	frontend.Start()
	config, err := websocket.NewConfig("ws"+strings.TrimPrefix(frontend.URL, "http")+"/demo.user.ServerTestWeb/ping?token=dev", frontend.URL)
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{"vite-hmr"}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	connection, err := config.DialContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if protocols := connection.Config().Protocol; len(protocols) != 1 || protocols[0] != "vite-hmr" {
		t.Fatalf("HMR negotiated protocols=%v", protocols)
	}
	var message string
	if err := websocket.Message.Receive(connection, &message); err != nil || message != connected {
		t.Fatalf("HMR connected message=%q error=%v", message, err)
	}
	if err := websocket.Message.Send(connection, ping); err != nil {
		t.Fatal(err)
	}
	if err := websocket.Message.Receive(connection, &message); err != nil || message != update {
		t.Fatalf("HMR update message=%q error=%v", message, err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-backendClosed:
	case <-time.After(5 * time.Second):
		t.Fatal("HMR backend connection remained open")
	}
	select {
	case <-proxyFinished:
	case <-time.After(5 * time.Second):
		t.Fatal("HMR proxy did not exit")
	}
	record := readServerTestLastLogRecord(t, logPath)
	if record.Level != "DEBUG" || record.Message != "web request finished" || record.Errors != "" {
		t.Fatalf("HMR completion log=%#v", record)
	}
}

func BenchmarkServerRequestCompletion(b *testing.B) {
	originalMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	b.Cleanup(func() { gin.SetMode(originalMode) })
	engine := gin.New()
	engine.Use(new(Server).ginLogger())
	engine.GET("/", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	request := httptest.NewRequestWithContext(b.Context(), "GET", "/", nil)
	writer := httptest.NewRecorder()
	b.ReportAllocs()
	for b.Loop() {
		engine.ServeHTTP(writer, request)
	}
}
