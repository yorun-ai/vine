package server

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"go.yorun.ai/vine/internal/utilfortest/spectest"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	httptrans "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

type testServerServiceServer interface {
	mustBeServerServiceServer()
}

type defaultTestServerServiceServer struct{}

func (*defaultTestServerServiceServer) Ping() {}

func (*defaultTestServerServiceServer) mustBeServerServiceServer() {}

func testServerApp() meta.App {
	return meta.MustNewApp("test.app", "1.0.0", "123e4567-e89b-12d3-a456-426614174012")
}

func TestNewRequiresApp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing App to panic")
		}
	}()
	New(Option{})
}

type testServerServiceServerER interface {
	mustBeServerServiceServerER()
}

type defaultTestServerServiceServerER struct{}

func (*defaultTestServerServiceServerER) Ping() error { return nil }

func (*defaultTestServerServiceServerER) mustBeServerServiceServerER() {}

type serverTestServiceImpl struct {
	defaultTestServerServiceServer
}

type serverCloneArguments struct {
	Names []string `json:"names" skel:"index(0)"`
}

type serverCloneServiceServer interface {
	Mutate([]string)

	mustBeServerCloneServiceServer()
}

type defaultServerCloneServiceServer struct{}

func (*defaultServerCloneServiceServer) Mutate([]string) {}

func (*defaultServerCloneServiceServer) mustBeServerCloneServiceServer() {}

type serverCloneServiceServerER interface {
	Mutate([]string) error

	mustBeServerCloneServiceServerER()
}

type defaultServerCloneServiceServerER struct{}

func (*defaultServerCloneServiceServerER) Mutate([]string) error { return nil }

func (*defaultServerCloneServiceServerER) mustBeServerCloneServiceServerER() {}

type serverCloneServiceImpl struct {
	defaultServerCloneServiceServer
}

func (*serverCloneServiceImpl) Mutate(names []string) {
	names[0] = "changed"
}

type _ServerTestExecutor struct {
	beforeExecute func()
	result        any
	err           ex.Error
	panicV        any
	profileLabels map[string]string
}

var (
	serverTestRegisterOnce sync.Once
	serverTestServiceInfo  = &spec.ServiceSpec{
		Type:                spec.ServiceSpecTypeServer,
		Name:                "TestService",
		SkelName:            "test.service.server",
		ServerType:          reflect.TypeFor[testServerServiceServer](),
		DefaultServerType:   reflect.TypeFor[*defaultTestServerServiceServer](),
		ERServerType:        reflect.TypeFor[testServerServiceServerER](),
		DefaultERServerType: reflect.TypeFor[*defaultTestServerServiceServerER](),
		Methods: []*spec.MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}
	serverCloneRegisterOnce sync.Once
	serverCloneServiceInfo  = &spec.ServiceSpec{
		Type:                spec.ServiceSpecTypeServer,
		Name:                "CloneService",
		SkelName:            "test.service.server.clone",
		ServerType:          reflect.TypeFor[serverCloneServiceServer](),
		DefaultServerType:   reflect.TypeFor[*defaultServerCloneServiceServer](),
		ERServerType:        reflect.TypeFor[serverCloneServiceServerER](),
		DefaultERServerType: reflect.TypeFor[*defaultServerCloneServiceServerER](),
		Methods: []*spec.MethodSpec{{
			Name:          "Mutate",
			SkelName:      "mutate",
			ArgumentsType: reflect.TypeFor[serverCloneArguments](),
		}},
	}
)

func (*_ServerTestExecutor) Init(spec.ImplDict) {}

func (e *_ServerTestExecutor) Execute(ctx spec.Context, _ spec.MethodImpl, _ []any) (any, ex.Error) {
	if e.beforeExecute != nil {
		e.beforeExecute()
	}
	if e.panicV != nil {
		panic(e.panicV)
	}
	if e.profileLabels != nil {
		for _, key := range []string{"vine.app", "vine.protocol", "vine.service", "vine.method"} {
			if value, ok := pprof.Label(ctx, key); ok {
				e.profileLabels[key] = value
			}
		}
	}
	return e.result, e.err
}

