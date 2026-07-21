package server

import (
	"context"
	"reflect"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type _DefaultExecutorDependency struct {
	Value string
}

type testDefaultExecutorDependency interface {
	ValueString() string
}

type _DefaultExecutorInterfaceDependency struct {
	value string
}

func (d *_DefaultExecutorInterfaceDependency) ValueString() string {
	return d.value
}

type testDefaultExecutorServiceServer interface {
	TraceSpan() string

	mustBeTestDefaultExecutorServiceServer()
}

type testDefaultExecutorNoResultServiceServer interface {
	mustBeTestDefaultExecutorNoResultServiceServer()
}

type testDefaultExecutorNoResultServiceServerER interface {
	Ping() error

	mustBeTestDefaultExecutorNoResultServiceServerER()
}

type _DefaultExecutorTestServiceServer struct{}

func (*_DefaultExecutorTestServiceServer) TraceSpan() string { return "" }

func (*_DefaultExecutorTestServiceServer) mustBeTestDefaultExecutorServiceServer() {}

type _DefaultExecutorNoResultTestServiceServer struct{}

func (*_DefaultExecutorNoResultTestServiceServer) Ping() {}

func (*_DefaultExecutorNoResultTestServiceServer) mustBeTestDefaultExecutorNoResultServiceServer() {
}

type testDefaultExecutorServiceServerER interface {
	TraceSpan() (string, ex.Error)

	mustBeTestDefaultExecutorServiceServerER()
}

type _DefaultExecutorTestServiceServerER struct{}

func (*_DefaultExecutorTestServiceServerER) TraceSpan() (string, ex.Error) { return "", nil }

func (*_DefaultExecutorTestServiceServerER) mustBeTestDefaultExecutorServiceServerER() {}

type _DefaultExecutorNoResultTestServiceServerER struct{}

func (*_DefaultExecutorNoResultTestServiceServerER) Ping() error { return nil }

func (*_DefaultExecutorNoResultTestServiceServerER) mustBeTestDefaultExecutorNoResultServiceServerER() {
}

type _DefaultExecutorContextServiceImpl struct {
	_DefaultExecutorTestServiceServer
	Ctx spec.Context
}

func (s *_DefaultExecutorContextServiceImpl) TraceSpan() string {
	return s.Ctx.Trace().Span()
}

type _DefaultExecutorNoContextFieldService struct {
	_DefaultExecutorTestServiceServer
}

func (s *_DefaultExecutorNoContextFieldService) TraceSpan() string {
	return "ok"
}

type _DefaultExecutorInstanceServiceImpl struct {
	_DefaultExecutorTestServiceServer
	Dep *_DefaultExecutorDependency
}

func (s *_DefaultExecutorInstanceServiceImpl) TraceSpan() string {
	return s.Dep.Value
}

type _DefaultExecutorInterfaceServiceImpl struct {
	_DefaultExecutorTestServiceServer
	Dep testDefaultExecutorDependency
}

func (s *_DefaultExecutorInterfaceServiceImpl) TraceSpan() string {
	return s.Dep.ValueString()
}

type _DefaultExecutorERContextServiceImpl struct {
	_DefaultExecutorTestServiceServerER
	Ctx spec.Context
}

func (s *_DefaultExecutorERContextServiceImpl) TraceSpan() (string, ex.Error) {
	return s.Ctx.Trace().Span(), nil
}

type _DefaultExecutorMultiContextService struct {
	_DefaultExecutorTestServiceServer
	Primary   spec.Context
	Secondary spec.Context
}

type _DefaultExecutorPrivateContextService struct {
	_DefaultExecutorTestServiceServer
	ctx spec.Context
}

type _DefaultExecutorNoResultServiceImpl struct {
	_DefaultExecutorNoResultTestServiceServer
}

func (*_DefaultExecutorNoResultServiceImpl) Ping() {}

type _DefaultExecutorErrorServiceImpl struct {
	_DefaultExecutorTestServiceServerER
}

func (*_DefaultExecutorErrorServiceImpl) TraceSpan() (string, ex.Error) {
	return "", ex.New(ex.OperationFailed, "default-executor-error")
}

var _DefaultExecutorTraceServiceSpec = &spec.ServiceSpec{
	Type:     spec.ServiceSpecTypeServer,
	Name:     "DefaultExecutorTraceService",
	SkelName: "server.defaultExecutor.trace",

	ServerType:        reflect.TypeOf((*testDefaultExecutorServiceServer)(nil)).Elem(),
	DefaultServerType: reflect.TypeOf(&_DefaultExecutorTestServiceServer{}),

	ERServerType:        reflect.TypeOf((*testDefaultExecutorServiceServerER)(nil)).Elem(),
	DefaultERServerType: reflect.TypeOf(&_DefaultExecutorTestServiceServerER{}),

	Methods: []*spec.MethodSpec{{
		Name:       "TraceSpan",
		SkelName:   "traceSpan",
		ResultType: reflect.TypeOf(""),
	}},
}

var _DefaultExecutorNoResultServiceSpec = &spec.ServiceSpec{
	Type:     spec.ServiceSpecTypeServer,
	Name:     "DefaultExecutorNoResultService",
	SkelName: "server.defaultExecutor.noResult",

	ServerType:        reflect.TypeOf((*testDefaultExecutorNoResultServiceServer)(nil)).Elem(),
	DefaultServerType: reflect.TypeOf(&_DefaultExecutorNoResultTestServiceServer{}),

	ERServerType:        reflect.TypeOf((*testDefaultExecutorNoResultServiceServerER)(nil)).Elem(),
	DefaultERServerType: reflect.TypeOf(&_DefaultExecutorNoResultTestServiceServerER{}),

	Methods: []*spec.MethodSpec{{
		Name:     "Ping",
		SkelName: "ping",
	}},
}

func init() {
	spec.Register(_DefaultExecutorTraceServiceSpec)
	spec.Register(_DefaultExecutorNoResultServiceSpec)
}

func TestDefaultExecutorInjectsSpecContextField(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorContextServiceImpl]()},
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if !meta.IsValidSpan(rpcResponse.Result().(string)) {
		t.Fatalf("expected result to be execution span, got %v", rpcResponse.Result())
	}
}

