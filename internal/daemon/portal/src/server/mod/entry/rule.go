package entry

import (
	"net"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vpre"
)

const (
	targetTypeSite              = "SITE"
	targetTypePermanentRedirect = "PERMANENT_REDIRECT"
	targetTypeTemporaryRedirect = "TEMPORARY_REDIRECT"
)

type _Key struct {
	scheme spec.Scheme
	port   int
}

type _Rule struct {
	name       string
	scheme     spec.Scheme
	host       string
	port       int
	pathPrefix string

	siteManager     *site.Manager
	redirectionSite spec.Site
	targetSiteName  string
}

func newRule(rule redised.PortalRule, siteManager *site.Manager) (*_Rule, bool) {
	isRedirection := rule.TargetType == targetTypePermanentRedirect || rule.TargetType == targetTypeTemporaryRedirect
	isEntry := rule.TargetType == targetTypeSite
	if !isRedirection && !isEntry {
		logger.Warn("vine.portal entry rule target type is not supported", "rule", rule.Name, "targetType", rule.TargetType)
		return nil, false
	}

	scheme := spec.Scheme(rule.Scheme)
	entryRule := &_Rule{
		name:       rule.Name,
		scheme:     scheme,
		host:       entryRuleHost(rule.Host),
		port:       entryRulePort(scheme, rule.Port),
		pathPrefix: rule.PathPrefix,
	}

	if isRedirection {
		entryRule.redirectionSite = site.NewRedirectionSite(rule.TargetType == targetTypePermanentRedirect, rule.RedirectionPattern)
		return entryRule, true
	}

	entryRule.siteManager = siteManager
	entryRule.targetSiteName = rule.SiteName
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
		scheme: r.scheme,
		port:   r.port,
	}
}

func (r _Rule) Matches(request *http.Request) bool {
	if !r.matchesHost(requestHost(request)) {
		return false
	}
	return r.matchesPathPrefix(request.URL.Path)
}

func (r _Rule) matchesHost(host string) bool {
	if r.host == "" {
		return true
	}
	return r.host == host
}

func (r _Rule) matchesPathPrefix(path string) bool {
	if r.pathPrefix == "" || r.pathPrefix == "/" {
		return true
	}
	if path == r.pathPrefix {
		return true
	}
	return strings.HasPrefix(path, r.pathPrefix+"/")
}

func (r _Rule) Serve(ctx *spec.Context) {
	if r.redirectionSite != nil {
		r.redirectionSite.Serve(ctx)
		return
	}

	if targetSite, ok := r.siteManager.Site(r.targetSiteName); ok {
		ctx.Request = r.trimPathPrefix(ctx.Request)
		ctx.EntryOrigin = spec.EntryOrigin{
			Scheme: r.scheme,
			Host:   r.host,
			Port:   r.port,
		}
		targetSite.Serve(ctx)
		return
	}

	http.Error(ctx.ResponseWriter, "portal entry is not found: "+r.targetSiteName, http.StatusServiceUnavailable)
}

func (r _Rule) trimPathPrefix(request *http.Request) *http.Request {
	if r.pathPrefix == "" || r.pathPrefix == "/" {
		return request
	}
	nextPath := strings.TrimPrefix(request.URL.Path, r.pathPrefix)
	if nextPath == "" {
		nextPath = "/"
	}
	if !strings.HasPrefix(nextPath, "/") {
		nextPath = "/" + nextPath
	}
	next := request.Clone(request.Context())
	url := *next.URL
	url.Path = nextPath
	url.RawPath = ""
	next.URL = &url
	return next
}

func requestHost(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.Host)
	if err == nil {
		return host
	}
	return request.Host
}
