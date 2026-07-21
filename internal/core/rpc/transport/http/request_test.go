package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type validateArgumentsInput struct {
	Name *string `arg:"0"`
}

type validateRequestServiceServer interface {
	mustBeValidateRequestServiceServer()
}

type defaultValidateRequestServiceServer struct{}

func (*defaultValidateRequestServiceServer) mustBeValidateRequestServiceServer() {}

type validateRequestServiceServerER interface {
	mustBeValidateRequestServiceServerER()
}

type defaultValidateRequestServiceServerER struct{}

func (*defaultValidateRequestServiceServerER) mustBeValidateRequestServiceServerER() {}

type _PingNoArgsServiceServer interface {
	mustBePingNoArgsServiceServer()
}

type _DefaultPingNoArgsServiceServer struct{}

func (*_DefaultPingNoArgsServiceServer) Ping() {}

func (*_DefaultPingNoArgsServiceServer) mustBePingNoArgsServiceServer() {}

type _PingNoArgsServiceServerER interface {
	mustBePingNoArgsServiceServerER()
}

type _DefaultPingNoArgsServiceServerER struct{}

func (*_DefaultPingNoArgsServiceServerER) Ping() error { return nil }

func (*_DefaultPingNoArgsServiceServerER) mustBePingNoArgsServiceServerER() {}

type _PingNoArgsServiceImpl struct {
	_DefaultPingNoArgsServiceServer
}

type _ValidateRequestServiceImpl struct {
	defaultValidateRequestServiceServer
}

func (*_ValidateRequestServiceImpl) CreateUser(*validateArgumentsInput) {}

func _newRequestTestHandlerDict(t *testing.T, serviceInfo *spec.ServiceSpec, handlerType reflect.Type) *spec.ImplDict {
	t.Helper()

	serviceInfo.Type = spec.ServiceSpecTypeServer
	spec.Register(serviceInfo)
	handlerDict := spec.NewImplDict()
	handlerDict.Add(handlerType)
	return handlerDict
}

func TestDecodeRequestRejectsMissingBody(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080"+method.FullURLPath(), bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	_, err = DecodeRequest(req)
	if err == nil || err.Error() != "missing request body" {
		t.Fatalf("expected missing request body error, got %v", err)
	}
}

