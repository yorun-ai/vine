package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/logger"
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

func registerServerTestWeb() {
	spec.Register(&spec.WebSpec{
		Name:              "ServerTestWeb",
		SkelName:          "demo.user.ServerTestWeb",
		ServerType:        reflect.TypeFor[_ServerTestWebServer](),
		DefaultServerType: reflect.TypeFor[*_DefaultServerTestWebServer](),
	})
}

type _ServerTestInvalidHandler struct{}

func (*_ServerTestInvalidHandler) Routes(*spec.Router) {}

func (*_ServerTestInvalidHandler) mustBeServerTestWebServer() {}

type _ServerTestExecutor struct {
	initRoutes []spec.RouteInfo
	executed   []spec.RouteInfo
	contexts   []*gin.Context
	execute    func(route spec.RouteInfo, ginCtx *gin.Context)
}

func (e *_ServerTestExecutor) Init(routes []spec.RouteInfo) {
	e.initRoutes = append([]spec.RouteInfo{}, routes...)
}

func (e *_ServerTestExecutor) Execute(route spec.RouteInfo, ginCtx *gin.Context) {
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
		execute: func(route spec.RouteInfo, ginCtx *gin.Context) {
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
		execute: func(route spec.RouteInfo, ginCtx *gin.Context) {
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

func TestServerLogsSuccessfulRequestAtDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := setServerTestDefaultLogger(t)
	server := newServerTestServer(&_ServerTestExecutor{
		execute: func(route spec.RouteInfo, ginCtx *gin.Context) {
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
		execute: func(route spec.RouteInfo, ginCtx *gin.Context) {
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
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
		Mode:       logger.ModeJSON,
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
