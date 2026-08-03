package httputil

import (
	"context"
	"errors"
	"net/http"
)

// ShutdownServer gracefully stops server and force-closes its active
// connections when the graceful-shutdown context expires or otherwise fails.
func ShutdownServer(server *http.Server, ctx context.Context) error {
	err := server.Shutdown(ctx)
	if err == nil {
		return nil
	}
	return errors.Join(err, server.Close())
}
