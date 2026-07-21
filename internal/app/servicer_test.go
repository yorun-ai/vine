package app

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/rpc/server"
)

type testRPCSpec struct {
	Application
	ServicerEnabled
}

func (*testRPCSpec) Name() string {
	return "test.rpc"
}

func TestServicerHTTPHandlerUsesServerHandler(t *testing.T) {
	app := newTestAppImpl()
	spec := &testRPCSpec{}
	server := server.New(server.Option{
		App:          app.info,
		HandlerTypes: []reflect.Type{T[*ConsoleServiceServerImpl]()},
	})
	servicer := &_Servicer{
		appInfo: app.info,
		spec:    spec,
		server:  server,
	}

	assert.NotNil(t, servicer.httpHandler())
	assert.Equal(t, app.info, servicer.appInfo)
	assert.Same(t, spec, servicer.spec)
}
