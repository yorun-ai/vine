package rpcproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
)

func (p *RpcProxy) handleIn(w http.ResponseWriter, r *http.Request) {
	if err := rpchttp.CheckRequestMethod(r); err != nil {
		p.writeGatewayError(w, r, ex.New(ex.InvalidRequest, "invalid request", ex.WithDetail(err.Error())))
		return
	}

	appState, rpcPath, exErr := p.resolveInboundAppState(r.URL.Path)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}
	if !appState.instance.TryStartWork() {
		p.writeGatewayError(w, r, ex.New(ex.ServiceUnavailable, "rpc proxy inbound target unavailable"))
		return
	}
	defer appState.instance.FinishWork()

	targetEndpoint := appState.serviceEndpoint
	if rpcinproc.IsEndpoint(targetEndpoint) {
		p.forwardInboundInproc(w, r, appState, targetEndpoint, rpcPath)
		return
	}

	targetURL := targetEndpoint + rpcPath
	p.forwardInbound(w, r, appState, targetURL)
}

func (p *RpcProxy) resolveInboundAppState(path string) (*_AppState, string, ex.Error) {
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(pathParts) != 3 {
		return nil, "", ex.New(ex.InvalidRequest, "invalid request path")
	}

	appState, ok := p.getAppStateByInstanceID(pathParts[0])
	if !ok {
		return nil, "", ex.New(ex.ServiceUnavailable, "rpc proxy inbound target unavailable")
	}

	rpcPath := "/" + pathParts[1] + "/" + pathParts[2]
	if !appState.hasService(pathParts[1]) {
		return nil, "", ex.New(ex.ServiceUnavailable, "rpc proxy inbound target unavailable")
	}
	return appState, rpcPath, nil
}

func (p *RpcProxy) forwardInbound(w http.ResponseWriter, r *http.Request, appState *_AppState, targetURL string) {
	resp, body, exErr := p.forwardInboundRequest(r, targetURL)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	p.writeValidatedInboundResponse(w, r, resp, body, appState.appInfo)
}

func (p *RpcProxy) forwardInboundInproc(w http.ResponseWriter, r *http.Request, appState *_AppState, targetEndpoint string, rpcPath string) {
	req := r.Clone(r.Context())
	req.URL.Path = rpcPath
	req.RequestURI = rpcPath
	rpcRequest, err := rpchttp.DecodeRequest(req)
	if err != nil {
		p.writeGatewayError(w, r, ex.New(ex.InvalidRequest, "invalid inbound request", ex.WithDetail(err.Error())))
		return
	}

	rpcResponse, exErr := p.roundTrip(targetEndpoint, rpcRequest)
	if exErr != nil {
		p.writeGatewayError(w, r, exErr)
		return
	}

	recorder := httptest.NewRecorder()
	if err := rpchttp.WriteResponse(recorder, r, rpcResponse); err != nil {
		p.writeGatewayError(w, r, ex.New(ex.ServiceUnavailable, "invalid inbound response", ex.WithDetail(err.Error())))
		return
	}

	resp := recorder.Result()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.writeGatewayError(w, r, ex.New(ex.ServiceUnavailable, "inbound response body cannot be read", ex.WithDetail(err.Error())))
		return
	}
	p.writeValidatedInboundResponse(w, r, resp, body, appState.appInfo)
}

func (p *RpcProxy) writeValidatedInboundResponse(w http.ResponseWriter, req *http.Request, resp *http.Response, body []byte, localApp meta.App) {
	serverApp, err := rpchttp.DecodeServerFromHeader(resp.Header)
	if err != nil {
		p.writeGatewayError(w, req, ex.New(ex.ServiceUnavailable, "invalid inbound response server", ex.WithDetail(err.Error())))
		return
	}

	if !rpchttp.ServerMatchesApp(serverApp, localApp) {
		p.writeGatewayError(w, req, ex.New(ex.ServiceUnavailable, "inbound response server mismatch"))
		return
	}

	writeInResponse(w, resp, body)
}

func (p *RpcProxy) forwardInboundRequest(r *http.Request, targetURL string) (*http.Response, []byte, ex.Error) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		return nil, nil, ex.New(ex.Internal, "failed to create proxy request", ex.WithDetail(err.Error()))
	}

	req.Header = r.Header.Clone()
	return p.forward(r.Context(), req)
}
