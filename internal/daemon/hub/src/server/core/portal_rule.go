package core

import (
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
	rule.RoutePathPrefix = NormalizePortalRuleRoutePathPrefix(rule.RouteType, rule.RoutePathPrefix)
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

	next.RoutePathPrefix = NormalizePortalRuleRoutePathPrefix(next.RouteType, next.RoutePathPrefix)
	m.PortalRuleRepo.SaveRule(&next)
	return next
}

// NormalizePortalRuleRoutePathPrefix validates a site-relative escaped path prefix.
// Empty and root prefixes preserve the legacy prefix-stripping behavior.
func NormalizePortalRuleRoutePathPrefix(routeType, routePathPrefix string) string {
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
