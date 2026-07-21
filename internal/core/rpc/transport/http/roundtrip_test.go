package http

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type testServiceServer interface {
	Ping(name string) string
}

type defaultTestServiceServer struct{}

func (*defaultTestServiceServer) Ping(string) string { return "" }

type testServiceServerER interface {
	Ping(name string) (string, ex.Error)
}

type defaultTestServiceServerER struct{}

func (*defaultTestServiceServerER) Ping(string) (string, ex.Error) { return "", nil }

type pingArguments struct {
	Name string `arg:"0"`
}

var (
	testServiceInfoOnce         sync.Once
	testServiceInfoInst         spec.ServiceInfo
	testHandlerDictOnce         sync.Once
	testHandlerDictInst         *spec.ImplDict
	standaloneMethodInfoCounter uint64
)

type testServiceImpl struct {
	defaultTestServiceServer
}

func testMethodInfo() *spec.MethodSpec {
	return &spec.MethodSpec{
		Name:          "Ping",
		SkelName:      "ping",
		ArgumentsType: reflect.TypeOf(pingArguments{}),
		ResultType:    reflect.TypeOf(""),
	}
}

func newStandaloneMethodInfo(argumentsType reflect.Type, resultType reflect.Type, argumentsContainsBinaryType bool, resultContainsBinaryType bool) spec.MethodInfo {
	serviceInfo := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "StandaloneService",
		SkelName: fmt.Sprintf("test.standalone.%d", atomic.AddUint64(&standaloneMethodInfoCounter, 1)),
		Methods: []*spec.MethodSpec{{
			Name:                        "Ping",
			SkelName:                    "ping",
			ArgumentsType:               argumentsType,
			ResultType:                  resultType,
			ArgumentsContainsBinaryType: argumentsContainsBinaryType,
			ResultContainsBinaryType:    resultContainsBinaryType,
		}},
	})
	return serviceInfo.Methods()[0]
}

func testServiceInfo() spec.ServiceInfo {
	testServiceInfoOnce.Do(func() {
		method := testMethodInfo()
		si := &spec.ServiceSpec{
			Type:                spec.ServiceSpecTypeServer,
			Name:                "TestService",
			SkelName:            "test.TestService",
			ServerType:          reflect.TypeOf((*testServiceServer)(nil)).Elem(),
			DefaultServerType:   reflect.TypeOf(&defaultTestServiceServer{}),
			ERServerType:        reflect.TypeOf((*testServiceServerER)(nil)).Elem(),
			DefaultERServerType: reflect.TypeOf(&defaultTestServiceServerER{}),
			Methods:             []*spec.MethodSpec{method},
		}
		spec.Register(si)
		testServiceInfoInst = si.Methods[0].Info().Service()
	})
	return testServiceInfoInst
}

func testHandlerDict() *spec.ImplDict {
	testHandlerDictOnce.Do(func() {
		testServiceInfo()
		testHandlerDictInst = spec.NewImplDict()
		testHandlerDictInst.Add(reflect.TypeOf(&testServiceImpl{}))
	})
	return testHandlerDictInst
}

func testContext() spec.Context {
	trace := meta.InitialTrace()
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		panic(err)
	}
	actor := meta.NewAnonymousActor()
	ctx := &spec.ContextImpl{
		ContextImpl: meta.ContextImpl{
			Context:    context.Background(),
			TraceValue: trace,
			ActorValue: actor,
		},
		ClientValue: client,
	}
	return ctx
}

func testServerApp() meta.App {
	app, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		panic(err)
	}
	return app
}

