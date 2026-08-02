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

	target, exErr := p.resolveOutboundTarget(serviceName, clientApp)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}

	targetURL := target.endpoint + r.URL.Path
	p.forwardOutboundWithTransport(w, r, targetURL, target.transport)
}

func (p *RpcProxy) forwardOutbound(w http.ResponseWriter, r *http.Request, targetURL string) {
	p.forwardOutboundWithTransport(w, r, targetURL, p.transport)
}

func (p *RpcProxy) forwardOutboundWithTransport(w http.ResponseWriter, r *http.Request, targetURL string, transport http.RoundTripper) {
	resp, body, exErr := p.forwardOutboundRequestWithTransport(r, targetURL, transport)
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
	return p.forwardOutboundRequestWithTransport(r, targetURL, p.transport)
}

func (p *RpcProxy) forwardOutboundRequestWithTransport(r *http.Request, targetURL string, transport http.RoundTripper) (*http.Response, []byte, ex.Error) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		return nil, nil, ex.New(ex.Internal, "failed to create proxy request", ex.WithDetail(err.Error()))
	}

	req.Header = r.Header.Clone()
	return p.forwardWithTransport(r.Context(), req, transport)
}

func (p *RpcProxy) serveRpcOut(rpcRequest spec.Request) spec.Response {
	serviceName := rpcRequest.MethodInfo().Service().SkelName()
	target, exErr := p.resolveOutboundTarget(serviceName, rpcRequest.Client())
	if exErr != nil {
		return &spec.ResponseImpl{
			ServerValue: p.App,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  exErr,
		}
	}

	rpcResponse, exErr := p.roundTripWithTransport(target.endpoint, rpcRequest, target.transport)
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
	target, exErr := p.resolveOutboundTarget(serviceName, clientApp)
	return target.endpoint, exErr
}

func (p *RpcProxy) resolveOutboundTarget(serviceName string, clientApp meta.App) (_OutboundTarget, ex.Error) {
	appState, ok := p.getAppStateByInstanceID(clientApp.InstanceId())
	if !ok {
		return _OutboundTarget{}, ex.New(ex.ServiceUnavailable, "rpc proxy outbound source unavailable")
	}

	if !meta.IsSame(appState.appInfo, clientApp) {
		return _OutboundTarget{}, ex.New(ex.ClientForbidden, "client app mismatch")
	}

	p.retainService(serviceName, clientApp.InstanceId())
	registration, ok := p.nextServiceEndpoint(serviceName)
	if !ok {
		return _OutboundTarget{}, ex.New(ex.ServiceUnavailable, "rpc proxy outbound target unavailable")
	}

	if targetAppState, ok := p.getAppStateByInstanceID(registration.AppInstanceId); ok {
		if !targetAppState.hasService(serviceName) {
			return _OutboundTarget{}, ex.New(ex.ServiceUnavailable, "rpc proxy outbound local target unavailable")
		}
		if targetAppState.draining && targetAppState.appInfo.InstanceId() != clientApp.InstanceId() {
			return _OutboundTarget{}, ex.New(ex.ServiceUnavailable, "rpc proxy outbound local target unavailable")
		}
		return _OutboundTarget{endpoint: targetAppState.serviceEndpoint, transport: p.transport}, nil
	}

	transport, err := p.Identity.BackendTransport(registration.ServerIdentity, registration.Endpoint)
	if err != nil {
		return _OutboundTarget{}, ex.New(ex.ServiceUnavailable, "rpc proxy outbound target is insecure", ex.WithDetail(err.Error()))
	}
	return _OutboundTarget{endpoint: registration.Endpoint, transport: transport}, nil
}
