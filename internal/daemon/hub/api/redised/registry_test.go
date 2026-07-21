package redised

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppStatusKeys(t *testing.T) {
	assert.Equal(t, "app:demo.app:status:instance-1", FormatAppStatusKey("demo.app", "instance-1"))
	assert.Equal(t, "app:*:status:*", FormatAppStatusPattern())
}

func TestRpcKeys(t *testing.T) {
	assert.Equal(t, "rpc:demo.service:endpoint", FormatRpcServiceRegistrationPrefix("demo.service"))
	assert.Equal(t, "rpc:demo.service:endpoint:demo.app:instance-1", FormatRpcServiceRegistrationKey("demo.service", "demo.app", "instance-1"))
}
