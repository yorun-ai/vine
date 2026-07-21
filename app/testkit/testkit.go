package testkit

import (
	"time"

	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
)

const (
	linkRpcInvokeEndpoint = link.InprocEndpoint + coreapp.PathRpcInvoke
	rpcProxyOutEndpoint   = link.InprocEndpoint + rpcproxy.PathOut
	registerTimeout       = 5 * time.Second
)

// Option configures a standalone application test runtime.
type Option struct {
	// SeedYAMLFile is the base Hub seed configuration file.
	SeedYAMLFile string
	// ConfigOverrides replaces application configuration values in the test seed.
	ConfigOverrides []ConfigOverride
}
