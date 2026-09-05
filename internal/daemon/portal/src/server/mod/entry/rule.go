package entry

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vpre"
)

const (
	routeTypeSite              = "SITE"
	routeTypePermanentRedirect = "PERMANENT_REDIRECT"
	routeTypeTemporaryRedirect = "TEMPORARY_REDIRECT"
)

type _Key struct {
	scheme spec.Scheme
	port   int
}

type _Rule struct {
	name            string
	matchScheme     spec.Scheme
	matchHost       string
	matchPort       int
	matchPathPrefix string
	routePathPrefix string

	siteManager     *site.Manager
	redirectionSite spec.Site
	routeSiteName   string
}

func newRule(rule redised.PortalRule, siteManager *site.Manager) (*_Rule, bool) {
	isRedirection := rule.RouteType == routeTypePermanentRedirect || rule.RouteType == routeTypeTemporaryRedirect
	isEntry := rule.RouteType == routeTypeSite
	if !isRedirection && !isEntry {
		entryLogger.Warn("vine.portal entry rule target type is not supported", "rule", rule.Name, "routeType", rule.RouteType)
		return nil, false
	}

	scheme := spec.Scheme(rule.MatchScheme)
	entryRule := &_Rule{
		name:            rule.Name,
		matchScheme:     scheme,
		matchHost:       entryRuleHost(rule.MatchHost),
		matchPort:       entryRulePort(scheme, rule.MatchPort),
		matchPathPrefix: rule.MatchPathPrefix,
		routePathPrefix: strings.TrimRight(rule.RoutePathPrefix, "/"),
	}

	if isRedirection {
		entryRule.redirectionSite = site.NewRedirectionSite(rule.RouteType == routeTypePermanentRedirect, rule.RouteRedirectionPattern)
		return entryRule, true
	}

	entryRule.siteManager = siteManager
	entryRule.routeSiteName = rule.RouteSiteName
	return entryRule, true
}

func entryRulePort(scheme spec.Scheme, port int) int {
	switch scheme {
	case spec.SchemeHTTP:
		if port == 0 {
			port = defaultHTTPEntryPort
		}
	case spec.SchemeHTTPS:
		if port == 0 {
			port = defaultHTTPSEntryPort
		}
	default:
		vpre.Panicf("unknown port scheme: %s", string(scheme))
	}

	vpre.Check(port != 0, "unsupported entry scheme: %s", string(scheme))
	return port
}

func entryRuleHost(host string) string {
	return host
}

func (r _Rule) Key() _Key {
	return _Key{
		scheme: r.matchScheme,
		port:   r.matchPort,
	}
}

func (r _Rule) Matches(request *http.Request) bool {
	if !r.matchesHost(requestHost(request)) {
		return false
	}
	return r.matchesPathPrefix(request.URL.Path)
}

func (r _Rule) matchesHost(host string) bool {
	if r.matchHost == "" {
		return true
	}
	return r.matchHost == host
}

func (r _Rule) matchesPathPrefix(path string) bool {
	if r.matchPathPrefix == "" || r.matchPathPrefix == "/" {
		return true
	}
	if path == r.matchPathPrefix {
		return true
	}
	return strings.HasPrefix(path, r.matchPathPrefix+"/")
}

func (r _Rule) Serve(ctx *spec.Context) {
	if r.redirectionSite != nil {
		r.redirectionSite.Serve(ctx)
		return
	}

	if targetSite, ok := r.siteManager.Site(r.routeSiteName); ok {
		ctx.Request = r.rewritePath(ctx.Request)
		ctx.EntryOrigin = spec.EntryOrigin{
			Scheme: r.matchScheme,
			Host:   r.matchHost,
			Port:   r.matchPort,
		}
		targetSite.Serve(ctx)
		return
	}

	http.Error(ctx.ResponseWriter, "portal entry is not found: "+r.routeSiteName, http.StatusServiceUnavailable)
}

func (r _Rule) rewritePath(request *http.Request) *http.Request {
	if r.routePathPrefix == "" && (r.matchPathPrefix == "" || r.matchPathPrefix == "/") {
		return request
	}
	suffix := request.URL.EscapedPath()
	if r.matchPathPrefix != "" && r.matchPathPrefix != "/" {
		// Match uses the decoded path. Consume the same number of decoded bytes
		// without decoding the suffix, so escaped slashes retain their meaning.
		offset := 0
		for range len(r.matchPathPrefix) {
			if suffix[offset] == '%' {
				offset += 3
			} else {
				offset++
			}
		}
		suffix = suffix[offset:]
	}
	nextEscapedPath := r.routePathPrefix + suffix
	if nextEscapedPath == "" {
		nextEscapedPath = "/"
	}
	nextPath, err := url.PathUnescape(nextEscapedPath)
	vpre.CheckNilError(err, "decode portal target path")
	next := request.Clone(request.Context())
	next.URL.Path = nextPath
	next.URL.RawPath = nextEscapedPath
	return next
}

func requestHost(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.Host)
	if err == nil {
		return host
	}
	return request.Host
}