func ensureServerTestServiceRegistered() {
	serverTestRegisterOnce.Do(func() {
		spec.Register(serverTestServiceInfo)
	})
}

func ensureServerCloneServiceRegistered() {
	serverCloneRegisterOnce.Do(func() {
		spec.Register(serverCloneServiceInfo)
	})
}

func TestServerHandleWithoutArgumentsReturnsNoErrorResponse(t *testing.T) {
	ensureServerTestServiceRegistered()

	trace := meta.InitialTrace()
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	handlerDict := spec.NewImplDict()
	handlerDict.Add(reflect.TypeFor[*serverTestServiceImpl]())
	serverTestMethodInfo := serverTestServiceInfo.Methods[0].Info()
	methodImpl, err := handlerDict.GetMethodImpl(serverTestServiceInfo.SkelName, serverTestMethodInfo.SkelName())
	if err != nil {
		t.Fatalf("GetMethodImpl() error = %v", err)
	}

	server := New(Option{
		App:          client,
		HandlerTypes: []reflect.Type{reflect.TypeFor[*serverTestServiceImpl]()},
	})
	response := server.handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      trace,
		ClientValue:     client,
		MethodInfoValue: serverTestMethodInfo,
		MethodImplValue: methodImpl,
	})

	if response == nil {
		t.Fatalf("expected response to be returned")
	}
	if response.Method() != serverTestMethodInfo {
		t.Fatalf("expected method to be preserved")
	}
	if response.Result() != nil {
		t.Fatalf("expected nil result, got %#v", response.Result())
	}
	if response.Error() == nil || response.Error().Type() != ex.NoError {
		t.Fatalf("expected no error response, got %#v", response.Error())
	}
}

func TestServerHandleAddsProfileLabels(t *testing.T) {
	method := spectest.RpcService(&spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeBoth,
		Name:     "ProfileService",
		SkelName: "test.service.profile",
		Methods:  []*spec.MethodSpec{{Name: "Ping", SkelName: "ping"}},
	}).Methods()[0]
	executor := &_ServerTestExecutor{profileLabels: map[string]string{}}
	server := &Server{
		opt:      &Option{App: testServerApp()},
		executor: executor,
	}

	server.handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: method,
	})

	assert.Equal(t, map[string]string{
		"vine.app":      "test.app",
		"vine.protocol": "rpc",
		"vine.service":  "test.service.profile",
		"vine.method":   "ping",
	}, executor.profileLabels)
}

func TestServerRpcHandlerClonesArguments(t *testing.T) {
	ensureServerCloneServiceRegistered()

	serverApp, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	server := New(Option{
		App:          serverApp,
		HandlerTypes: []reflect.Type{reflect.TypeFor[*serverCloneServiceImpl]()},
	})
	arguments := &serverCloneArguments{Names: []string{"vine"}}

	response := server.RpcHandler().ServeRpc(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		ClientValue:     client,
		MethodInfoValue: serverCloneServiceInfo.Methods[0].Info(),
		ArgumentsValue:  arguments,
	})

	if response.Error() == nil || response.Error().Type() != ex.NoError {
		t.Fatalf("expected no error response, got %#v", response.Error())
	}
	if arguments.Names[0] != "vine" {
		t.Fatalf("expected original arguments to stay unchanged, got %#v", arguments.Names)
	}
}

func TestServerHandlePreservesExecutorError(t *testing.T) {
	method := spectest.RpcService(&spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeBoth,
		Name:     "TestService",
		SkelName: "test.service.handle.error",
		Methods:  []*spec.MethodSpec{{Name: "Ping", SkelName: "ping"}},
	}).Methods()[0]
	wantErr := ex.New(ex.OperationFailed, "boom")
	response := (&Server{
		opt: &Option{App: testServerApp()},
		executor: &_ServerTestExecutor{
			err: wantErr,
		},
	}).handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: method,
	})

	if response.Error() != wantErr {
		t.Fatalf("expected executor error to be preserved, got %#v", response.Error())
	}
}

