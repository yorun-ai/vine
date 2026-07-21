package webproxy

import (
	"errors"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/meta"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	"go.yorun.ai/vine/internal/util/httputil"
)

func (p *WebProxy) handleIn(w http.ResponseWriter, r *http.Request) {
	route, suffix, err := p.resolveInboundRoute(r.URL.Path)
	if err != nil {
		writeInboundError(w, err)
		return
	}

	targetEndpoint := route.endpoint
	if webinproc.IsEndpoint(targetEndpoint) {
		p.forwardInboundInproc(w, r, targetEndpoint, route.path+suffix)
		return
	}

	p.forwardInbound(w, r, targetEndpoint+suffix)
}

func parseInstanceScopedWebProxyPath(path string) (appInstanceID string, webName string, suffix string, err error) {
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] == "" {
		return "", "", "", errors.New("missing app instance id")
	}

	appInstanceID = parts[0]
	if !meta.IsValidInstanceId(appInstanceID) {
		return "", "", "", errors.New("invalid app instance id")
	}

	webName = parts[1]
	if webName == "" {
		return "", "", "", errors.New("missing web name")
	}

	if len(parts) == 2 {
		return appInstanceID, webName, "", nil
	}
	return appInstanceID, webName, "/" + parts[2], nil
}

func (p *WebProxy) resolveInboundRoute(path string) (*_WebRoute, string, error) {
	appInstanceID, webName, suffix, err := parseInstanceScopedWebProxyPath(path)
	if err != nil {
		return nil, "", err
	}

	route, ok := p.webRouteByInstanceID(appInstanceID, webName)
	if !ok {
		return nil, "", errWebEndpointUnavailable
	}

	return route, suffix, nil
}

func (p *WebProxy) forwardInboundInproc(w http.ResponseWriter, r *http.Request, targetEndpoint string, targetPath string) {
	if httputil.IsUpgradeRequest(r) {
		req := r.Clone(r.Context())
		req.URL.Path = targetPath
		req.RequestURI = targetPath
		if req.URL.RawQuery != "" {
			req.RequestURI += "?" + req.URL.RawQuery
		}
		if err := webinproc.ServeUpgrade(targetEndpoint, w, req); err != nil {
			http.Error(w, "proxy websocket request failed", http.StatusServiceUnavailable)
		}
		return
	}

	req := r.Clone(r.Context())
	req.URL.Path = targetPath
	req.RequestURI = req.URL.Path

	resp, err := webinproc.RoundTrip(targetEndpoint, req)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()
	httputil.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	httputil.CopyResponseBody(w, resp)
}

func (p *WebProxy) forwardInbound(w http.ResponseWriter, r *http.Request, targetURL string) {
	targetURL = appendRequestQuery(targetURL, r.URL.RawQuery)
	if httputil.IsUpgradeRequest(r) {
		if err := httputil.ForwardUpgrade(w, r, targetURL, nil); err != nil {
			http.Error(w, "proxy websocket request failed", http.StatusServiceUnavailable)
		}
		return
	}

	targetReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusServiceUnavailable)
		return
	}

	httputil.CopyHeader(targetReq.Header, r.Header)

	resp, err := p.transport.RoundTrip(targetReq)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()
	httputil.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	httputil.CopyResponseBody(w, resp)
}
