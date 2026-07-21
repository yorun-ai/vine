package httputil

import (
	"context"
	"net/http"
	"time"
)

const (
	DefaultHttpRequestTimeout = 30 * time.Second
	DefaultSSERequestTimeout  = 600 * time.Second
	DefaultSSEIdleTimeout     = 60 * time.Second
)

func ContextWithForwardTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	if _, ok := r.Context().Deadline(); ok {
		return context.WithCancel(r.Context())
	}

	if IsEventStreamRequest(r) {
		return context.WithTimeout(r.Context(), DefaultSSERequestTimeout)
	}

	return context.WithTimeout(r.Context(), DefaultHttpRequestTimeout)
}
