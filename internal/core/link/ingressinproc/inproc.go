package ingressinproc

import (
	"errors"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/util/vpre"
)

const EndpointScheme = "link+inproc://"

// handlerByEndpoint is the single process-wide inproc registry. It is mutated
// only during app startup/shutdown and is read-only while serving requests.
var handlerByEndpoint = map[string]http.Handler{}

func Register(endpoint string, handler http.Handler) {
	vpre.Check(IsEndpoint(endpoint), "link ingress inproc endpoint %s must start with %s", endpoint, EndpointScheme)
	vpre.Check(len(endpoint) > len(EndpointScheme), "link ingress inproc endpoint host is empty")
	vpre.CheckNotNil(handler, "link ingress inproc handler cannot be nil")
	vpre.CheckNil(handlerByEndpoint[endpoint], "link ingress inproc endpoint %s already registered", endpoint)

	handlerByEndpoint[endpoint] = handler
}

func Unregister(endpoint string) {
	delete(handlerByEndpoint, endpoint)
}

func RoundTrip(endpoint string, req *http.Request) (*http.Response, error) {
	endpointPath, _, _ := strings.Cut(endpoint, "?")
	handler, suffix, ok := getHandler(endpointPath)
	if !ok {
		return nil, errors.New("handler is not registered for link ingress inproc endpoint " + endpoint)
	}

	next := req.Clone(req.Context())
	next.URL.Path = suffix
	next.RequestURI = suffix
	if next.URL.RawQuery != "" {
		next.RequestURI += "?" + next.URL.RawQuery
	}

	return httputil.InprocRoundTrip(handler, next)
}

func ServeUpgrade(endpoint string, w http.ResponseWriter, req *http.Request) error {
	endpointPath, _, _ := strings.Cut(endpoint, "?")
	handler, suffix, ok := getHandler(endpointPath)
	if !ok {
		return errors.New("handler is not registered for link ingress inproc endpoint " + endpoint)
	}

	next := req.Clone(req.Context())
	next.URL.Path = suffix
	next.RequestURI = suffix
	if next.URL.RawQuery != "" {
		next.RequestURI += "?" + next.URL.RawQuery
	}

	handler.ServeHTTP(w, next)
	return nil
}

func getHandler(endpoint string) (http.Handler, string, bool) {
	matchedEndpoint := ""
	var matchedHandler http.Handler
	for registeredEndpoint, handler := range handlerByEndpoint {
		if !matchEndpoint(registeredEndpoint, endpoint) {
			continue
		}
		if len(registeredEndpoint) > len(matchedEndpoint) {
			matchedEndpoint = registeredEndpoint
			matchedHandler = handler
		}
	}
	if matchedHandler == nil {
		return nil, "", false
	}

	suffix := strings.TrimPrefix(endpoint, matchedEndpoint)
	if suffix == "" {
		suffix = "/"
	}
	return matchedHandler, suffix, true
}

func matchEndpoint(registeredEndpoint string, endpoint string) bool {
	if endpoint == registeredEndpoint {
		return true
	}
	return strings.HasPrefix(endpoint, registeredEndpoint+"/")
}

func IsEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, EndpointScheme)
}

func Endpoint(hostPath string) string {
	return EndpointScheme + hostPath
}
