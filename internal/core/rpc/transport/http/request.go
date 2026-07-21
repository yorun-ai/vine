package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

type _RequestDecoder struct {
	httpRequest *http.Request
	rpcRequest  *spec.RequestImpl
}

type _RequestPayloadJson struct {
	Params json.RawMessage `json:"params"`
}

type _RequestPayloadCbor struct {
	Params cbor.RawMessage `json:"params"`
}

func DecodeRequest(httpRequest *http.Request) (spec.Request, error) {
	decoder := &_RequestDecoder{
		httpRequest: httpRequest,
	}
	return decoder.decode()
}

func (d *_RequestDecoder) decode() (spec.Request, error) {
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
		d.decodeArguments,
	}
	for _, decode := range decodes {
		if err = decode(); err != nil {
			return nil, err
		}
	}

	return d.rpcRequest, nil
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
	methodInfo, ok := spec.GetMethodInfo(serviceSkelName, methodSkelName)
	if ok {
		d.rpcRequest.MethodInfoValue = methodInfo
		return nil
	}
	return fmt.Errorf("method %s/%s not found", serviceSkelName, methodSkelName)
}

func (d *_RequestDecoder) decodeArguments() error {
	methodInfo := d.rpcRequest.MethodInfo()
	if !methodInfo.HasArguments() {
		return nil
	}

	bodyBytes, err := io.ReadAll(d.httpRequest.Body)
	if err != nil {
		return fmt.Errorf("request body cannot be read")
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("missing request body")
	}

	arguments := methodInfo.NewArguments()
	if err = d.decodeArgumentsBytes(bodyBytes, arguments); err != nil {
		return err
	}

	err = methodInfo.ValidateArguments(arguments)
	if err != nil {
		return err
	}

	d.rpcRequest.ArgumentsValue = arguments
	return nil
}

func (d *_RequestDecoder) decodeArgumentsBytes(bodyBytes []byte, arguments any) error {
	var rawMessage []byte
	var unmarshal func(data []byte, v any) error
	switch MediaTypeOf(d.httpRequest.Header.Get(HeaderContentType)) {
	case ContentTypeJson:
		requestPayload := &_RequestPayloadJson{}
		if err := json.Unmarshal(bodyBytes, requestPayload); err != nil {
			return fmt.Errorf("request body cannot be parsed")
		}
		rawMessage = requestPayload.Params
		unmarshal = json.Unmarshal
	case ContentTypeCbor:
		requestPayload := &_RequestPayloadCbor{}
		if err := cbor.Unmarshal(bodyBytes, requestPayload); err != nil {
			return fmt.Errorf("request body cannot be parsed")
		}
		rawMessage = requestPayload.Params
		unmarshal = cbor.Unmarshal
	default:
		return fmt.Errorf("request body cannot be parsed")
	}

	if len(rawMessage) == 0 {
		return fmt.Errorf("missing request params")
	}
	if err := unmarshal(rawMessage, arguments); err != nil {
		return fmt.Errorf("request body cannot be parsed")
	}
	return nil
}

func encodeRequest(endpoint string, rpcRequest spec.Request) (*http.Request, error) {
	ctx := rpcRequest.Context()
	url := endpoint + rpcRequest.MethodInfo().FullURLPath()
	bodyBytes := bytes.NewReader(encodeArgumentsToBytes(rpcRequest))

	httpRequest, err := http.NewRequestWithContext(ctx, RequestMethod, url, bodyBytes)
	if err != nil {
		return nil, err
	}

	header := httpRequest.Header
	EncodeContentTypeHeadersToHeaderByMethod(header, rpcRequest.MethodInfo())
	EncodeRequestOptionsToHeader(header, rpcRequest.Context())
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
	deadline, ok := ctx.Deadline()
	if !ok {
		return
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return
	}
	EncodeOptionsToHeader(header, &Options{Timeout: timeout})
}

func encodeArgumentsToBytes(rpcRequest spec.Request) []byte {
	methodInfo := rpcRequest.MethodInfo()
	contentType := requestBodyContentType(methodInfo)
	var arguments any = &spec.EmptyArguments{}
	if methodInfo.HasArguments() {
		arguments = rpcRequest.Arguments()
		vpre.CheckNotNil(arguments, "request arguments cannot be nil for %s", methodInfo.Name())
	}

	switch contentType {
	case ContentTypeCbor:
		requestPayload := &_RequestPayloadCbor{
			Params: vcode.MustMarshalCbor(arguments),
		}
		return vcode.MustMarshalCbor(requestPayload)
	default:
		requestPayload := &_RequestPayloadJson{
			Params: vcode.MustMarshalJson(arguments),
		}
		return vcode.MustMarshalJson(requestPayload)
	}
}
