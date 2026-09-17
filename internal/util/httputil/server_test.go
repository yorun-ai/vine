package httputil

import (
	"context"
	"errors"
	"io"
	"net"
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

func TestUnencryptedHTTP2ProtocolsServesHTTP1AndH2CPriorKnowledge(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	server := NewServer(listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	server.Protocols = UnencryptedHTTP2Protocols()
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	url := "http://" + listener.Addr().String() + "/"

	http1Response, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = http1Response.Body.Close() }()
	assert.Equal(t, 1, http1Response.ProtoMajor)

	h2cResponse, err := NewH2CClient().Get(url)
	assert.NoError(t, err)
	defer func() { _ = h2cResponse.Body.Close() }()
	assert.Equal(t, 2, h2cResponse.ProtoMajor)

	body, err := io.ReadAll(h2cResponse.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}
