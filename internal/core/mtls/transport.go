package mtls

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/http2"
)

func newH2CTransport() http.RoundTripper {
	return &http2.Transport{
		AllowHTTP:          true,
		DisableCompression: true,
		DialTLSContext: func(ctx context.Context, network string, addr string, _ *tls.Config) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		},
	}
}

// BackendTransport validates a discovered endpoint before creating a client
// transport for the component identity recorded with that endpoint.
func (i *Identity) BackendTransport(serverIdentity string, endpoint string) (http.RoundTripper, error) {
	if !i.Enabled() {
		return i.HTTPTransport(serverIdentity), nil
	}
	if serverIdentity == "" {
		return nil, errors.New("mTLS backend server identity is missing")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse mTLS backend endpoint: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("mTLS backend endpoint must use https, got %q", parsed.Scheme)
	}
	return i.HTTPTransport(serverIdentity), nil
}