func TestServerHTTPHandlerReturnsStandardVrpcErrorForInvalidRequest(t *testing.T) {
	ensureServerTestServiceRegistered()

	serverApp, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	logPath := filepath.Join(t.TempDir(), "rpc-rejected.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: logPath})
	server := New(Option{
		App:          serverApp,
		Logger:       log,
		HandlerTypes: []reflect.Type{reflect.TypeFor[*serverTestServiceImpl]()},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/invalid-path", bytes.NewBufferString("{"))
	header := req.Header
	trace := meta.InitialTrace()
	httptrans.EncodeTraceToHeader(header, trace)
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	httptrans.EncodeClientToHeader(header, client)
	header.Set(httptrans.HeaderAccept, "application/vrpc+json")
	header.Set(httptrans.HeaderContentType, "application/vrpc+json")

	recorder := httptest.NewRecorder()
	server.HTTPHandler().ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != httptrans.ResponseStatusCode {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if resp.Header.Get(httptrans.HeaderContentType) != "application/vrpc+json" {
		t.Fatalf("unexpected content-type: %s", resp.Header.Get(httptrans.HeaderContentType))
	}
	if resp.Header.Get(httptrans.HeaderRpcStatus) != string(ex.InvalidRequest) {
		t.Fatalf("unexpected vrpc-status: %s", resp.Header.Get(httptrans.HeaderRpcStatus))
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Contains(bodyBytes, []byte(`"code":"INVALID_REQUEST"`)) {
		t.Fatalf("unexpected response body: %s", string(bodyBytes))
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read rejection log: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logBytes), &record); err != nil {
		t.Fatalf("decode rejection log: %v", err)
	}
	if record["msg"] != "rpc server request rejected" || record["level"] != "ERROR" || record["code"] != "INVALID_REQUEST" {
		t.Fatalf("unexpected rejection record: %#v", record)
	}
	if record["vrpcId"] != trace.Id() || record["clientName"] != client.Name() {
		t.Fatalf("validated rejection metadata was not preserved: %#v", record)
	}
	if _, exists := record["body"]; exists {
		t.Fatalf("raw request body leaked into rejection log: %#v", record)
	}
	if _, exists := record["rawBody"]; exists {
		t.Fatalf("raw request body leaked into rejection log: %#v", record)
	}
}

func TestServerHandleConvertsRecoveredPanicToInternalError(t *testing.T) {
	method := spectest.RpcService(&spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeBoth,
		Name:     "TestService",
		SkelName: "test.service.handle.panic",
		Methods:  []*spec.MethodSpec{{Name: "Ping", SkelName: "ping"}},
	}).Methods()[0]
	app, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	response := (&Server{
		opt: &Option{App: app},
		executor: &_ServerTestExecutor{
			panicV: "boom",
		},
	}).handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		ClientValue:     client,
		MethodInfoValue: method,
	})

	if response.Error() == nil || response.Error().Code() != ex.Internal {
		t.Fatalf("expected internal error, got %#v", response.Error())
	}
}

func TestServerHandleDoesNotTreatPanickedOKErrorAsSuccess(t *testing.T) {
	method := spectest.RpcService(new(spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeBoth,
		Name:     "TestService",
		SkelName: "test.service.handle.ok-panic",
		Methods:  []*spec.MethodSpec{{Name: "Ping", SkelName: "ping"}},
	})).Methods()[0]
	response := (&Server{
		opt: &Option{},
		executor: new(_ServerTestExecutor{
			panicV: ex.NewOK(),
		}),
	}).handle(new(spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: method,
	}))

	if response.Error() == nil || response.Error().Code() != ex.Internal {
		t.Fatalf("panicked OK error must be converted to failure, got %#v", response.Error())
	}
}

