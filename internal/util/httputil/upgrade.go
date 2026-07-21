package httputil

import (
	"net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"strings"
)

func IsUpgradeRequest(r *http.Request) bool {
	return headerContainsToken(r.Header, "Connection", "upgrade") && r.Header.Get("Upgrade") != ""
}

func ForwardUpgrade(w http.ResponseWriter, r *http.Request, targetURL string, transport http.RoundTripper) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	proxy := &stdhttputil.ReverseProxy{
		Rewrite: func(request *stdhttputil.ProxyRequest) {
			request.Out.URL.Scheme = target.Scheme
			request.Out.URL.Host = target.Host
			request.Out.URL.Path = target.Path
			request.Out.URL.RawPath = target.RawPath
			request.Out.URL.RawQuery = target.RawQuery
			request.Out.Host = target.Host
			request.SetXForwarded()
		},
		Transport: transport,
	}
	proxy.ServeHTTP(w, r)
	return nil
}

func headerContainsToken(header http.Header, key string, token string) bool {
	for _, value := range header.Values(key) {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
