package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vcode"
)

type _ResponseDecoder struct {
	methodInfo spec.MethodInfo

	httpResponse *http.Response
	rpcResponse  *spec.ResponseImpl

	statusCode ex.Code
}

type _ResponsePayloadJson struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type _ResponsePayloadCbor struct {
	Result cbor.RawMessage `json:"result"`
	Error  cbor.RawMessage `json:"error"`
}

type _DecodedResponseBody struct {
	ResultBytes []byte
	ErrorBytes  []byte
	Unmarshal   func([]byte, any) error
}

func decodeResponse(httpResponse *http.Response, methodInfo spec.MethodInfo) (spec.Response, error) {
	decoder := _ResponseDecoder{
		httpResponse: httpResponse,
		methodInfo:   methodInfo,
	}
	return decoder.decode()
}

func (d *_ResponseDecoder) decode() (spec.Response, error) {
	d.rpcResponse = &spec.ResponseImpl{
		MethodValue: d.methodInfo,
	}

	var err error
	decodes := []func() error{
		d.checkHttpHeaders,
		d.decodeServer,
		d.decodeStatusCode,
		d.decodeBody,
	}
	for _, decode := range decodes {
		if err = decode(); err != nil {
			return nil, err
		}
	}

	return d.rpcResponse, nil
}

func (d *_ResponseDecoder) checkHttpHeaders() error {
	return CheckResponseHeaders(d.httpResponse.Header)
}

func (d *_ResponseDecoder) decodeServer() (err error) {
	d.rpcResponse.ServerValue, err = DecodeServerFromHeader(d.httpResponse.Header)
	return err
}

func (d *_ResponseDecoder) decodeStatusCode() (err error) {
	d.statusCode, err = DecodeStatusCodeFromHeader(d.httpResponse.Header)
	return err
}

func (d *_ResponseDecoder) decodeBody() error {
	bodyBytes, err := io.ReadAll(d.httpResponse.Body)
	if err != nil {
		return fmt.Errorf("response body cannot be read")
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("missing response body")
	}

	decodedBody, err := d.decodeBodyPayload(bodyBytes)
	if err != nil {
		return err
	}

	if d.statusCode != ex.OK {
		if isEmptyErrorPayload(decodedBody.ErrorBytes) {
			return fmt.Errorf("response body error does not match response status")
		}
		exErr, decodeErr := ex.DecodeError(decodedBody.ErrorBytes, decodedBody.Unmarshal)
		if decodeErr != nil {
			return fmt.Errorf("invalid response body")
		}
		d.rpcResponse.ErrorValue = exErr
		return nil
	}

	if !isEmptyErrorPayload(decodedBody.ErrorBytes) {
		return fmt.Errorf("response body error does not match response status")
	}

	d.rpcResponse.ErrorValue = ex.NewOK()
	if !d.methodInfo.HasResult() {
		return nil
	}

	result := d.methodInfo.NewResult()
	if err = decodedBody.Unmarshal(decodedBody.ResultBytes, result); err != nil {
		return fmt.Errorf("response body cannot be parsed")
	}

	d.rpcResponse.ResultValue = reflect.ValueOf(result).Elem().Interface()
	return nil
}

func (d *_ResponseDecoder) decodeBodyPayload(bodyBytes []byte) (*_DecodedResponseBody, error) {
	switch MediaTypeOf(d.httpResponse.Header.Get(HeaderContentType)) {
	case ContentTypeJson:
		responsePayload := &_ResponsePayloadJson{}
		if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, fmt.Errorf("response body cannot be parsed")
		}
		return &_DecodedResponseBody{
			ResultBytes: responsePayload.Result,
			ErrorBytes:  responsePayload.Error,
			Unmarshal:   json.Unmarshal,
		}, nil
	case ContentTypeCbor:
		responsePayload := &_ResponsePayloadCbor{}
		if err := cbor.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, fmt.Errorf("response body cannot be parsed")
		}
		return &_DecodedResponseBody{
			ResultBytes: responsePayload.Result,
			ErrorBytes:  responsePayload.Error,
			Unmarshal:   cbor.Unmarshal,
		}, nil
	default:
		return nil, fmt.Errorf("response body cannot be parsed")
	}
}