func TestDecodeRequestWithoutArgumentsSkipsBody(t *testing.T) {
	service := &spec.ServiceSpec{
		Name:                "Service",
		SkelName:            "service",
		ServerType:          reflect.TypeOf((*_PingNoArgsServiceServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&_DefaultPingNoArgsServiceServer{}),
		ERServerType:        reflect.TypeOf((*_PingNoArgsServiceServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&_DefaultPingNoArgsServiceServerER{}),
		Methods: []*spec.MethodSpec{{
			Name:       "Ping",
			SkelName:   "ping_no_args",
			ResultType: testMethodInfo().ResultType,
		}},
	}
	_newRequestTestHandlerDict(t, service, reflect.TypeOf(&_PingNoArgsServiceImpl{}))
	method := service.Methods[0].Info()

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080/service/ping_no_args", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	got, err := DecodeRequest(req)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if got.MethodInfo() != method {
		t.Fatalf("unexpected method: got %v want %v", got.MethodInfo(), method)
	}
	if got.Arguments() != nil {
		t.Fatalf("expected nil arguments, got %#v", got.Arguments())
	}
}

func TestDecodeRequestRejectsInvalidJSONBody(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080"+method.FullURLPath(), bytes.NewBufferString("{"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	_, err = DecodeRequest(req)
	if err == nil || err.Error() != "request body cannot be parsed" {
		t.Fatalf("expected request body cannot be parsed error, got %v", err)
	}
}

func TestDecodeRequestRejectsInvalidGeneratedArguments(t *testing.T) {
	service := &spec.ServiceSpec{
		Name:                "Service",
		SkelName:            "service.request_test",
		ServerType:          reflect.TypeOf((*validateRequestServiceServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&defaultValidateRequestServiceServer{}),
		ERServerType:        reflect.TypeOf((*validateRequestServiceServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&defaultValidateRequestServiceServerER{}),
		Methods: []*spec.MethodSpec{{
			Name:              "CreateUser",
			SkelName:          "create_user",
			ArgumentsType:     reflect.TypeOf(validateArgumentsInput{}),
			ValidateArguments: func(any) error { return io.ErrUnexpectedEOF },
		}},
	}
	_newRequestTestHandlerDict(t, service, reflect.TypeOf(&_ValidateRequestServiceImpl{}))

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080/service.request_test/create_user", bytes.NewBufferString(`{"params":{"Name":null}}`))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	_, err = DecodeRequest(req)
	if err != io.ErrUnexpectedEOF {
		t.Fatalf("expected generated argument validation error, got %v", err)
	}
}

func TestEncodeRequestWrapsArgumentsInParams(t *testing.T) {
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
		t.Fatalf("encodeRequest() error = %v", err)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(bodyBytes); got != `{"params":{"Name":"vine"}}` {
		t.Fatalf("unexpected request body: got %s", got)
	}
}

func TestEncodeRequestWritesTimeoutOption(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]
	rpcCtx := testContext()
	reqCtx, cancel := context.WithTimeout(rpcCtx, time.Second)
	defer cancel()
	msg := &spec.RequestImpl{
		ContextValue:    reqCtx,
		TraceValue:      rpcCtx.Trace(),
		ActorValue:      rpcCtx.Actor(),
		InitiatorValue:  rpcCtx.Initiator(),
		ClientValue:     rpcCtx.Client(),
		MethodInfoValue: method,
		ArgumentsValue:  &pingArguments{Name: "vine"},
	}

	req, err := encodeRequest("http://localhost:8080", msg)
	if err != nil {
		t.Fatalf("encodeRequest() error = %v", err)
	}

	options, err := DecodeOptionsFromHeader(req.Header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	if options.Timeout <= 0 || options.Timeout > time.Second {
		t.Fatalf("unexpected timeout: %s", options.Timeout)
	}
}

func TestDecodeRequestAppliesTimeoutOption(t *testing.T) {
	service := &spec.ServiceSpec{
		Name:                "Service",
		SkelName:            "service.timeout",
		ServerType:          reflect.TypeOf((*_PingNoArgsServiceServer)(nil)).Elem(),
		DefaultServerType:   reflect.TypeOf(&_DefaultPingNoArgsServiceServer{}),
		ERServerType:        reflect.TypeOf((*_PingNoArgsServiceServerER)(nil)).Elem(),
		DefaultERServerType: reflect.TypeOf(&_DefaultPingNoArgsServiceServerER{}),
		Methods: []*spec.MethodSpec{{
			Name:       "Ping",
			SkelName:   "ping_timeout",
			ResultType: testMethodInfo().ResultType,
		}},
	}
	_newRequestTestHandlerDict(t, service, reflect.TypeOf(&_PingNoArgsServiceImpl{}))

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080/service.timeout/ping_timeout", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeOptionsToHeader(req.Header, &Options{Timeout: time.Second})
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	got, err := DecodeRequest(req)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	deadline, ok := got.Context().Deadline()
	if !ok {
		t.Fatal("expected decoded request context deadline")
	}
	timeout := time.Until(deadline)
	if timeout <= 0 || timeout > time.Second {
		t.Fatalf("unexpected timeout: %s", timeout)
	}
}

func TestDecodeRequestParsesCborBody(t *testing.T) {
	si := testServiceInfo()
	method := si.Methods()[0]
	body := cbor.RawMessage(mustTestMarshalCbor(t, &_RequestPayloadCbor{
		Params: mustTestMarshalCbor(t, &pingArguments{Name: "vine"}),
	}))

	req, err := http.NewRequest(RequestMethod, "http://localhost:8080"+method.FullURLPath(), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	EncodeContentTypeHeadersToHeaderByMethod(req.Header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), true, false))
	EncodeTraceToHeader(req.Header, testContext().Trace())
	EncodeClientToHeader(req.Header, testContext().Client())

	got, err := DecodeRequest(req)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	args, ok := got.Arguments().(*pingArguments)
	if !ok || args.Name != "vine" {
		t.Fatalf("unexpected arguments: %+v", got.Arguments())
	}
}

func TestEncodeRequestUsesCborForBinaryArgumentsType(t *testing.T) {
	method := newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), true, false)
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
		t.Fatalf("encodeRequest() error = %v", err)
	}
	if req.Header.Get(HeaderContentType) != ContentTypeCbor {
		t.Fatalf("unexpected request content-type: %s", req.Header.Get(HeaderContentType))
	}
	if req.Header.Get(HeaderAccept) != ContentTypeJson {
		t.Fatalf("unexpected request accept: %s", req.Header.Get(HeaderAccept))
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	got, err := cbor.Diagnose(bodyBytes)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got != `{`+`"params": {"Name": "vine"}`+`}` {
		t.Fatalf("unexpected cbor body: %s", got)
	}
}

func TestEncodeRequestUsesCborAcceptForBinaryResultType(t *testing.T) {
	method := newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, true)
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
		t.Fatalf("encodeRequest() error = %v", err)
	}
	if req.Header.Get(HeaderAccept) != ContentTypeCbor+", "+ContentTypeJson {
		t.Fatalf("unexpected request accept: %s", req.Header.Get(HeaderAccept))
	}
	if req.Header.Get(HeaderContentType) != ContentTypeJson {
		t.Fatalf("unexpected request content-type: %s", req.Header.Get(HeaderContentType))
	}
}

func mustTestMarshalCbor(t *testing.T, value any) []byte {
	t.Helper()
	data, err := cbor.Marshal(value)
	if err != nil {
		t.Fatalf("cbor.Marshal() error = %v", err)
	}
	return data
}
