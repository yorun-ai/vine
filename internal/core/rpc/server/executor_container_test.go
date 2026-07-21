package server

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type testCTRServiceServer interface {
	TraceSpan() string

	mustBeTestCTRServiceServer()
}

type testCTRNoResultServiceServer interface {
	Ping()

	mustBeTestCTRNoResultServiceServer()
}

type testCTRNoResultServiceServerER interface {
	Ping() error

	mustBeTestCTRNoResultServiceServerER()
}

type _DefaultTestCTRServiceServer struct{}

func (*_DefaultTestCTRServiceServer) TraceSpan() string { return "" }

func (*_DefaultTestCTRServiceServer) mustBeTestCTRServiceServer() {}

type _DefaultTestCTRNoResultServiceServer struct{}

func (*_DefaultTestCTRNoResultServiceServer) Ping() {}

func (*_DefaultTestCTRNoResultServiceServer) mustBeTestCTRNoResultServiceServer() {}

type testCTRServiceServerER interface {
	TraceSpan() (string, ex.Error)

	mustBeTestCTRServiceServerER()
}

type _DefaultTestCTRServiceServerER struct{}

func (*_DefaultTestCTRServiceServerER) TraceSpan() (string, ex.Error) { return "", nil }

func (*_DefaultTestCTRServiceServerER) mustBeTestCTRServiceServerER() {}

type _DefaultTestCTRNoResultServiceServerER struct{}

func (*_DefaultTestCTRNoResultServiceServerER) Ping() error { return nil }

func (*_DefaultTestCTRNoResultServiceServerER) mustBeTestCTRNoResultServiceServerER() {}

type _CTRSpecDependency struct {
	Value string
}

type _CTRTraceServiceImpl struct {
	_DefaultTestCTRServiceServerER
	Ctx spec.Context        `inject:""`
	Dep *_CTRSpecDependency `inject:""`
}

func (s *_CTRTraceServiceImpl) TraceSpan() (string, ex.Error) {
	return s.Ctx.Trace().Span() + ":" + s.Dep.Value, nil
}

type _CTRNoResultServiceImpl struct {
	_DefaultTestCTRNoResultServiceServerER
}

func (*_CTRNoResultServiceImpl) Ping() error { return nil }

type _CTRErrorServiceImpl struct {
	_DefaultTestCTRServiceServerER
}

func (*_CTRErrorServiceImpl) TraceSpan() (string, ex.Error) {
	return "", ex.New(ex.OperationFailed, "container-executor-error")
}

type _CTRWrappedResult struct {
	Items []string
}

type testCTRWrappedServiceServer interface {
	Preview() _CTRWrappedResult

	mustBeTestCTRWrappedServiceServer()
}

type testCTRWrappedServiceServerER interface {
	Preview() (_CTRWrappedResult, ex.Error)

	mustBeTestCTRWrappedServiceServerER()
}

type _DefaultTestCTRWrappedServiceServer struct{}

func (*_DefaultTestCTRWrappedServiceServer) Preview() _CTRWrappedResult {
	ex.PanicNew(ex.InvalidRequest, "method preview is not implemented")
	return _CTRWrappedResult{}
}

func (*_DefaultTestCTRWrappedServiceServer) mustBeTestCTRWrappedServiceServer() {}

type _DefaultTestCTRWrappedServiceServerER struct{}

func (*_DefaultTestCTRWrappedServiceServerER) Preview() (_CTRWrappedResult, ex.Error) {
	return _CTRWrappedResult{}, nil
}

func (*_DefaultTestCTRWrappedServiceServerER) mustBeTestCTRWrappedServiceServerER() {}

type _CTRWrappedServiceImpl struct {
	_DefaultTestCTRWrappedServiceServer
}

func (*_CTRWrappedServiceImpl) Preview() _CTRWrappedResult {
	ex.PanicNew(ex.OperationFailed, "wrapped-service-error")
	return _CTRWrappedResult{}
}

type _CTRWrappedServiceServerERWrapper struct {
	_DefaultTestCTRWrappedServiceServer
	serverImpl testCTRWrappedServiceServer
}

func newCTRWrappedServiceServerERWrapper(serverImpl testCTRWrappedServiceServer) testCTRWrappedServiceServerER {
	return &_CTRWrappedServiceServerERWrapper{serverImpl: serverImpl}
}

