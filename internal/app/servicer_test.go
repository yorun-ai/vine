package app

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/rpc/server"
)

func TestServicerHTTPHandlerUsesServerHandler(t *testing.T) {
	app := newTestAppImpl()
	server := server.New(server.Option{
		App:          app.info,
		HandlerTypes: []reflect.Type{T[*ConsoleServiceServerImpl]()},
	})
	servicer := &_Servicer{
		appInfo: app.info,
		server:  server,
	}

	assert.NotNil(t, servicer.httpHandler())
	assert.Equal(t, app.info, servicer.appInfo)
}

func TestInternalRuntimeBuildsOnlyRequestedAdditionalServicerHandlers(t *testing.T) {
	app := newTestAppImpl()
	httpHandler, rpcHandler := app.AdditionalServicer(T[*ConsoleServiceServerImpl]())

	assert.NotNil(t, httpHandler)
	assert.NotNil(t, rpcHandler)
}
