package site

import (
	"net"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

const redirectionSiteName = "redirection"

type _RedirectionSite struct {
	isPermanent     bool
	redirectPattern string
}

func NewRedirectionSite(isPermanent bool, redirectPattern string) spec.Site {
	return &_RedirectionSite{
		isPermanent:     isPermanent,
		redirectPattern: redirectPattern,
	}
}

func (s *_RedirectionSite) Name() string {
	return redirectionSiteName
}

func (s *_RedirectionSite) Serve(ctx *spec.Context) {
	http.Redirect(ctx.ResponseWriter, ctx.Request, s.location(ctx), s.statusCode())
}

func (s *_RedirectionSite) Update(config redised.PortalSite) bool {
	return false
}

func (s *_RedirectionSite) Stop() {}

func (s *_RedirectionSite) location(ctx *spec.Context) string {
	return replaceRedirectionPlaceholders(s.redirectPattern, ctx)
}

func (s *_RedirectionSite) statusCode() int {
	// Use 307/308 so redirect keeps the original HTTP method and body.
	if s.isPermanent {
		return http.StatusPermanentRedirect
	}
	return http.StatusTemporaryRedirect
}

func replaceRedirectionPlaceholders(pattern string, ctx *spec.Context) string {
	return replacePlaceholders(pattern, func(key string) (string, bool) {
		return redirectionPlaceholderValue(key, ctx)
	})
}

func replacePlaceholders(pattern string, value func(string) (string, bool)) string {
	var builder strings.Builder
	for len(pattern) > 0 {
		start := strings.IndexByte(pattern, '{')
		if start < 0 {
			builder.WriteString(pattern)
			break
		}
		end := strings.IndexByte(pattern[start+1:], '}')
		if end < 0 {
			builder.WriteString(pattern)
			break
		}
		end += start + 1

		builder.WriteString(pattern[:start])
		key := pattern[start+1 : end]
		if replacement, ok := value(key); ok {
			builder.WriteString(replacement)
		} else {
			builder.WriteString(pattern[start : end+1])
		}
		pattern = pattern[end+1:]
	}
	return builder.String()
}

func redirectionPlaceholderValue(key string, ctx *spec.Context) (string, bool) {
	r := ctx.Request
	switch key {
	case "scheme":
		return r.URL.Scheme, true
	case "host":
		return requestHost(r), true
	case "uri":
		return r.URL.RequestURI(), true
	case "path":
		return r.URL.Path, true
	case "query":
		return r.URL.RawQuery, true
	case "method":
		return r.Method, true
	case "remote":
		return ctx.RemoteAddr, true
	default:
		return "", false
	}
}

func requestHost(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.Host)
	if err == nil {
		return host
	}
	return request.Host
}
