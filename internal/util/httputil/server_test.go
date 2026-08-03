package httputil

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdownServerForceClosesAfterGracefulTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	handlerStarted := make(chan struct{})
	handlerStopped := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(handlerStarted)
		<-request.Context().Done()
		close(handlerStopped)
	})}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("HTTP handler did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err = ShutdownServer(server, ctx)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), err)

	for name, done := range map[string]<-chan struct{}{
		"handler": handlerStopped,
		"request": requestDone,
	} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("%s did not stop after force close", name)
		}
	}
	serveErr := <-serveDone
	assert.ErrorIs(t, serveErr, http.ErrServerClosed)
}
