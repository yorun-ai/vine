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
	"go.yorun.ai/vine/internal/core/mtls/mtlstest"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/daemon"
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

func TestServerServesOnlyControlRpcRoute(t *testing.T) {
	var requestPath string
	runtime := &_TestInternalRuntime{
		httpHandler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			requestPath = request.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}),
		rpcHandler: testRpcHandler(),
	}
	server := &Server{
		Context:         context.Background(),
		Flag:            &flag.Flag{ControlListen: "127.0.0.1:0"},
		InprocFlag:      &app.InternalInprocFlag{},
		InternalRuntime: runtime,
	}
	require.NoError(t, server.BeforeAppStart())
	t.Cleanup(server.BeforeAppStop)

	assert.Equal(t, []reflect.Type{
		app.T[*impl.InfoServiceServerImpl](),
		app.T[*impl.RegistryServiceServerImpl](),
		app.T[*impl.LockServiceServerImpl](),
		app.T[*impl.PortalRegistryServiceServerImpl](),
	}, runtime.handlerTypes)

	response, err := http.Get("http://" + server.httpServer.Addr + coreapp.PathRpcInvoke + "/vine.hub.control.InfoService/getInfo")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusNoContent, response.StatusCode)
	assert.Equal(t, "/vine.hub.control.InfoService/getInfo", requestPath)

	response, err = http.Get("http://" + server.httpServer.Addr + coreapp.PathWebAccess)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestServerUsesMutualTLS(t *testing.T) {
	ca := mtlstest.NewCA(t)
	hubIdentity := ca.Identity(t, daemon.HubIdentity.SPIFFEPath())
	linkIdentity := ca.Identity(t, daemon.LinkIdentity.SPIFFEPath())
	runtime := &_TestInternalRuntime{
		httpHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		rpcHandler: testRpcHandler(),
	}
	server := &Server{
		Context:         context.Background(),
		Flag:            &flag.Flag{ControlListen: "127.0.0.1:0"},
		InprocFlag:      &app.InternalInprocFlag{},
		InternalRuntime: runtime,
		Identity:        hubIdentity,
	}
	require.NoError(t, server.BeforeAppStart())
	t.Cleanup(server.BeforeAppStop)

	client := &http.Client{Transport: linkIdentity.HTTPTransport(daemon.HubIdentity.SPIFFEPath())}
	response, err := client.Get("https://" + server.httpServer.Addr + coreapp.PathRpcInvoke + "/vine.hub.control.InfoService/getInfo")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusNoContent, response.StatusCode)

	plainResponse, err := http.Get("http://" + server.httpServer.Addr + coreapp.PathRpcInvoke)
	require.NoError(t, err)
	require.NoError(t, plainResponse.Body.Close())
	assert.Equal(t, http.StatusBadRequest, plainResponse.StatusCode)
}

func TestServerRegistersDedicatedControlInprocEndpoint(t *testing.T) {
	runtime := &_TestInternalRuntime{
		httpHandler: http.NotFoundHandler(),
		rpcHandler:  testRpcHandler(),
	}
	server := &Server{
		InprocFlag:      &app.InternalInprocFlag{Enabled: true},
		InternalRuntime: runtime,
	}
	require.NoError(t, server.BeforeAppStart())
	endpoint := server.inprocEndpoint
	assert.Equal(t, rpcinproc.Endpoint(hubapp.HubControlInprocHostPath, coreapp.PathRpcInvoke), endpoint)
	assert.Panics(t, func() {
		rpcinproc.Register(endpoint, testRpcHandler())
	})

	server.BeforeAppStop()
	assert.NotPanics(t, func() {
		rpcinproc.Register(endpoint, testRpcHandler())
	})
	rpcinproc.Unregister(endpoint)
}

func TestServerReturnsControlListenError(t *testing.T) {
	previousListenTCP := listenTCP
	t.Cleanup(func() { listenTCP = previousListenTCP })
	listenTCP = func(string, string) (net.Listener, error) {
		return nil, errors.New("address unavailable")
	}

	server := &Server{
		Flag:       &flag.Flag{ControlListen: "127.0.0.1:7071"},
		InprocFlag: &app.InternalInprocFlag{},
		InternalRuntime: &_TestInternalRuntime{
			httpHandler: http.NotFoundHandler(),
			rpcHandler:  testRpcHandler(),
		},
	}

	require.EqualError(t, server.BeforeAppStart(), "hub control API listen failed: address unavailable")
	assert.Nil(t, server.rpcHTTPHandler)
	assert.Nil(t, server.rpcHandler)
}

func testRpcHandler() rpcspec.RpcHandler {
	return rpcspec.RpcHandlerFunc(func(rpcspec.Request) rpcspec.Response { return nil })
}
