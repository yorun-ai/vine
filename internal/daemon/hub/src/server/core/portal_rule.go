package core

import (
	"net"
	"net/url"
	"strings"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
)

const (
	DashboardAdminApiRuleName = "vine.hub.admin-api"
	DashboardWebRuleName      = "vine.hub.dashboard-web"
)

// Structs

type PortalRule struct {
	Id                      int
	Name                    string
	MatchScheme             string
	MatchHost               string
	MatchPort               int
	MatchPathPrefix         string
	RoutePathPrefix         string
	RouteType               string
	RouteSiteName           string
	RouteRedirectionPattern string
	BuiltIn                 bool
}

type PortalRuleCreation struct {
	Name                    string
	MatchScheme             string
	MatchHost               string
	MatchPort               int
	MatchPathPrefix         string
	RoutePathPrefix         string
	RouteType               string
	RouteSiteName           string
	RouteRedirectionPattern string
}

type PortalRuleUpdate struct {
	Name                    *string
	MatchScheme             *string
	MatchHost               *string
	MatchPort               *int
	MatchPathPrefix         *string
	RoutePathPrefix         *string
	RouteType               *string
	RouteSiteName           *string
	RouteRedirectionPattern *string
}

type PortalDashboardAccess struct {
	Scheme     string
	Host       string
	Port       int
	PathPrefix string
}

// Repo

type PortalRuleRepo interface {
	ListRules() []PortalRule
	GetRuleById(id int) (*PortalRule, bool)
	GetRuleByName(name string) (*PortalRule, bool)
	SaveRule(rule *PortalRule)
	RemoveRule(id int) bool
}

// Core

type PortalRuleCore struct {
	PortalRuleRepo PortalRuleRepo `inject:""`
	PortalCertRepo PortalCertRepo `inject:""`
}

func (m *PortalRuleCore) List() []PortalRule {
	rules := m.PortalRuleRepo.ListRules()
	ret := make([]PortalRule, 0, len(rules))
	for i := range rules {
		if rules[i].BuiltIn {
			continue
		}
		ret = append(ret, rules[i])
	}
	return ret
}

func (m *PortalRuleCore) Get(id int) PortalRule {
	rule, ok := m.PortalRuleRepo.GetRuleById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	return *rule
}

func (m *PortalRuleCore) Create(creation PortalRuleCreation) PortalRule {
	_, ok := m.PortalRuleRepo.GetRuleByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("entry rule %q already exists", creation.Name))

	rule := PortalRule{
		Name:                    creation.Name,
		MatchScheme:             creation.MatchScheme,
		MatchHost:               creation.MatchHost,
		MatchPort:               creation.MatchPort,
		MatchPathPrefix:         creation.MatchPathPrefix,
		RoutePathPrefix:         creation.RoutePathPrefix,
		RouteType:               creation.RouteType,
		RouteSiteName:           creation.RouteSiteName,
		RouteRedirectionPattern: creation.RouteRedirectionPattern,
	}
	rule = m.Validate(rule)
	m.PortalRuleRepo.SaveRule(&rule)
	return rule
}

