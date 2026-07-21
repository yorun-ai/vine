package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestBootServiceReturnsRpcProxyEndpointPath(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag: &flag.Flag{},
	}

	info := service.GetInfo()

	assert.Equal(t, "/rpc/proxy/out", info.RpcProxyEndpointPath)
	assert.False(t, info.SkipDomainSchemas)
}

func TestBootServiceReturnsSkipDomainSchemas(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag: &flag.Flag{HubInprocMode: true},
	}

	info := service.GetInfo()

	assert.True(t, info.SkipDomainSchemas)
}
