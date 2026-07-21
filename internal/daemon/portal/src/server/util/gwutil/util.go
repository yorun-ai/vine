package gwutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	coreapp "go.yorun.ai/vine/internal/core/app"
)

// _ReadCloserWithCancel cancels the forward context when the response body is closed.
type _ReadCloserWithCancel struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func newReadCloserWithCancel(readCloser io.ReadCloser, cancel context.CancelFunc) _ReadCloserWithCancel {
	return _ReadCloserWithCancel{
		ReadCloser: readCloser,
		cancel:     cancel,
	}
}

func (c _ReadCloserWithCancel) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

func cloneForwardRequest(ctx context.Context, r *http.Request) *http.Request {
	request := r.Clone(ctx)
	request.Header = r.Header.Clone()
	return request
}

func ContextWithoutClientCancel(requestContext context.Context, gatewayContext context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestContext), timeout)
	stopGatewayCancel := context.AfterFunc(gatewayContext, cancel)
	return ctx, func() {
		stopGatewayCancel()
		cancel()
	}
}

var ErrInvalidForwardRequest = errors.New("invalid forward request")

func invalidForwardRequestError(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidForwardRequest, err)
}

func normalizeForwardError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return err
}

func splitWebInprocEndpoint(endpoint string, requestPath string) (string, string) {
	index := strings.Index(endpoint, coreapp.PathWebAccess)
	if index < 0 {
		return endpoint, requestPath
	}
	registeredEnd := index + len(coreapp.PathWebAccess)
	basePath := endpoint[registeredEnd:]
	targetPath := joinWebPath(coreapp.PathWebAccess, basePath, requestPath)
	if requestPath == "/" && targetPath != "/" {
		targetPath += "/"
	}
	return endpoint[:registeredEnd], targetPath
}

func joinWebPath(paths ...string) string {
	joined := ""
	for _, path := range paths {
		path = strings.Trim(path, "/")
		if path == "" {
			continue
		}
		joined += "/" + path
	}
	if joined == "" {
		return "/"
	}
	return joined
}