func (m *PortalRuleCore) Update(id int, update PortalRuleUpdate) PortalRule {
	rule, ok := m.PortalRuleRepo.GetRuleById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	ex.PanicNewIfNot(!rule.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be updated", rule.Name))

	next := *rule
	if update.Name != nil {
		if *update.Name != rule.Name {
			_, exists := m.PortalRuleRepo.GetRuleByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("entry rule %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.MatchScheme != nil {
		next.MatchScheme = *update.MatchScheme
	}
	if update.MatchHost != nil {
		next.MatchHost = *update.MatchHost
	}
	if update.MatchPort != nil {
		next.MatchPort = *update.MatchPort
	}
	if update.MatchPathPrefix != nil {
		next.MatchPathPrefix = *update.MatchPathPrefix
	}
	if update.RoutePathPrefix != nil {
		next.RoutePathPrefix = *update.RoutePathPrefix
	}
	if update.RouteType != nil {
		next.RouteType = *update.RouteType
	}
	if update.RouteSiteName != nil {
		next.RouteSiteName = *update.RouteSiteName
	}
	if update.RouteRedirectionPattern != nil {
		next.RouteRedirectionPattern = *update.RouteRedirectionPattern
	}

	next = m.Validate(next)
	m.PortalRuleRepo.SaveRule(&next)
	return next
}

// normalizePortalRuleRoutePathPrefix validates a site-relative escaped path prefix.
// Empty and root prefixes preserve the legacy prefix-stripping behavior.
func normalizePortalRuleRoutePathPrefix(routeType, routePathPrefix string) string {
	if routePathPrefix == "" {
		return ""
	}
	ex.PanicNewIfNot(routeType == PortalRuleRouteTypeSite, ex.OperationFailed, "routePathPrefix is only supported for SITE rules")
	u, err := url.ParseRequestURI(routePathPrefix)
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, "routePathPrefix must be a valid absolute path")
	ex.PanicNewIfNot(strings.HasPrefix(routePathPrefix, "/") && !strings.HasPrefix(routePathPrefix, "//") && !strings.ContainsAny(routePathPrefix, "?#") && u.Scheme == "" && u.Host == "", ex.OperationFailed, "routePathPrefix must be a path without scheme, host, query or fragment")
	ex.PanicNewIfNot(!strings.Contains(u.Path, "\\") && strings.IndexFunc(u.Path, unicode.IsControl) < 0 && strings.IndexFunc(routePathPrefix, unicode.IsSpace) < 0, ex.OperationFailed, "routePathPrefix contains unsupported characters")
	for segment := range strings.SplitSeq(u.Path, "/") {
		ex.PanicNewIfNot(segment != "." && segment != "..", ex.OperationFailed, "routePathPrefix must not contain dot segments")
	}
	return strings.TrimRight(u.EscapedPath(), "/")
}

func (m *PortalRuleCore) Remove(id int) {
	rule, ok := m.PortalRuleRepo.GetRuleById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	ex.PanicNewIfNot(!rule.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be removed", rule.Name))

	ok = m.PortalRuleRepo.RemoveRule(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
}

func (m *PortalRuleCore) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) []PortalRule {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	host = strings.TrimSpace(host)
	pathPrefix = normalizeDashboardPathPrefix(pathPrefix)

	ex.PanicNewIfNot(scheme == "http" || scheme == "https", ex.OperationFailed, "dashboard scheme must be http or https")
	ex.PanicNewIfNot(port >= 0 && port <= 65535, ex.OperationFailed, "dashboard port must be between 0 and 65535")
	if scheme == "https" {
		ex.PanicNewIfNot(host != "", ex.OperationFailed, "dashboard https host is required")
		ex.PanicNewIfNot(m.hasConfiguredCertForHost(host), ex.OperationFailed, ex.F("dashboard https host %q has no configured certificate", host))
	}

	adminRule := m.dashboardRule(DashboardAdminApiRuleName)
	webRule := m.dashboardRule(DashboardWebRuleName)

	adminRule.MatchScheme = scheme
	adminRule.MatchHost = host
	adminRule.MatchPort = port
	webRule.MatchScheme = scheme
	webRule.MatchHost = host
	webRule.MatchPort = port
	webRule.MatchPathPrefix = pathPrefix

	adminRule.normalizeAndValidate()
	webRule.normalizeAndValidate()
	m.PortalRuleRepo.SaveRule(adminRule)
	m.PortalRuleRepo.SaveRule(webRule)

	return []PortalRule{
		*adminRule,
		*webRule,
	}
}

func (m *PortalRuleCore) DashboardAccess() PortalDashboardAccess {
	adminRule := m.dashboardRule(DashboardAdminApiRuleName)
	webRule := m.dashboardRule(DashboardWebRuleName)
	return PortalDashboardAccess{
		Scheme:     adminRule.MatchScheme,
		Host:       adminRule.MatchHost,
		Port:       adminRule.MatchPort,
		PathPrefix: webRule.MatchPathPrefix,
	}
}

func normalizeDashboardPathPrefix(pathPrefix string) string {
	trimmed := strings.TrimSpace(pathPrefix)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func (m *PortalRuleCore) hasConfiguredCertForHost(host string) bool {
	if m.PortalCertRepo == nil {
		return false
	}
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	for _, cert := range m.PortalCertRepo.ListCerts() {
		if cert == nil || cert.PrivateKeyBase64 == "" {
			continue
		}
		for _, domain := range cert.Domains {
			if portalCertDomainMatchesHost(domain, normalizedHost) {
				return true
			}
		}
	}
	return false
}

func portalCertDomainMatchesHost(domain string, host string) bool {
	normalizedDomain := strings.ToLower(strings.TrimSpace(domain))
	if normalizedDomain == "" || host == "" {
		return false
	}
	if normalizedDomain == host {
		return true
	}
	if strings.HasPrefix(normalizedDomain, "*.") {
		suffix := strings.TrimPrefix(normalizedDomain, "*")
		return strings.HasSuffix(host, suffix) && strings.Count(host, ".") == strings.Count(normalizedDomain, ".")
	}
	return false
}

func (m *PortalRuleCore) dashboardRule(name string) *PortalRule {
	rule, ok := m.PortalRuleRepo.GetRuleByName(name)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("dashboard entry rule %q not found", name))
	ex.PanicNewIfNot(rule.BuiltIn, ex.OperationFailed, ex.F("dashboard entry rule %q is not a built-in rule", name))
	return rule
}

// normalizeAndValidate enforces the constraints of a complete rule before persistence.
func (r *PortalRule) normalizeAndValidate() {
	fail := func(ok bool, message string) {
		ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal rule %q: %s", r.Name, message))
	}
	fail(strings.TrimSpace(r.Name) != "", "name is required")
	fail(r.MatchScheme == "http" || r.MatchScheme == "https", "matchScheme must be http or https")
	fail(r.MatchPort >= 0 && r.MatchPort <= 65535, "matchPort must be between 0 and 65535")
	if r.MatchHost != "" {
		fail(!strings.ContainsAny(r.MatchHost, "/?#@*\\") && strings.IndexFunc(r.MatchHost, unicode.IsSpace) < 0 && strings.IndexFunc(r.MatchHost, unicode.IsControl) < 0, "matchHost must be a hostname or IP without a port")
		if net.ParseIP(r.MatchHost) == nil {
			host, err := url.Parse("//" + r.MatchHost)
			fail(err == nil && host.Hostname() == r.MatchHost && !strings.ContainsAny(r.MatchHost, ":[]"), "matchHost must be a hostname or IP without a port")
		}
	}
	if r.MatchPathPrefix != "" {
		fail(strings.HasPrefix(r.MatchPathPrefix, "/") && !strings.ContainsAny(r.MatchPathPrefix, "?#\\") && strings.IndexFunc(r.MatchPathPrefix, unicode.IsSpace) < 0 && strings.IndexFunc(r.MatchPathPrefix, unicode.IsControl) < 0, "matchPathPrefix must be an absolute path without query or fragment")
		for part := range strings.SplitSeq(r.MatchPathPrefix, "/") {
			fail(part != "." && part != "..", "matchPathPrefix must not contain dot segments")
		}
	}
	switch r.RouteType {
	case PortalRuleRouteTypeSite:
		fail(strings.TrimSpace(r.RouteSiteName) != "", "routeSiteName is required for SITE rules")
		fail(r.RouteRedirectionPattern == "", "routeRedirectionPattern is not supported for SITE rules")
	case PortalRuleRouteTypePermanentRedirect, PortalRuleRouteTypeTemporaryRedirect:
		fail(r.RouteSiteName == "", "routeSiteName is only supported for SITE rules")
		fail(strings.TrimSpace(r.RouteRedirectionPattern) != "", "routeRedirectionPattern is required for redirect rules")
		fail(strings.IndexFunc(r.RouteRedirectionPattern, unicode.IsControl) < 0, "routeRedirectionPattern contains control characters")
		pattern := r.RouteRedirectionPattern
		for len(pattern) > 0 {
			start := strings.IndexAny(pattern, "{}")
			if start < 0 {
				break
			}
			fail(pattern[start] == '{', "routeRedirectionPattern contains an unmatched brace")
			end := strings.IndexByte(pattern[start+1:], '}')
			fail(end >= 0, "routeRedirectionPattern contains an unmatched brace")
			end += start + 1
			switch pattern[start+1 : end] {
			case "scheme", "host", "uri", "path", "query", "method", "remote":
			default:
				fail(false, "routeRedirectionPattern contains an unsupported placeholder")
			}
			pattern = pattern[end+1:]
		}
	default:
		fail(false, "routeType must be SITE, PERMANENT_REDIRECT or TEMPORARY_REDIRECT")
	}
	r.RoutePathPrefix = normalizePortalRuleRoutePathPrefix(r.RouteType, r.RoutePathPrefix)
}

// Validate checks and normalizes a complete user rule without accessing storage.
// Reserved built-in names are protected independently of database contents.
func (*PortalRuleCore) Validate(rule PortalRule) PortalRule {
	ex.PanicNewIfNot(rule.Name != DashboardAdminApiRuleName && rule.Name != DashboardWebRuleName,
		ex.OperationFailed, ex.F("built-in entry rule %q cannot be replaced", rule.Name))
	rule.normalizeAndValidate()
	return rule
}

// Save creates or replaces a complete user rule by name, preserving an existing ID.
func (m *PortalRuleCore) Save(rule PortalRule) PortalRule {
	rule = m.Validate(rule)
	rule.Id = 0
	rule.BuiltIn = false
	if current, ok := m.PortalRuleRepo.GetRuleByName(rule.Name); ok {
		ex.PanicNewIfNot(!current.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be replaced", rule.Name))
		rule.Id = current.Id
	}
	m.PortalRuleRepo.SaveRule(&rule)
	return rule
}

// EnsureDashboardRule provisions a built-in rule, preserving configured access
// unless an explicit access update or legacy migration requires a refresh.
func (m *PortalRuleCore) EnsureDashboardRule(rule PortalRule, refreshAccess bool) {
	ex.PanicNewIfNot(rule.Name == DashboardAdminApiRuleName || rule.Name == DashboardWebRuleName, ex.OperationFailed, "not a dashboard rule")
	rule.BuiltIn = true
	if old, ok := m.PortalRuleRepo.GetRuleByName(rule.Name); ok {
		rule.Id = old.Id
		if !refreshAccess {
			rule.MatchScheme = old.MatchScheme
			rule.MatchHost = old.MatchHost
			rule.MatchPort = old.MatchPort
			rule.MatchPathPrefix = old.MatchPathPrefix
		}
	}
	rule.normalizeAndValidate()
	m.PortalRuleRepo.SaveRule(&rule)
}