func TestEncodeRequestDecodeRequestRoundTrip(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]
	rpcCtx := testContext()
	msg := &spec.RequestImpl{
		ContextValue:    rpcCtx,
		TraceValue:      rpcCtx.Trace(),
		ActorValue:      rpcCtx.Actor(),
		InitiatorValue:  rpcCtx.Initiator(),
		ClientValue:     rpcCtx.Client(),
		MethodInfoValue: method,
		ArgumentsValue:  &pingArguments{Name: "vine"},
	}

	req, err := encodeRequest("http://localhost:8080", msg)
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}

	got, err := DecodeRequest(req)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	gotCtx := got.Context()
	if gotCtx == nil {
		t.Fatalf("decoded request context is nil")
	}

	if got.MethodInfo() == nil {
		t.Fatalf("decoded method is nil")
	}
	if got.MethodInfo().SkelName() != method.SkelName() {
		t.Fatalf("unexpected method skel name: got %s want %s", got.MethodInfo().SkelName(), method.SkelName())
	}
	if got.MethodInfo().Name() != method.Name() {
		t.Fatalf("unexpected method name: got %s want %s", got.MethodInfo().Name(), method.Name())
	}
	gotRPCCtx := &spec.ContextImpl{
		ContextImpl: meta.ContextImpl{
			Context:        gotCtx,
			TraceValue:     got.Trace(),
			InitiatorValue: got.Initiator(),
			ActorValue:     got.Actor(),
		},
		ClientValue: got.Client(),
	}
	if gotRPCCtx.Trace().Id() != rpcCtx.Trace().Id() {
		t.Fatalf("unexpected id: got %s want %s", gotRPCCtx.Trace().Id(), rpcCtx.Trace().Id())
	}
	if gotRPCCtx.Trace().Span() != rpcCtx.Trace().Span() {
		t.Fatalf("unexpected span: got %s want %s", gotRPCCtx.Trace().Span(), rpcCtx.Trace().Span())
	}
	args, ok := got.Arguments().(*pingArguments)
	if !ok || args.Name != "vine" {
		t.Fatalf("unexpected arguments: %+v", got.Arguments())
	}
}

func TestWriteResponseDecodeResponseRoundTrip(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]
	reqCtx := testContext()
	reqMsg := &spec.RequestImpl{
		ContextValue:    reqCtx,
		TraceValue:      reqCtx.Trace(),
		ActorValue:      reqCtx.Actor(),
		InitiatorValue:  reqCtx.Initiator(),
		ClientValue:     reqCtx.Client(),
		MethodInfoValue: method,
		ArgumentsValue:  &pingArguments{Name: "vine"},
	}
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: reqMsg.MethodInfo(),
		ResultValue: "pong",
		ErrorValue:  ex.NewOK(),
	}
	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, nil, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}
	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()

	got, err := decodeResponse(httpResp, reqMsg.MethodInfo())
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}

	if got.Server().Name() != msg.Server().Name() {
		t.Fatalf("unexpected server: got %s want %s", got.Server().Name(), msg.Server().Name())
	}
	if got.Server().Version() != msg.Server().Version() {
		t.Fatalf("unexpected server version: got %s want %s", got.Server().Version(), msg.Server().Version())
	}
	if got.Result() != "pong" {
		t.Fatalf("unexpected result: got %#v", got.Result())
	}
	if got := httpResp.Header.Get(HeaderRpcTrace); got != "" {
		t.Fatalf("unexpected response trace header: %s", got)
	}
}

func TestDecodeRequestRejectsDuplicatedHeader(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]
	ctx := testContext()
	msg := &spec.RequestImpl{
		ContextValue:    ctx,
		TraceValue:      ctx.Trace(),
		ActorValue:      ctx.Actor(),
		InitiatorValue:  ctx.Initiator(),
		ClientValue:     ctx.Client(),
		MethodInfoValue: method,
		ArgumentsValue:  &pingArguments{Name: "vine"},
	}
	req, err := encodeRequest("http://localhost:8080", msg)
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}
	req.Header.Add(HeaderRpcTrace, "duplicated")

	_, err = DecodeRequest(req)
	if err == nil {
		t.Fatalf("expected duplicated header error")
	}
}
