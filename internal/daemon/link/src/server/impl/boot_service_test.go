package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestBootServiceReturnsRpcProxyEndpointPath(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag:       &flag.Flag{},
		InprocFlag: &app.InternalInprocFlag{},
	}

	info := service.GetInfo()

	assert.Equal(t, "/rpc/proxy/out", info.RpcProxyEndpointPath)
	assert.False(t, info.SkipDomainSchemas)
}

func TestBootServiceKeepsDomainSchemasForNetworkAppWithInprocHub(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag:       &flag.Flag{HubInprocMode: true},
		InprocFlag: &app.InternalInprocFlag{},
	}

	info := service.GetInfo()

	assert.False(t, info.SkipDomainSchemas)
}

func TestBootServiceKeepsDomainSchemasForInprocAppWithNetworkHub(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag:       &flag.Flag{},
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
	}

	info := service.GetInfo()

	assert.False(t, info.SkipDomainSchemas)
}

func TestBootServiceSkipsDomainSchemasWhenHubAndAppShareProcess(t *testing.T) {
	service := &BootServiceServerImpl{
		Flag:       &flag.Flag{HubInprocMode: true},
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
	}

	info := service.GetInfo()

	assert.True(t, info.SkipDomainSchemas)
}