func (service *_CTRWrappedServiceServerERWrapper) server() testCTRWrappedServiceServer {
	if service.serverImpl == nil {
		return &service._DefaultTestCTRWrappedServiceServer
	}
	return service.serverImpl
}

func (service *_CTRWrappedServiceServerERWrapper) Preview() (ret _CTRWrappedResult, err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	ret = service.server().Preview()
	return
}

func (*_CTRWrappedServiceServerERWrapper) mustBeTestCTRWrappedServiceServerER() {}

type containerExecutorMethodInfoStub struct {
	spec.MethodInfo
	name string
}

func (m containerExecutorMethodInfoStub) Name() string {
	return m.name
}

type containerExecutorMethodImplStub struct {
	spec.MethodImpl
	info   spec.MethodInfo
	method reflect.Method
}

func (m containerExecutorMethodImplStub) Info() spec.MethodInfo {
	return m.info
}

func (m containerExecutorMethodImplStub) Method() reflect.Method {
	return m.method
}

var _CTRTraceServiceSpec = &spec.ServiceSpec{
	Type:     spec.ServiceSpecTypeServer,
	Name:     "CTRTraceService",
	SkelName: "server.ctr.trace",

	ServerType:        reflect.TypeOf((*testCTRServiceServer)(nil)).Elem(),
	DefaultServerType: reflect.TypeOf(&_DefaultTestCTRServiceServer{}),

	ERServerType:        reflect.TypeOf((*testCTRServiceServerER)(nil)).Elem(),
	DefaultERServerType: reflect.TypeOf(&_DefaultTestCTRServiceServerER{}),

	Methods: []*spec.MethodSpec{{
		Name:       "TraceSpan",
		SkelName:   "traceSpan",
		ResultType: reflect.TypeOf(""),
	}},
}

var _CTRNoResultServiceSpec = &spec.ServiceSpec{
	Type:     spec.ServiceSpecTypeServer,
	Name:     "CTRNoResultService",
	SkelName: "server.ctr.noResult",

	ServerType:        reflect.TypeOf((*testCTRNoResultServiceServer)(nil)).Elem(),
	DefaultServerType: reflect.TypeOf(&_DefaultTestCTRNoResultServiceServer{}),

	ERServerType:        reflect.TypeOf((*testCTRNoResultServiceServerER)(nil)).Elem(),
	DefaultERServerType: reflect.TypeOf(&_DefaultTestCTRNoResultServiceServerER{}),

	Methods: []*spec.MethodSpec{{
		Name:     "Ping",
		SkelName: "ping",
	}},
}

var _CTRWrappedServiceSpec = &spec.ServiceSpec{
	Type:     spec.ServiceSpecTypeServer,
	Name:     "CTRWrappedService",
	SkelName: "server.ctr.wrapped",

	ServerType:          reflect.TypeOf((*testCTRWrappedServiceServer)(nil)).Elem(),
	DefaultServerType:   reflect.TypeOf(&_DefaultTestCTRWrappedServiceServer{}),
	ERServerType:        reflect.TypeOf((*testCTRWrappedServiceServerER)(nil)).Elem(),
	WrapperERServerCtor: newCTRWrappedServiceServerERWrapper,
	DefaultERServerType: reflect.TypeOf(&_DefaultTestCTRWrappedServiceServerER{}),
	Methods: []*spec.MethodSpec{{
		Name:       "Preview",
		SkelName:   "preview",
		ResultType: reflect.TypeOf(_CTRWrappedResult{}),
		ValidateResult: func(value any) error {
			result := value.(_CTRWrappedResult)
			if result.Items == nil {
				return ex.New(ex.OperationFailed, "result.Items cannot be nil")
			}
			return nil
		},
	}},
}

func init() {
	spec.Register(_CTRTraceServiceSpec)
	spec.Register(_CTRNoResultServiceSpec)
	spec.Register(_CTRWrappedServiceSpec)
}

