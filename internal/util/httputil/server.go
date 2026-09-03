package httputil

import (
	"context"
	"errors"
	"net/http"
)

// DefaultMaxHeaderValueCount is Vine's default limit for the number of header
// values in an HTTP request. It is intentionally stricter than net/http's
// default while leaving ample room for browser and infrastructure headers.
const DefaultMaxHeaderValueCount = 128

// NewServer creates an HTTP server with Vine's shared request limits.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:                addr,
		Handler:             handler,
		MaxHeaderValueCount: DefaultMaxHeaderValueCount,
	}
}

// ShutdownServer gracefully stops server and force-closes its active
// connections when the graceful-shutdown context expires or otherwise fails.
func ShutdownServer(server *http.Server, ctx context.Context) error {
	err := server.Shutdown(ctx)
	if err == nil {
		return nil
	}
	return errors.Join(err, server.Close())
}
