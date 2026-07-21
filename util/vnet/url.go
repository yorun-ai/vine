package vnet

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"go.yorun.ai/vine/util/vpre"
)

// HttpURL is a validated HTTP or HTTPS URL with a required numeric port.
type HttpURL struct {
	*url.URL
	port int
}

// Port returns the parsed numeric port.
func (u *HttpURL) Port() int {
	return u.port
}

// JoinPathPrefix joins pathPrefix to the URL path while preserving an explicit trailing slash.
func (u *HttpURL) JoinPathPrefix(pathPrefix string) string {
	joined := path.Join(u.EscapedPath(), pathPrefix)
	// Keep an explicit trailing slash in pathPrefix after path.Join cleans it.
	if strings.HasSuffix(pathPrefix, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	return joined
}

const defaultScheme = "http"

// ParseHttpURL parses an HTTP or HTTPS URL with a required port.
// A missing scheme defaults to http, and an empty path is normalized to "/".
func ParseHttpURL(raw string) (*HttpURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("url is empty")
	}

	if !strings.Contains(raw, "://") {
		raw = defaultScheme + "://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	// check scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http or https")
	}

	// check port
	portText := parsed.Port()
	if portText == "" {
		return nil, fmt.Errorf("url port is required")
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, err
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("url port must be between 0 and 65535")
	}

	// normalize path
	if parsed.EscapedPath() == "" {
		parsed.Path = "/"
	}

	return &HttpURL{
		URL:  parsed,
		port: port,
	}, nil
}

// MustParseHttpURL is like ParseHttpURL but panics on failure.
func MustParseHttpURL(rawURL string) *HttpURL {
	parsed, err := ParseHttpURL(rawURL)
	vpre.CheckNilError(err, "parse url failed")
	return parsed
}
