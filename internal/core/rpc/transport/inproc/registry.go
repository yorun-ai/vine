package inproc

import (
	"strings"

	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

const EndpointScheme = "rpc+inproc://"

// handlerByEndpoint is the single process-wide inproc registry. It is mutated
// only during app startup/shutdown and is read-only while serving requests.
var handlerByEndpoint = map[string]spec.RpcHandler{}

func Register(endpoint string, handler spec.RpcHandler) {
	vpre.Check(IsEndpoint(endpoint), "inproc endpoint %s must start with %s", endpoint, EndpointScheme)
	vpre.Check(len(endpoint) > len(EndpointScheme), "inproc endpoint host is empty")
	vpre.CheckNotNil(handler, "inproc handler cannot be nil")
	vpre.CheckNil(handlerByEndpoint[endpoint], "inproc endpoint %s already registered", endpoint)
	handlerByEndpoint[endpoint] = handler
}

func Unregister(endpoint string) {
	delete(handlerByEndpoint, endpoint)
}

func getHandler(endpoint string) (spec.RpcHandler, bool) {
	handler := handlerByEndpoint[endpoint]
	return handler, handler != nil
}

func IsEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, EndpointScheme)
}

// Endpoint builds a rpc+inproc endpoint from a scheme-less host path and route paths.
func Endpoint(hostPath string, paths ...string) string {
	endpoint := EndpointScheme + hostPath
	for _, path := range paths {
		endpoint += path
	}
	return endpoint
}
