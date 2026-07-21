package access

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	"go.yorun.ai/vine/internal/util/httputil"
)

type _InvokeResponseBody[T any] struct {
	Result T                 `json:"result"`
	Error  *_InvokeErrorBody `json:"error"`
}

type _InvokeErrorBody struct {
	Message string `json:"message"`
}

func (o *Auther) buildInvokeRequest(serviceSkelName string, methodSkelName string, params map[string]any) *http.Request {
	// The host is only a placeholder for building a valid request; ForwardRequest rewrites it from the selected endpoint.
	return rpchttp.BuildInvokeRequest(rpchttp.InvokeRequest{
		Context:         o.Request.Context(),
		Endpoint:        "https://portal-access.local",
		ServiceSkelName: serviceSkelName,
		MethodSkelName:  methodSkelName,
		Params:          params,
		Trace:           o.Trace.NewChildTrace(),
		Client:          o.Initiator,
		Actor:           o.actor,
		Initiator:       o.Initiator,
	})
}

func (o *Auther) forwardInvokeRequest(request *http.Request, serviceSkelName string, serviceLabel string) (*http.Response, ex.Code, string, bool) {
	registration, configured := o.endpointManager.NextRpcEndpoint(serviceSkelName)
	if !configured || registration == nil {
		return nil, ex.ServiceUnavailable, serviceLabel + " is unavailable: " + serviceSkelName, false
	}

	response, err := gwutil.ForwardRequest(request, registration.Endpoint)
	if err != nil {
		code := ex.ServiceUnavailable
		if errors.Is(err, context.DeadlineExceeded) {
			code = ex.GatewayTimeout
		}
		return nil, code, serviceLabel + " forward failed: " + err.Error(), false
	}
	return response, ex.OK, "", true
}

func readInvokeResponse[T any](response *http.Response, serviceLabel string, badResponseMessage string, defaultErrorMessage string) (T, ex.Code, string, bool) {
	var zero T
	body, err := httputil.ReadResponseBody(response)
	if err != nil {
		return zero, ex.ServiceUnavailable, serviceLabel + " response body cannot be read", false
	}

	if response.StatusCode != http.StatusOK {
		return zero, mapInvokeHttpStatus(response.StatusCode), serviceLabel + " service returned status " + http.StatusText(response.StatusCode), false
	}

	statusCode, err := rpchttp.DecodeStatusCodeFromHeader(response.Header)
	if err != nil {
		return zero, ex.ServiceUnavailable, badResponseMessage, false
	}

	responseBody := &_InvokeResponseBody[T]{}
	if err = json.Unmarshal(body, responseBody); err != nil {
		return zero, ex.ServiceUnavailable, badResponseMessage, false
	}

	if statusCode == ex.OK {
		return responseBody.Result, ex.OK, "", true
	}

	message := defaultErrorMessage
	if responseBody.Error != nil && responseBody.Error.Message != "" {
		message = responseBody.Error.Message
	}
	return zero, mapInvokeStatusCode(statusCode), message, false
}

func mapInvokeHttpStatus(statusCode int) ex.Code {
	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		return ex.ClientForbidden
	}
	if statusCode == http.StatusGatewayTimeout {
		return ex.GatewayTimeout
	}
	return ex.ServiceUnavailable
}

func mapInvokeStatusCode(statusCode ex.Code) ex.Code {
	if !statusCode.IsUnresponsive() {
		return statusCode
	}
	if statusCode == ex.InvocationTimeout {
		return ex.GatewayTimeout
	}
	return ex.ServiceUnavailable
}
