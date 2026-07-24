package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

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
	Names []string `json:"names" arg:"0"`
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
	result any
	err    ex.Error
	panicV any
}

type serverNestedInvokeInnerExecutor struct {
	executionTrace meta.Trace
}

func (*serverNestedInvokeInnerExecutor) Init(spec.ImplDict) {}

func (e *serverNestedInvokeInnerExecutor) Execute(rpcContext spec.Context, methodImpl spec.MethodImpl, arguments []any) (any, ex.Error) {
	e.executionTrace = rpcContext.Trace()
	return nil, nil
}

var (
	serverTestRegisterOnce sync.Once
	serverTestServiceInfo  = &spec.ServiceSpec{
		Type:                spec.ServiceSpecTypeServer,
		Name:                "TestService",
		SkelName:            "test.service.server",
		ServerType:          reflect.TypeOf((*testServerServiceServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&defaultTestServerServiceServer{}),
		ERServerType:        reflect.TypeOf((*testServerServiceServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&defaultTestServerServiceServerER{}),
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
		ServerType:          reflect.TypeOf((*serverCloneServiceServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&defaultServerCloneServiceServer{}),
		ERServerType:        reflect.TypeOf((*serverCloneServiceServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&defaultServerCloneServiceServerER{}),
		Methods: []*spec.MethodSpec{{
			Name:          "Mutate",
			SkelName:      "mutate",
			ArgumentsType: reflect.TypeOf(serverCloneArguments{}),
		}},
	}
)

func (*_ServerTestExecutor) Init(spec.ImplDict) {}

func (e *_ServerTestExecutor) Execute(spec.Context, spec.MethodImpl, []any) (any, ex.Error) {
	if e.panicV != nil {
		panic(e.panicV)
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
	handlerDict.Add(reflect.TypeOf(&serverTestServiceImpl{}))
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
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "TestService",
		SkelName: "test.service.handle.error",
		Methods:  []*spec.MethodSpec{{Name: "Ping", SkelName: "ping"}},
	}).Methods()[0]
	wantErr := ex.New(ex.OperationFailed, "boom")
	response := (&Server{
		opt: &Option{},
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
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: logPath})
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
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
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
	method := spec.ConvertSpecToInfoForTest(new(spec.ServiceSpec{
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