func isEmptyErrorPayload(bytes []byte) bool {
	return len(bytes) == 0 ||
		string(bytes) == "null" ||
		len(bytes) == 1 && bytes[0] == 0xf6
}

func WriteRequestErrorResponse(w http.ResponseWriter, r *http.Request, server meta.App, err ex.Error) error {
	response := &spec.ResponseImpl{
		ServerValue: server,
		ErrorValue:  err,
		ResultValue: nil,
	}
	return WriteResponse(w, r, response)
}

func WriteResponse(w http.ResponseWriter, r *http.Request, rpcResponse spec.Response) error {
	if rpcResponse.Method() == nil {
		return writeResponseWithContentType(w, rpcResponse, ContentTypeJson)
	}
	return writeResponseWithContentType(w, rpcResponse, responseBodyContentType(r, rpcResponse.Method()))
}

func writeResponseWithContentType(w http.ResponseWriter, rpcResponse spec.Response, contentType string) error {
	header := w.Header()
	EncodeContentTypeHeadersToHeader(header, contentType)
	EncodeStatusCodeToHeader(header, rpcResponse.Error().Code())
	EncodeServerToHeader(header, rpcResponse.Server())
	w.WriteHeader(ResponseStatusCode)

	bodyBytes := encodeResponseToBytes(rpcResponse, contentType)
	_, err := w.Write(bodyBytes)
	return err
}

func encodeResponseToBytes(rpcResponse spec.Response, contentType string) []byte {
	switch contentType {
	case ContentTypeCbor:
		responsePayload := &_ResponsePayloadCbor{}
		if rpcResponse.Error().Type() == ex.NoError {
			responsePayload.Result = vcode.MustMarshalCbor(rpcResponse.Result())
			return vcode.MustMarshalCbor(responsePayload)
		}
		responsePayload.Result = vcode.MustMarshalCbor(nil)
		responsePayload.Error = ex.EncodeError(rpcResponse.Error(), vcode.MustMarshalCbor)
		return vcode.MustMarshalCbor(responsePayload)
	default:
		// Default to JSON as the transport fallback when CBOR is not selected.
		responsePayload := &_ResponsePayloadJson{}
		if rpcResponse.Error().Type() == ex.NoError {
			responsePayload.Result = vcode.MustMarshalJson(rpcResponse.Result())
			return vcode.MustMarshalJson(responsePayload)
		}
		responsePayload.Result = vcode.MustMarshalJson(nil)
		responsePayload.Error = ex.EncodeError(rpcResponse.Error(), vcode.MustMarshalJson)
		return vcode.MustMarshalJson(responsePayload)
	}
}

func ClearResponseErrorDetail(bodyBytes []byte, contentType string) ([]byte, error) {
	switch MediaTypeOf(contentType) {
	case ContentTypeJson:
		responsePayload := &_ResponsePayloadJson{}
		if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, err
		}
		if isEmptyErrorPayload(responsePayload.Error) {
			return bodyBytes, nil
		}
		errorBytes, err := ex.ClearErrorDetail(responsePayload.Error, json.Unmarshal, vcode.MustMarshalJson)
		if err != nil {
			return nil, err
		}
		responsePayload.Error = errorBytes
		return vcode.MustMarshalJson(responsePayload), nil
	case ContentTypeCbor:
		responsePayload := &_ResponsePayloadCbor{}
		if err := cbor.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, err
		}
		if isEmptyErrorPayload(responsePayload.Error) {
			return bodyBytes, nil
		}
		errorBytes, err := ex.ClearErrorDetail(responsePayload.Error, cbor.Unmarshal, vcode.MustMarshalCbor)
		if err != nil {
			return nil, err
		}
		responsePayload.Error = errorBytes
		return vcode.MustMarshalCbor(responsePayload), nil
	default:
		return bodyBytes, nil
	}
}
