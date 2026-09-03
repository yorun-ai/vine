package rpcproxy

import (
	"context"
	"errors"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/util/httputil"
)

func (p *RpcProxy) roundTrip(endpoint string, rpcRequest spec.Request) (spec.Response, ex.Error) {
	return p.roundTripWithTransport(endpoint, rpcRequest, p.transport)
}

func (p *RpcProxy) roundTripWithTransport(endpoint string, rpcRequest spec.Request, transport http.RoundTripper) (spec.Response, ex.Error) {
	if rpcinproc.IsEndpoint(endpoint) {
		return rpcinproc.RoundTrip(endpoint, rpcRequest)
	}
	return rpchttp.RoundTripWithTransport(endpoint, rpcRequest, transport)
}

func (p *RpcProxy) forward(reqCtx context.Context, req *http.Request) (*http.Response, []byte, ex.Error) {
	return p.forwardWithTransport(reqCtx, req, p.transport)
}

func (p *RpcProxy) forwardWithTransport(reqCtx context.Context, req *http.Request, transport http.RoundTripper) (*http.Response, []byte, ex.Error) {
	resp, err := transport.RoundTrip(req)
	if err != nil {
		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
			return nil, nil, ex.New(ex.GatewayTimeout, "proxy request timed out")
		}
		return nil, nil, ex.New(ex.ServiceUnavailable, "proxy request failed", ex.WithDetail(err.Error()))
	}
	body, readErr := rpchttp.ReadResponseBody(resp)
	if readErr != nil {
		_ = resp.Body.Close()
		return nil, nil, ex.New(ex.ServiceUnavailable, "proxy response body cannot be read", ex.WithDetail(readErr.Error()))
	}
	return resp, body, nil
}

func (p *RpcProxy) writeGatewayError(w http.ResponseWriter, r *http.Request, err ex.Error) {
	_ = rpchttp.WriteRequestErrorResponse(w, r, p.App, mapGatewayResponseError(err))
}

func mapGatewayResponseError(err ex.Error) ex.Error {
	code := mapGatewayResponseCode(err.Code())
	if code == err.Code() {
		return err
	}

	options := []ex.ErrorOption{}
	if err.Reason() != "" {
		options = append(options, ex.WithReason(err.Reason()))
	}
	if err.Detail() != "" {
		options = append(options, ex.WithDetail(err.Detail()))
	}
	return ex.New(code, err.Message(), options...)
}

func mapGatewayResponseCode(code ex.Code) ex.Code {
	if !code.IsUnresponsive() {
		return code
	}
	if code == ex.InvocationTimeout {
		return ex.GatewayTimeout
	}
	return ex.ServiceUnavailable
}

func clearBodyHeaders(header http.Header) {
	header.Del(rpchttp.HeaderContentLength)
}

func writeInResponse(w http.ResponseWriter, resp *http.Response, body []byte) {
	httputil.CopyHeader(w.Header(), resp.Header)
	clearBodyHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}
