package httputil

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"
)

// ValidatePathPrefix validates an absolute, URL-escaped path prefix without
// query, fragment, host, scheme, whitespace, control characters, or dot
// segments. Empty is accepted as an unset prefix.
func ValidatePathPrefix(value string) error {
	if value == "" {
		return nil
	}
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("must be a valid absolute path: %w", err)
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "?#") || u.Scheme != "" || u.Host != "" {
		return fmt.Errorf("must be a path without scheme, host, query or fragment")
	}
	if strings.Contains(value, "\\") || strings.IndexFunc(u.Path, unicode.IsControl) >= 0 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return fmt.Errorf("contains unsupported characters")
	}
	for segment := range strings.SplitSeq(u.Path, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("must not contain dot segments")
		}
	}
	return nil
}

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
