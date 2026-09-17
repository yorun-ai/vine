package server

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/web/assets"
	"go.yorun.ai/vine/internal/core/web/spec"
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

func TestServerRecoveryIgnoresAbortHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ginCtx.Status(http.StatusOK)
			panic(http.ErrAbortHandler)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

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

func TestServerLogsSuccessfulRequestAtDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := setServerTestDefaultLogger(t)
	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ginCtx.Status(http.StatusNotModified)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	record := readServerTestLastLogRecord(t, logPath)
	if record.Level != "DEBUG" || record.Message != "web request finished" {
		t.Fatalf("unexpected log record: %#v", record)
	}
}

func TestServerLogsBadRequestAtWarn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := setServerTestDefaultLogger(t)
	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.Route, ginCtx *gin.Context) {
			ginCtx.Status(http.StatusNotFound)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/demo.user.ServerTestWeb/ping", nil)
	server.HTTPHandler().ServeHTTP(recorder, request)

	record := readServerTestLastLogRecord(t, logPath)
	if record.Level != "WARN" || record.Message != "web request finished" {
		t.Fatalf("unexpected log record: %#v", record)
	}
}

type _ServerTestLogRecord struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
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
