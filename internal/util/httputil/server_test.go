package httputil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServerAppliesSharedRequestLimits(t *testing.T) {
	handler := http.NewServeMux()

	server := NewServer("127.0.0.1:8080", handler)

	assert.Equal(t, "127.0.0.1:8080", server.Addr)
	assert.Same(t, handler, server.Handler)
	assert.Equal(t, DefaultMaxHeaderValueCount, server.MaxHeaderValueCount)
}

func TestShutdownServerForceClosesAfterGracefulTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		handlerStarted := make(chan struct{})
		handlerStopped := make(chan struct{})
		server := httptest.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(handlerStarted)
			<-request.Context().Done()
			close(handlerStopped)
		}))
		client := server.Client()

		requestDone := make(chan struct{})
		go func() {
			defer close(requestDone)
			response, requestErr := client.Get(server.URL)
			if requestErr == nil {
				_ = response.Body.Close()
			}
		}()
		<-handlerStarted

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := ShutdownServer(server.Config, ctx)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), err)

		<-handlerStopped
		<-requestDone
	})
}