func TestExecutorUsesCTRInjectionAndExecutionScoping(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_CTRTraceServiceImpl]()},
		Executor: NewContainerExecutor(nil, []di.BindApplier{
			func(b *di.Binder) {
				b.BindFactory(func() *_CTRSpecDependency {
					return &_CTRSpecDependency{Value: "dep"}
				}).In(di.ExecutionScope)
			},
		}),
	})
	traceMethod := _CTRTraceServiceSpec.Methods[0].Info()

	rpcResponse := srv.handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: traceMethod,
		MethodImplValue: mustContainerMethodImpl(t, reflect.TypeOf(&_CTRTraceServiceImpl{}), traceMethod),
	})

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	result, ok := rpcResponse.Result().(string)
	if !ok || !strings.HasSuffix(result, ":dep") || !meta.IsValidSpan(strings.TrimSuffix(result, ":dep")) {
		t.Fatalf("expected result to contain execution span and dependency, got %v", rpcResponse.Result())
	}
}

func TestContainerExecutorReturnsNilForMethodWithoutResult(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_CTRNoResultServiceImpl]()},
		Executor:     NewContainerExecutor(nil, nil),
	})

	rpcResponse := srv.handle(newContainerExecutorRequest(srv, _CTRNoResultServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	if rpcResponse.Result() != nil {
		t.Fatalf("expected nil result, got %#v", rpcResponse.Result())
	}
}

func TestContainerExecutorReturnsERMethodError(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_CTRErrorServiceImpl]()},
		Executor:     NewContainerExecutor(nil, nil),
	})

	rpcResponse := srv.handle(newContainerExecutorRequest(srv, _CTRTraceServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OperationFailed {
		t.Fatalf("expected operation failed error, got %#v", rpcResponse.Error())
	}
}

func TestContainerExecutorReturnsWrappedNonERMethodError(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_CTRWrappedServiceImpl]()},
		Executor:     NewContainerExecutor(nil, nil),
	})

	rpcResponse := srv.handle(newContainerExecutorRequest(srv, _CTRWrappedServiceSpec.Methods[0].Info()))

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OperationFailed {
		t.Fatalf("expected operation failed error, got %#v", rpcResponse.Error())
	}
}

func TestContainerExecutorUsesResolvedMethodImplInsteadOfMethodInfoName(t *testing.T) {
	srv := New(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*_CTRTraceServiceImpl]()},
		Executor: NewContainerExecutor(nil, []di.BindApplier{
			func(b *di.Binder) {
				b.BindFactory(func() *_CTRSpecDependency {
					return &_CTRSpecDependency{Value: "dep"}
				}).In(di.ExecutionScope)
			},
		}),
	})

	method := _CTRTraceServiceSpec.Methods[0].Info()
	methodImpl := mustContainerMethodImpl(t, reflect.TypeOf(&_CTRTraceServiceImpl{}), method)
	stubMethodInfo := containerExecutorMethodInfoStub{
		MethodInfo: method,
		name:       "MissingMethodName",
	}
	stubMethodImpl := containerExecutorMethodImplStub{
		MethodImpl: methodImpl,
		info:       stubMethodInfo,
		method:     methodImpl.Method(),
	}

	rpcResponse := srv.handle(&spec.RequestImpl{
		ContextValue:    context.Background(),
		TraceValue:      meta.InitialTrace(),
		MethodInfoValue: stubMethodInfo,
		MethodImplValue: stubMethodImpl,
	})

	if rpcResponse.Error() == nil || rpcResponse.Error().Code() != ex.OK {
		t.Fatalf("expected no error, got %#v", rpcResponse.Error())
	}
	result, ok := rpcResponse.Result().(string)
	if !ok || !strings.HasSuffix(result, ":dep") || !meta.IsValidSpan(strings.TrimSuffix(result, ":dep")) {
		t.Fatalf("expected result to contain execution span and dependency, got %v", rpcResponse.Result())
	}
}

func newContainerExecutorRequest(srv *Server, method spec.MethodInfo) spec.Request {
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

func mustContainerMethodImpl(t *testing.T, handlerType reflect.Type, method spec.MethodInfo) spec.MethodImpl {
	t.Helper()

	handlerDict := spec.NewImplDict()
	handlerDict.Add(handlerType)
	methodImpl, err := handlerDict.GetMethodImpl(method.Service().SkelName(), method.SkelName())
	if err != nil {
		t.Fatalf("GetMethodImpl() error = %v", err)
	}
	return methodImpl
}
