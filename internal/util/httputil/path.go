package httputil

import (
	"net/http"
	"strings"
)

func PathPrefix(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	return "/" + parts[0]
}

func TrimPathPrefix(r *http.Request, prefix string) *http.Request {
	nextPath := strings.TrimPrefix(r.URL.Path, prefix)
	if nextPath == "" {
		nextPath = "/"
	}
	if !strings.HasPrefix(nextPath, "/") {
		nextPath = "/" + nextPath
	}
	next := r.Clone(r.Context())
	url := *next.URL
	url.Path = nextPath
	url.RawPath = ""
	next.URL = &url
	return next
}
