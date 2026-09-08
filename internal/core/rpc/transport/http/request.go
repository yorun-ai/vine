package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
	rpchttp "go.yorun.ai/vrpc/transport/http"
)

type _RequestDecoder struct {
	httpRequest     *http.Request
	rpcRequest      *spec.RequestImpl
	serviceSkelName string
	methodSkelName  string
	bodyBytes       []byte
}

type RejectionDiagnostic struct {
	Trace           meta.Trace
	Client          meta.App
	Method          spec.MethodInfo
	ServiceSkelName string
	MethodSkelName  string
}

func DecodeRequest(httpRequest *http.Request) (spec.Request, error) {
	request, _, err := DecodeRequestWithDiagnostic(httpRequest)
	return request, err
}

func DecodeRequestWithDiagnostic(httpRequest *http.Request) (spec.Request, *RejectionDiagnostic, error) {
	decoder := &_RequestDecoder{
		httpRequest: httpRequest,
	}
	return decoder.decode()
}

func (d *_RequestDecoder) decode() (spec.Request, *RejectionDiagnostic, error) {
	d.rpcRequest = &spec.RequestImpl{
		ContextValue: d.httpRequest.Context(),
	}

	var err error
	decodes := []func() error{
		d.checkHttpMethod,
		d.checkHttpHeaders,
		d.decodeOptions,
		d.decodeTrace,
		d.decodeClient,
		d.decodeInitiator,
		d.decodeActor,
		d.decodeMethod,
		d.readBody,
		d.decodeArguments,
	}
	for _, decode := range decodes {
		if err = decode(); err != nil {
			diagnostic := new(RejectionDiagnostic{
				Trace:           d.rpcRequest.TraceValue,
				Client:          d.rpcRequest.ClientValue,
				Method:          d.rpcRequest.MethodInfoValue,
				ServiceSkelName: d.serviceSkelName,
				MethodSkelName:  d.methodSkelName,
			})
			d.rpcRequest.Cancel()
			return nil, diagnostic, err
		}
	}

	return d.rpcRequest, nil, nil
}

func (d *_RequestDecoder) checkHttpMethod() error {
	return CheckRequestMethod(d.httpRequest)
}

func (d *_RequestDecoder) checkHttpHeaders() error {
	return CheckRequestHeaders(d.httpRequest.Header)
}

func (d *_RequestDecoder) decodeOptions() error {
	options, err := DecodeOptionsFromHeader(d.httpRequest.Header)
	if err != nil {
		return err
	}
	if options.Timeout <= 0 {
		return nil
	}

	reqContext, cancel := context.WithTimeout(d.rpcRequest.ContextValue, options.Timeout)
	d.rpcRequest.ContextValue = reqContext
	d.rpcRequest.CancelValue = cancel
	return nil
}

func (d *_RequestDecoder) decodeTrace() (err error) {
	d.rpcRequest.TraceValue, err = DecodeTraceFromHeader(d.httpRequest.Header)
	return err
}

func (d *_RequestDecoder) decodeClient() (err error) {
	d.rpcRequest.ClientValue, err = DecodeClientFromHeader(d.httpRequest.Header)
	return err
}

func (d *_RequestDecoder) decodeInitiator() (err error) {
	d.rpcRequest.InitiatorValue, err = DecodeInitiatorFromHeader(d.httpRequest.Header)
	return err
}

func (d *_RequestDecoder) decodeActor() (err error) {
	d.rpcRequest.ActorValue, err = DecodeActorFromHeader(d.httpRequest.Header)
	return err
}

func (d *_RequestDecoder) decodeMethod() error {
	serviceSkelName, methodSkelName, err := ParseServiceAndMethodFromPath(d.httpRequest.URL.Path)
	if err != nil {
		return err
	}
	d.serviceSkelName = serviceSkelName
	d.methodSkelName = methodSkelName
	methodInfo, ok := spec.GetMethodInfo(serviceSkelName, methodSkelName)
	if ok {
		d.rpcRequest.MethodInfoValue = methodInfo
		return nil
	}
	return fmt.Errorf("method %s/%s not found", serviceSkelName, methodSkelName)
}

func (d *_RequestDecoder) readBody() (err error) {
	d.bodyBytes, err = ReadRequestBody(d.httpRequest)
	if err != nil {
		return fmt.Errorf("request body cannot be read: %w", err)
	}
	return err
}

func (d *_RequestDecoder) decodeArguments() error {
	methodInfo := d.rpcRequest.MethodInfo()
	if !methodInfo.HasArguments() {
		return nil
	}

	if len(d.bodyBytes) == 0 {
		return fmt.Errorf("missing request body")
	}

	arguments := methodInfo.NewArguments()
	if err := d.decodeArgumentsBytes(d.bodyBytes, arguments); err != nil {
		return err
	}

	err := methodInfo.ValidateArguments(arguments)
	if err != nil {
		return err
	}

	d.rpcRequest.ArgumentsValue = arguments
	return nil
}

