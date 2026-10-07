package http

import (
	"fmt"
	"net/http"
	"reflect"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

type _ResponseDecoder struct {
	methodInfo spec.MethodInfo

	httpResponse *http.Response
	rpcResponse  *spec.ResponseImpl

	statusCode ex.Code
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
	return rpchttp.DecodeResponse(bodyBytes, d.httpResponse.Header.Get(HeaderContentType))
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
	var result any
	var errorValue any
	if rpcResponse.Error().Type() == ex.NoError {
		result = rpcResponse.Result()
	} else {
		errorValue = rpcResponse.Error()
	}
	encoded, err := rpchttp.EncodeResponse(result, errorValue, contentType)
	vpre.MustNil(err)
	return encoded
}

func ClearResponseErrorDetail(bodyBytes []byte, contentType string) ([]byte, error) {
	if !rpchttp.IsValidContentType(contentType) {
		return bodyBytes, nil
	}
	payload, err := rpchttp.DecodeResponse(bodyBytes, contentType)
	if err != nil {
		return nil, err
	}
	if rpchttp.IsEmptyErrorPayload(payload.ErrorBytes) {
		return bodyBytes, nil
	}
	errorValue, err := ex.DecodeError(payload.ErrorBytes, payload.Unmarshal)
	if err != nil {
		return nil, err
	}
	var options []ex.ErrorOption
	if errorValue.Reason() != "" {
		options = append(options, ex.WithReason(errorValue.Reason()))
	}
	return payload.EncodeWithError(ex.New(errorValue.Code(), errorValue.Message(), options...))
}
