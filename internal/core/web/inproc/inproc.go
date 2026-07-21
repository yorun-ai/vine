package inproc

import (
	"errors"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/util/vpre"
)

const EndpointScheme = "web+inproc://"

// handlerByEndpoint is the single process-wide inproc registry. It is mutated
// only during app startup/shutdown and is read-only while serving requests.
var handlerByEndpoint = map[string]http.Handler{}

func Register(endpoint string, handler http.Handler) {
	vpre.Check(IsEndpoint(endpoint), "inproc endpoint %s must start with %s", endpoint, EndpointScheme)
	vpre.Check(len(endpoint) > len(EndpointScheme), "inproc endpoint host is empty")
	vpre.CheckNotNil(handler, "inproc handler cannot be nil")
	vpre.CheckNil(handlerByEndpoint[endpoint], "inproc endpoint %s already registered", endpoint)
	handlerByEndpoint[endpoint] = handler
}

func Unregister(endpoint string) {
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
	handler := handlerByEndpoint[endpoint]
	return handler, handler != nil
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
