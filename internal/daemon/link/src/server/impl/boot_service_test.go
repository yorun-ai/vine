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
}

func TestBootServiceSkipsDomainSchemasOnlyWhenHubAndAppShareProcess(t *testing.T) {
	tests := []struct {
		name          string
		hubInprocMode bool
		appInprocMode bool
		wantSkip      bool
	}{
		{name: "network hub and network app", hubInprocMode: false, appInprocMode: false, wantSkip: false},
		{name: "inproc hub and network app", hubInprocMode: true, appInprocMode: false, wantSkip: false},
		{name: "network hub and inproc app", hubInprocMode: false, appInprocMode: true, wantSkip: false},
		{name: "inproc hub and inproc app", hubInprocMode: true, appInprocMode: true, wantSkip: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &BootServiceServerImpl{
				Flag:       &flag.Flag{HubInprocMode: test.hubInprocMode},
				InprocFlag: &app.InternalInprocFlag{Enabled: test.appInprocMode},
			}

			assert.Equal(t, test.wantSkip, service.GetInfo().SkipDomainSchemas)
		})
	}
}
