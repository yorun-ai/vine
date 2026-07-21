package httputil

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

var defaultDialer net.Dialer

// NewH2CClient creates a plain HTTP/2 client for Vine h2c endpoints.
func NewH2CClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP:          true,
			DisableCompression: true,
			DialTLSContext: func(ctx context.Context, network string, addr string, _ *tls.Config) (net.Conn, error) {
				return defaultDialer.DialContext(ctx, network, addr)
			},
		},
	}
}
