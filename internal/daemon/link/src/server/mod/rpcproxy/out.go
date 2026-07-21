package rpcproxy

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/util/httputil"
)

func (p *RpcProxy) handleOut(w http.ResponseWriter, r *http.Request) {
	if err := rpchttp.CheckRequestMethod(r); err != nil {
		p.writeGatewayError(w, r, ex.New(ex.InvalidRequest, "invalid request", ex.WithDetail(err.Error())))
		return
	}

	serviceName, _, err := rpchttp.ParseServiceAndMethodFromPath(r.URL.Path)
	if err != nil {
		p.writeGatewayError(w, r, ex.New(ex.InvalidRequest, "invalid request path", ex.WithDetail(err.Error())))
		return
	}

	clientApp, err := rpchttp.DecodeClientFromHeader(r.Header)
	if err != nil {
		p.writeGatewayError(w, r, ex.New(ex.InvalidRequest, "invalid client header", ex.WithDetail(err.Error())))
		return
	}

	targetEndpoint, exErr := p.resolveOutboundEndpoint(serviceName, clientApp)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}

	targetURL := targetEndpoint + r.URL.Path
	p.forwardOutbound(w, r, targetURL)
}

func (p *RpcProxy) forwardOutbound(w http.ResponseWriter, r *http.Request, targetURL string) {
	resp, body, exErr := p.forwardOutboundRequest(r, targetURL)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}

	defer func() { _ = resp.Body.Close() }()
	httputil.CopyHeader(w.Header(), resp.Header)
	clearBodyHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (p *RpcProxy) forwardOutboundRequest(r *http.Request, targetURL string) (*http.Response, []byte, ex.Error) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		return nil, nil, ex.New(ex.Internal, "failed to create proxy request", ex.WithDetail(err.Error()))
	}

	req.Header = r.Header.Clone()
	return p.forward(r.Context(), req)
}

func (p *RpcProxy) serveRpcOut(rpcRequest spec.Request) spec.Response {
	serviceName := rpcRequest.MethodInfo().Service().SkelName()
	targetEndpoint, exErr := p.resolveOutboundEndpoint(serviceName, rpcRequest.Client())
	if exErr != nil {
		return &spec.ResponseImpl{
			ServerValue: p.App,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  exErr,
		}
	}

	rpcResponse, exErr := p.roundTrip(targetEndpoint, rpcRequest)
	if exErr != nil {
		return &spec.ResponseImpl{
			ServerValue: p.App,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  exErr,
		}
	}

	return rpcResponse
}

func (p *RpcProxy) resolveOutboundEndpoint(serviceName string, clientApp meta.App) (string, ex.Error) {
	appState, ok := p.getAppStateByInstanceID(clientApp.InstanceId())
	if !ok {
		return "", ex.New(ex.ServiceUnavailable, "rpc proxy outbound source unavailable")
	}

	if !meta.IsSame(appState.appInfo, clientApp) {
		return "", ex.New(ex.ClientForbidden, "client app mismatch")
	}

	p.retainService(serviceName, clientApp.InstanceId())
	registration, ok := p.nextServiceEndpoint(serviceName)
	if !ok {
		return "", ex.New(ex.ServiceUnavailable, "rpc proxy outbound target unavailable")
	}

	if targetAppState, ok := p.getAppStateByInstanceID(registration.AppInstanceId); ok {
		if !targetAppState.hasService(serviceName) {
			return "", ex.New(ex.ServiceUnavailable, "rpc proxy outbound local target unavailable")
		}
		if targetAppState.draining && targetAppState.appInfo.InstanceId() != clientApp.InstanceId() {
			return "", ex.New(ex.ServiceUnavailable, "rpc proxy outbound local target unavailable")
		}
		return targetAppState.serviceEndpoint, nil
	}

	return registration.Endpoint, nil
}
