package core

import (
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
)

const (
	DashboardAdminApiRuleName = "vine.hub.admin-api"
	DashboardWebRuleName      = "vine.hub.dashboard-web"
)

// Structs

type PortalRule struct {
	Id                 int
	Name               string
	Scheme             string
	Host               string
	Port               int
	PathPrefix         string
	TargetType         string
	SiteName           string
	RedirectionPattern string
	BuiltIn            bool
}

type PortalRuleCreation struct {
	Name               string
	Scheme             string
	Host               string
	Port               int
	PathPrefix         string
	TargetType         string
	SiteName           string
	RedirectionPattern string
}

type PortalRuleUpdate struct {
	Name               *string
	Scheme             *string
	Host               *string
	Port               *int
	PathPrefix         *string
	TargetType         *string
	SiteName           *string
	RedirectionPattern *string
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
		Name:               creation.Name,
		Scheme:             creation.Scheme,
		Host:               creation.Host,
		Port:               creation.Port,
		PathPrefix:         creation.PathPrefix,
		TargetType:         creation.TargetType,
		SiteName:           creation.SiteName,
		RedirectionPattern: creation.RedirectionPattern,
	}
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
	if update.Scheme != nil {
		next.Scheme = *update.Scheme
	}
	if update.Host != nil {
		next.Host = *update.Host
	}
	if update.Port != nil {
		next.Port = *update.Port
	}
	if update.PathPrefix != nil {
		next.PathPrefix = *update.PathPrefix
	}
	if update.TargetType != nil {
		next.TargetType = *update.TargetType
	}
	if update.SiteName != nil {
		next.SiteName = *update.SiteName
	}
	if update.RedirectionPattern != nil {
		next.RedirectionPattern = *update.RedirectionPattern
	}

	m.PortalRuleRepo.SaveRule(&next)
	return next
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

	adminRule.Scheme = scheme
	adminRule.Host = host
	adminRule.Port = port
	webRule.Scheme = scheme
	webRule.Host = host
	webRule.Port = port
	webRule.PathPrefix = pathPrefix

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
		Scheme:     adminRule.Scheme,
		Host:       adminRule.Host,
		Port:       adminRule.Port,
		PathPrefix: webRule.PathPrefix,
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
