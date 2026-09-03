package inproc

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/util/vpre"
)

const EndpointScheme = "web+inproc://"

type _Registration struct {
	handler http.Handler
}

// handlerByEndpoint is the concurrency-safe process-wide inproc registry.
var (
	registryMutex     sync.RWMutex
	handlerByEndpoint = map[string]*_Registration{}
)

// Register adds handler and returns an idempotent cleanup bound to this registration.
func Register(endpoint string, handler http.Handler) func() {
	vpre.Check(IsEndpoint(endpoint), "inproc endpoint %s must start with %s", endpoint, EndpointScheme)
	vpre.Check(len(endpoint) > len(EndpointScheme), "inproc endpoint host is empty")
	vpre.CheckNotNil(handler, "inproc handler cannot be nil")

	registration := &_Registration{handler: handler}
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

func RoundTrip(endpoint string, req *http.Request) (*http.Response, error) {
	handler, ok := getHandler(endpoint)
	if !ok {
		return nil, errors.New("handler is not registered for endpoint " + endpoint)
	}

	return httputil.InprocRoundTrip(handler, req)
}

func ServeUpgrade(endpoint string, w http.ResponseWriter, req *http.Request) error {
	handler, ok := getHandler(endpoint)
	if !ok {
		return errors.New("handler is not registered for endpoint " + endpoint)
	}

	handler.ServeHTTP(w, req)
	return nil
}

func getHandler(endpoint string) (http.Handler, bool) {
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

// Endpoint builds a web+inproc endpoint from a scheme-less host path and route paths.
func Endpoint(hostPath string, paths ...string) string {
	endpoint := EndpointScheme + hostPath
	for _, path := range paths {
		endpoint += path
	}
	return endpoint
}
