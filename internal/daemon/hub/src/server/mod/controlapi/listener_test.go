package controlapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.yorun.ai/vine/internal/app"
	coreapp "go.yorun.ai/vine/internal/core/app"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	impl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/control"
)

type _TestInternalRuntime struct {
	handlerTypes []reflect.Type
	httpHandler  http.Handler
	rpcHandler   rpcspec.RpcHandler
}

func (r *_TestInternalRuntime) AdditionalServicer(handlerTypes ...reflect.Type) (http.Handler, rpcspec.RpcHandler) {
	r.handlerTypes = handlerTypes
	return r.httpHandler, r.rpcHandler
}

func TestListenerServesOnlyControlRpcRoute(t *testing.T) {
	var requestPath string
	runtime := &_TestInternalRuntime{
		httpHandler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			requestPath = request.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}),
		rpcHandler: testRpcHandler(),
	}
	listener := &Listener{
		Context:         context.Background(),
		Flag:            &flag.Flag{ControlListen: "127.0.0.1:0"},
		InprocFlag:      &app.InternalInprocFlag{},
		InternalRuntime: runtime,
	}
	require.NoError(t, listener.BeforeAppStart())
	t.Cleanup(listener.BeforeAppStop)

	assert.Equal(t, []reflect.Type{
		app.T[*impl.InfoServiceServerImpl](),
		app.T[*impl.RegistryServiceServerImpl](),
	}, runtime.handlerTypes)

	response, err := http.Get("http://" + listener.server.Addr + coreapp.PathRpcInvoke + "/vine.hub.control.InfoService/getInfo")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusNoContent, response.StatusCode)
	assert.Equal(t, "/vine.hub.control.InfoService/getInfo", requestPath)

	response, err = http.Get("http://" + listener.server.Addr + coreapp.PathWebAccess)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestListenerRegistersDedicatedControlInprocEndpoint(t *testing.T) {
	runtime := &_TestInternalRuntime{
		httpHandler: http.NotFoundHandler(),
		rpcHandler:  testRpcHandler(),
	}
	listener := &Listener{
		InprocFlag:      &app.InternalInprocFlag{Enabled: true},
		InternalRuntime: runtime,
	}
	require.NoError(t, listener.BeforeAppStart())
	endpoint := listener.inprocEndpoint
	assert.Equal(t, rpcinproc.Endpoint(hubapp.HubControlInprocHostPath, coreapp.PathRpcInvoke), endpoint)
	assert.Panics(t, func() {
		rpcinproc.Register(endpoint, testRpcHandler())
	})

	listener.BeforeAppStop()
	assert.NotPanics(t, func() {
		rpcinproc.Register(endpoint, testRpcHandler())
	})
	rpcinproc.Unregister(endpoint)
}

func TestListenerReturnsControlListenError(t *testing.T) {
	previousListenTCP := listenTCP
	t.Cleanup(func() { listenTCP = previousListenTCP })
	listenTCP = func(string, string) (net.Listener, error) {
		return nil, errors.New("address unavailable")
	}

	listener := &Listener{
		Flag:       &flag.Flag{ControlListen: "127.0.0.1:7071"},
		InprocFlag: &app.InternalInprocFlag{},
		InternalRuntime: &_TestInternalRuntime{
			httpHandler: http.NotFoundHandler(),
			rpcHandler:  testRpcHandler(),
		},
	}

	require.EqualError(t, listener.BeforeAppStart(), "hub control API listen failed: address unavailable")
	assert.Nil(t, listener.rpcHTTPHandler)
	assert.Nil(t, listener.rpcHandler)
}

func testRpcHandler() rpcspec.RpcHandler {
	return rpcspec.RpcHandlerFunc(func(rpcspec.Request) rpcspec.Response { return nil })
}