func TestServerHandleClassifiesOnlyRequestCancellation(t *testing.T) {
	method := spectest.RpcService(new(spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeBoth,
		Name:     "CancellationService",
		SkelName: "test.service.handle.cancellation",
		Methods:  []*spec.MethodSpec{{Name: "Get", SkelName: "get"}},
	})).Methods()[0]
	for _, tc := range []struct {
		name       string
		cancel     bool
		panicValue any
		returned   ex.Error
		code       ex.Code
		level      string
		panicked   bool
	}{
		{name: "database cancellation", cancel: true, panicValue: context.Canceled, code: ex.InvocationCancelled, level: "DEBUG"},
		{name: "wrapped cancellation", cancel: true, panicValue: fmt.Errorf("query: %w", context.Canceled), code: ex.InvocationCancelled, level: "DEBUG"},
		{name: "nested rpc cancellation", cancel: true, panicValue: ex.New(ex.InvocationCancelled, "nested request cancelled"), code: ex.InvocationCancelled, level: "DEBUG"},
		{name: "returned nested cancellation", cancel: true, returned: ex.RecoverExecution(ex.New(ex.InvocationCancelled, "nested request cancelled")), code: ex.InvocationCancelled, level: "DEBUG"},
		{name: "live request cancellation panic", panicValue: context.Canceled, code: ex.Internal, level: "ERROR", panicked: true},
		{name: "live request nested cancellation", panicValue: ex.New(ex.InvocationCancelled, "nested request cancelled"), code: ex.InvocationCancelled, level: "ERROR", panicked: true},
		{name: "unrelated panic after cancellation", cancel: true, panicValue: "database unavailable", code: ex.Internal, level: "ERROR", panicked: true},
		{name: "business error with cancelled cause", cancel: true, panicValue: ex.New(ex.OperationFailed, "business failure", ex.WithCause(context.Canceled)), code: ex.OperationFailed, level: "ERROR", panicked: true},
		{name: "timeout", panicValue: context.DeadlineExceeded, code: ex.Internal, level: "ERROR", panicked: true},
		{name: "completed before cancellation", cancel: true, code: ex.OK, level: "DEBUG"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			executor := new(_ServerTestExecutor{
				panicV: tc.panicValue,
				err:    tc.returned,
				result: "retained result",
			})
			if tc.cancel {
				executor.beforeExecute = cancel
			}
			path := filepath.Join(t.TempDir(), "cancellation.jsonl")
			log := logger.New("vine:test", logger.WithOption{
				Format:     logger.FormatJSON,
				Level:      logger.LevelDebug,
				OutputPath: path,
			})
			server := new(Server{
				opt:      new(Option{App: testServerApp()}),
				executor: executor,
				log:      log,
			})
			response := server.handle(new(spec.RequestImpl{
				ContextValue:    ctx,
				TraceValue:      meta.InitialTrace(),
				MethodInfoValue: method,
			}))
			wantCode := tc.code
			if wantCode.IsUnresponsive() {
				wantCode = ex.Internal
			}
			if response.Error().Code() != wantCode {
				t.Fatalf("unexpected response code: %v", response.Error())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
			var finished map[string]any
			if err := json.Unmarshal(lines[len(lines)-1], &finished); err != nil {
				t.Fatal(err)
			}
			if finished["msg"] != "rpc server handle finished" || finished["code"] != string(tc.code) || finished["level"] != tc.level {
				t.Fatalf("unexpected terminal record: %#v", finished)
			}
			_, hasPanic := finished["panic"]
			_, hasStack := finished["stack"]
			if hasPanic != tc.panicked || hasStack != tc.panicked {
				t.Fatalf("incorrect cancellation/panic diagnostics: %#v", finished)
			}
			if tc.code == ex.OK && response.Result() != "retained result" {
				t.Fatal("request cancellation must not replace completed results")
			}
		})
	}
}