func (d *_RequestDecoder) decodeArgumentsBytes(bodyBytes []byte, arguments any) error {
	var raw []byte
	var err error
	unmarshal := unmarshalJson
	switch MediaTypeOf(d.httpRequest.Header.Get(HeaderContentType)) {
	case ContentTypeJson:
		raw, err = rpchttp.DecodeJSONRequest(bodyBytes)
	case ContentTypeCbor:
		raw, err = rpchttp.DecodeCBORRequest(bodyBytes)
		unmarshal = cbor.Unmarshal
	default:
		return fmt.Errorf("request body cannot be parsed")
	}
	if err != nil {
		return err
	}
	if err := unmarshal(raw, arguments); err != nil {
		return fmt.Errorf("request body cannot be parsed")
	}
	return nil
}

func encodeRequest(endpoint string, rpcRequest spec.Request) (request *http.Request, err error) {
	defer func() {
		if recover() != nil {
			request = nil
			err = fmt.Errorf("request cannot be encoded")
		}
	}()

	ctx := rpcRequest.Context()
	encodedArguments, err := encodeArgumentsToBytes(rpcRequest)
	if err != nil {
		return nil, err
	}
	httpRequest, err := rpchttp.NewRequest(ctx, endpoint, rpcRequest.MethodInfo().FullURLPath(), encodedArguments, requestBodyContentType(rpcRequest.MethodInfo()), requestAcceptContentType(rpcRequest.MethodInfo()))
	if err != nil {
		return nil, err
	}
	header := httpRequest.Header
	EncodeTraceToHeader(header, rpcRequest.Trace())
	EncodeClientToHeader(header, rpcRequest.Client())
	if actor := rpcRequest.Actor(); actor != nil {
		EncodeActorToHeader(header, actor)
	}
	if initiator := rpcRequest.Initiator(); initiator != nil {
		EncodeInitiatorToHeader(header, initiator)
	}

	return httpRequest, nil
}

func EncodeRequestOptionsToHeader(header http.Header, ctx context.Context) {
	rpchttp.EncodeRequestOptionsToHeader(header, ctx)
}

func encodeArgumentsToBytes(rpcRequest spec.Request) (encoded []byte, err error) {
	defer func() {
		if recover() != nil {
			encoded = nil
			err = fmt.Errorf("request arguments cannot be encoded")
		}
	}()

	methodInfo := rpcRequest.MethodInfo()
	encoder := skel.EncoderForSkelName(methodInfo.Service().SkelName())
	contentType := requestBodyContentType(methodInfo)
	var arguments any = &spec.EmptyArguments{}
	if methodInfo.HasArguments() {
		arguments = rpcRequest.Arguments()
		vpre.CheckNotNil(arguments, "request arguments cannot be nil for %s", methodInfo.Name())
	}

	switch contentType {
	case ContentTypeCbor:
		encodedArguments, err := encoder.MarshalCbor(arguments)
		if err != nil {
			return nil, err
		}
		return rpchttp.EncodeCBORRequest(encodedArguments)
	default:
		encodedArguments, err := encoder.MarshalJson(arguments)
		if err != nil {
			return nil, err
		}
		return rpchttp.EncodeJSONRequest(encodedArguments)
	}
}

type InvokeRequest struct {
	Context         context.Context
	Endpoint        string
	ServiceSkelName string
	MethodSkelName  string
	Params          any
	Trace           meta.Trace
	Client          meta.App
	Actor           meta.Actor
	Initiator       meta.Initiator
}

func BuildInvokeRequest(invokeRequest InvokeRequest) *http.Request {
	params := vcode.MustMarshalJson(invokeRequest.Params)
	body, err := rpchttp.EncodeJSONRequest(params)
	vpre.MustNil(err)

	request, err := rpchttp.NewRequest(invokeRequest.Context, strings.TrimRight(invokeRequest.Endpoint, "/"), "/"+invokeRequest.ServiceSkelName+"/"+invokeRequest.MethodSkelName, body, ContentTypeJson, ContentTypeJson)
	vpre.MustNil(err)
	header := request.Header
	EncodeTraceToHeader(header, invokeRequest.Trace)
	EncodeClientToHeader(header, invokeRequest.Client)
	if invokeRequest.Actor != nil {
		EncodeActorToHeader(header, invokeRequest.Actor)
	}
	if invokeRequest.Initiator != nil {
		EncodeInitiatorToHeader(header, invokeRequest.Initiator)
	}
	return request
}