func TestDefaultExecutorInjectsSpecContextFieldForERService(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorERContextServiceImpl]()},
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if !meta.IsValidSpan(rpcResponse.Result().(string)) {
		t.Fatalf("expected result to be execution span, got %v", rpcResponse.Result())
	}
}

func TestDefaultExecutorReturnsNilForMethodWithoutResult(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorNoResultServiceImpl]()},
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorNoResultServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if rpcResponse.Result() != nil {
		t.Fatalf("expected nil result, got %#v", rpcResponse.Result())
	}
}

func TestDefaultExecutorReturnsERMethodError(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorErrorServiceImpl]()},
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OperationFailed {
		t.Fatalf("expected operation failed error, got %#v", rpcResponse.Error())
	}
}

func TestDefaultExecutorInjectsInstanceByType(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorInstanceServiceImpl]()},
		Executor:     NewDefaultExecutor(With(&_DefaultExecutorDependency{Value: "dep"})),
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if rpcResponse.Result() != "dep" {
		t.Fatalf("expected injected instance result, got %v", rpcResponse.Result())
	}
}

func TestDefaultExecutorInjectsInstanceByAsType(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_DefaultExecutorInterfaceServiceImpl]()},
		Executor: NewDefaultExecutor(
			WithAs(reflect.TypeOf((*testDefaultExecutorDependency)(nil)).Elem(), &_DefaultExecutorInterfaceDependency{value: "iface"}),
		),
	})

	rpcResponse := srv.handle(newDefaultExecutorRequest(srv, _DefaultExecutorTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if rpcResponse.Result() != "iface" {
		t.Fatalf("expected injected interface result, got %v", rpcResponse.Result())
	}
}

func TestInjectContextSkipsWhenNoSpecContextFieldExists(t *testing.T) {
	implValue := reflect.New(reflect.TypeOf(_DefaultExecutorNoContextFieldService{}))

	executor := newInitializedDefaultExecutor(t, reflect.TypeOf(&_DefaultExecutorNoContextFieldService{}))
	executor.inject(implValue, newDefaultExecutorRPCContext())
}

func TestDefaultExecutorInitPanicsWhenMultipleSpecContextFieldsExist(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	newInitializedDefaultExecutor(t, reflect.TypeOf(&_DefaultExecutorMultiContextService{}))
}

func TestDefaultExecutorInitPanicsWhenSpecContextFieldCannotBeSet(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	newInitializedDefaultExecutor(t, reflect.TypeOf(&_DefaultExecutorPrivateContextService{}))
}

func newDefaultExecutorRequest(srv *Server, method spec.MethodInfo) spec.Request {
	methodImpl, err := srv.implDict.GetMethodImpl(method.Service().SkelName(), method.SkelName())
	if err != nil {
		panic(err)
	}
	return &spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: method,
		MethodImplValue: methodImpl,
	}
}

func newDefaultExecutorRPCContext() spec.Context {
	return &spec.ContextImpl{
		ContextImpl: meta.ContextImpl{
			Context:    context.Background(),
			TraceValue: meta.InitialTrace(),
		},
	}
}

func newInitializedDefaultExecutor(t *testing.T, handlerType reflect.Type) *_DefaultExecutor {
	t.Helper()
	infoDict := spec.NewImplDict()
	infoDict.Add(handlerType)
	executor := &_DefaultExecutor{}
	executor.Init(*infoDict)
	return executor
}
