package gwutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	"go.yorun.ai/vine/internal/util/httputil"
)

type DoFunc func(req *http.Request) (*http.Response, error)

var httpClient = httputil.NewH2CClient()

// do is replaceable by same-package tests.
var do = func(req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

// ForwardRequest forwards portal requests to Vine endpoints. HTTP endpoints are
// expected to support h2c; inproc endpoints are dispatched directly.
func ForwardRequest(r *http.Request, endpoint string) (*http.Response, error) {
	ctx, cancel := httputil.ContextWithForwardTimeout(r)

	var response *http.Response
	var err error
	if ingressinproc.IsEndpoint(endpoint) {
		response, err = forwardIngressInprocRequest(ctx, r, endpoint)
	} else if rpcinproc.IsEndpoint(endpoint) {
		response, err = forwardRpcInprocRequest(ctx, r, endpoint)
	} else if webinproc.IsEndpoint(endpoint) {
		response, err = forwardWebInprocRequest(ctx, r, endpoint)
	} else {
		response, err = forwardHTTPRequest(ctx, r, endpoint)
	}

	if err != nil {
		cancel()
		return nil, err
	}
	response.Body = newReadCloserWithCancel(response.Body, cancel)
	return response, nil
}

func ForwardUpgrade(w http.ResponseWriter, r *http.Request, endpoint string) (bool, error) {
	if !httputil.IsUpgradeRequest(r) {
		return false, nil
	}
	if ingressinproc.IsEndpoint(endpoint) {
		return true, ingressinproc.ServeUpgrade(endpoint+r.URL.RequestURI(), w, r)
	}
	if !isHTTPEndpoint(endpoint) {
		return false, nil
	}
	return true, httputil.ForwardUpgrade(w, r, endpoint+r.URL.RequestURI(), nil)
}

func forwardIngressInprocRequest(ctx context.Context, r *http.Request, endpoint string) (*http.Response, error) {
	request := cloneForwardRequest(ctx, r)
	response, err := ingressinproc.RoundTrip(endpoint+r.URL.RequestURI(), request)
	if err != nil {
		return nil, normalizeForwardError(ctx, err)
	}
	return response, nil
}

func forwardRpcInprocRequest(ctx context.Context, r *http.Request, endpoint string) (*http.Response, error) {
	request := cloneForwardRequest(ctx, r)
	rpcRequest, err := rpchttp.DecodeRequest(request)
	if err != nil {
		return nil, invalidForwardRequestError(err)
	}
	rpcResponse, rpcErr := rpcinproc.RoundTrip(endpoint, rpcRequest)
	if rpcErr != nil {
		return nil, normalizeForwardError(ctx, rpcErr)
	}
	recorder := httptest.NewRecorder()
	if err := rpchttp.WriteResponse(recorder, request, rpcResponse); err != nil {
		return nil, err
	}
	return recorder.Result(), nil
}

func forwardWebInprocRequest(ctx context.Context, r *http.Request, endpoint string) (*http.Response, error) {
	request := cloneForwardRequest(ctx, r)
	registeredEndpoint, targetPath := splitWebInprocEndpoint(endpoint, r.URL.Path)
	urlValue := *request.URL
	urlValue.Path = targetPath
	urlValue.RawPath = ""
	request.URL = &urlValue
	request.RequestURI = urlValue.RequestURI()

	response, err := webinproc.RoundTrip(registeredEndpoint, request)
	if err != nil {
		return nil, normalizeForwardError(ctx, err)
	}
	return response, nil
}

func forwardHTTPRequest(ctx context.Context, r *http.Request, endpoint string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, r.Method, endpoint+r.URL.RequestURI(), r.Body)
	if err != nil {
		return nil, err
	}
	request.Header = r.Header.Clone()

	response, err := do(request)
	if err != nil {
		return nil, normalizeForwardError(ctx, err)
	}
	return response, nil
}

func isHTTPEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}
