package http

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

type _ResponseDecoder struct {
	methodInfo spec.MethodInfo

	httpResponse *http.Response
	rpcResponse  *spec.ResponseImpl

	statusCode ex.Code
}

type _ResponsePayloadJson = rpchttp.JSONResponse
type _ResponsePayloadCbor = rpchttp.CBORResponse

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
	bodyBytes, err := ReadResponseBody(d.httpResponse)
	if err != nil {
		return fmt.Errorf("response body cannot be read: %w", err)
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("missing response body")
	}

	decodedBody, err := d.decodeBodyPayload(bodyBytes)
	if err != nil {
		return err
	}

	if d.statusCode != ex.OK {
		if rpchttp.IsEmptyErrorPayload(decodedBody.ErrorBytes) {
			return fmt.Errorf("response body error does not match response status")
		}
		exErr, decodeErr := ex.DecodeError(decodedBody.ErrorBytes, decodedBody.Unmarshal)
		if decodeErr != nil {
			return fmt.Errorf("invalid response body")
		}
		d.rpcResponse.ErrorValue = exErr
		return nil
	}

	if !rpchttp.IsEmptyErrorPayload(decodedBody.ErrorBytes) {
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

func (d *_ResponseDecoder) decodeBodyPayload(bodyBytes []byte) (*rpchttp.ResponsePayload, error) {
	switch MediaTypeOf(d.httpResponse.Header.Get(HeaderContentType)) {
	case ContentTypeJson:
		return rpchttp.DecodeJSONResponse(bodyBytes)
	case ContentTypeCbor:
		payload, err := rpchttp.DecodeCBORResponse(bodyBytes)
		if err != nil {
			return nil, fmt.Errorf("response body cannot be parsed")
		}
		return payload, nil
	default:
		return nil, fmt.Errorf("response body cannot be parsed")
	}
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
	encoder := vcode.DefaultEncoder()
	if method := rpcResponse.Method(); method != nil {
		encoder = skel.EncoderForSkelName(method.Service().SkelName())
	}
	var result, errorBytes []byte
	var encoded []byte
	var err error
	if contentType == ContentTypeCbor {
		if rpcResponse.Error().Type() == ex.NoError {
			result = encoder.MustMarshalCbor(rpcResponse.Result())
		} else {
			result = vcode.MustMarshalCbor(nil)
			errorBytes = ex.EncodeError(rpcResponse.Error(), vcode.MustMarshalCbor)
		}
		encoded, err = rpchttp.EncodeCBORResponse(result, errorBytes)
	} else {
		if rpcResponse.Error().Type() == ex.NoError {
			result = encoder.MustMarshalJson(rpcResponse.Result())
		} else {
			result = vcode.MustMarshalJson(nil)
			errorBytes = ex.EncodeError(rpcResponse.Error(), vcode.MustMarshalJson)
		}
		encoded, err = rpchttp.EncodeJSONResponse(result, errorBytes)
	}
	if err != nil {
		panic(err)
	}
	return encoded
}

func ClearResponseErrorDetail(bodyBytes []byte, contentType string) ([]byte, error) {
	switch MediaTypeOf(contentType) {
	case ContentTypeJson:
		responsePayload := &_ResponsePayloadJson{}
		if err := json.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, err
		}
		if rpchttp.IsEmptyErrorPayload(responsePayload.Error) {
			return bodyBytes, nil
		}
		errorBytes, err := ex.ClearErrorDetail(responsePayload.Error, unmarshalJson, vcode.MustMarshalJson)
		if err != nil {
			return nil, err
		}
		responsePayload.Error = errorBytes
		return rpchttp.EncodeJSONResponse(responsePayload.Result, responsePayload.Error)
	case ContentTypeCbor:
		responsePayload := &_ResponsePayloadCbor{}
		if err := cbor.Unmarshal(bodyBytes, responsePayload); err != nil {
			return nil, err
		}
		if rpchttp.IsEmptyErrorPayload(responsePayload.Error) {
			return bodyBytes, nil
		}
		errorBytes, err := ex.ClearErrorDetail(responsePayload.Error, cbor.Unmarshal, vcode.MustMarshalCbor)
		if err != nil {
			return nil, err
		}
		responsePayload.Error = errorBytes
		return rpchttp.EncodeCBORResponse(responsePayload.Result, responsePayload.Error)
	default:
		return bodyBytes, nil
	}
}

func unmarshalJson(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
