package inproc

import (
	"strings"
	"sync"

	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
)

const EndpointScheme = "rpc+inproc://"

// HandlerFunc receives an isolated request. A direct ServeRpc call takes the
// same snapshot that RoundTrip takes before dispatching asynchronously.
type HandlerFunc func(*spec.RequestImpl) spec.Response

func (f HandlerFunc) ServeRpc(request spec.Request) spec.Response {
	return f(spec.CloneInprocRequest(request))
}

type _Registration struct {
	handler HandlerFunc
}

// handlerByEndpoint is the concurrency-safe process-wide inproc registry.
var (
	registryMutex     sync.RWMutex
	handlerByEndpoint = map[string]*_Registration{}
)

// Register adds handler and returns an idempotent cleanup bound to this registration.
func Register(endpoint string, handler spec.RpcHandler) func() {
	vpre.Check(IsEndpoint(endpoint), "inproc endpoint %s must start with %s", endpoint, EndpointScheme)
	vpre.Check(len(endpoint) > len(EndpointScheme), "inproc endpoint host is empty")
	vpre.CheckNotNil(handler, "inproc handler cannot be nil")

	serve, ok := handler.(HandlerFunc)
	if !ok {
		serve = func(request *spec.RequestImpl) spec.Response { return handler.ServeRpc(request) }
	}
	registration := &_Registration{handler: serve}
	registryMutex.Lock()
	defer registryMutex.Unlock()
	vpre.CheckNil(handlerByEndpoint[endpoint], "inproc endpoint %s already registered", endpoint)
	handlerByEndpoint[endpoint] = registration

	var once sync.Once
	return func() {
		once.Do(func() {
			registryMutex.Lock()
			defer registryMutex.Unlock()
			if handlerByEndpoint[endpoint] == registration {
				delete(handlerByEndpoint, endpoint)
			}
		})
	}
}

func Unregister(endpoint string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	delete(handlerByEndpoint, endpoint)
}

func getHandler(endpoint string) (HandlerFunc, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	registration := handlerByEndpoint[endpoint]
	if registration == nil {
		return nil, false
	}
	return registration.handler, true
}

func IsEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, EndpointScheme)
}

// Endpoint builds a rpc+inproc endpoint from a scheme-less host path and route paths.
func Endpoint(hostPath string, paths ...string) string {
	var endpoint strings.Builder
	endpoint.WriteString(EndpointScheme + hostPath)
	for _, path := range paths {
		endpoint.WriteString(path)
	}
	return endpoint.String()
}
