package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalRuntimeBuildsOnlyRequestedAdditionalServicerHandlers(t *testing.T) {
	app := newTestAppImpl()
	httpHandler, rpcHandler := app.AdditionalServicer(T[*ConsoleServiceServerImpl]())

	assert.NotNil(t, httpHandler)
	assert.NotNil(t, rpcHandler)
}
