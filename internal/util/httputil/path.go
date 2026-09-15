package httputil

import (
	"net/http"
	"path"
	"strings"
)

// JoinPath joins slash-separated path fragments using path.Join.
// It cleans dot segments, repeated slashes, and trailing slashes except at root.
func JoinPath(parts ...string) string {
	return path.Join(parts...)
}

func PathPrefix(path string) string {
	prefix, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return "/" + prefix
}

func StripPathPrefix(r *http.Request, prefix string) *http.Request {
	nextPath := strings.TrimPrefix(r.URL.Path, prefix)
	if nextPath == "" {
		nextPath = "/"
	}
	if !strings.HasPrefix(nextPath, "/") {
		nextPath = "/" + nextPath
	}
	next := r.Clone(r.Context())
	next.URL.Path = nextPath
	next.URL.RawPath = ""
	return next
}
