package core

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/util/httputil"
)

const (
	DashboardAdminApiRuleName = "vine.hub.admin-api"
	DashboardWebRuleName      = "vine.hub.dashboard-web"
)

// Structs

type PortalRule struct {
	FieldSources FieldSources
	Id           int
	Name         string
	// EntryId identifies the Portal entry that owns the access configuration.
	EntryId int
	// EntryName names the entry a caller wants the rule to join, such as a seed
	// that references a declared entry. Hub resolves the entry when it saves the
	// rule and clears the field, because the entry, not the rule, stores the
	// access. A stored rule leaves this field empty: its EntryId names the entry.
	EntryName string
	// MatchScheme, MatchHost, and MatchPort describe the entry the rule belongs
	// to. The entry stores them; Hub assembles these values when it reads a
	// rule, so a rule never owns access configuration of its own.
	MatchScheme             string
	MatchHost               string
	MatchPort               int
	MatchPathPrefix         string
	RouteType               string
	RouteSiteName           string
	RouteRedirectionPattern string
	RoutePathPrefix         string
	BuiltIn                 bool
	// Enabled decides whether Hub publishes the rule to Portal.
	Enabled bool
}

type PortalRuleCreation struct {
	Name string
	// EntryName names the Portal entry that owns the access the rule matches.
	EntryName               string
	MatchPathPrefix         string
	RouteType               string
	RouteSiteName           string
	RouteRedirectionPattern string
	RoutePathPrefix         string
	// Enabled is optional and defaults to true.
	Enabled *bool
}

type PortalRuleUpdate struct {
	Name                    *string
	MatchPathPrefix         *string
	RouteType               *string
	RouteSiteName           *string
	RouteRedirectionPattern *string
	RoutePathPrefix         *string
	Enabled                 *bool
}

type PortalDashboardAccess struct {
	Scheme     string
	Host       string
	Port       int
	PathPrefix string
}

// ResolvePortalRulePaths returns the effective prefixes sent to Portal.
func ResolvePortalRulePaths(rule *PortalRule, site *PortalSite) (string, string) {
	if site == nil || site.WebMountPath == "" || rule.RouteType != PortalRuleRouteTypeSite {
		return rule.MatchPathPrefix, rule.RoutePathPrefix
	}
	mountPath := strings.TrimRight(site.WebMountPath, "/")
	if mountPath == "" {
		return "/", ""
	}
	return mountPath, mountPath
}

// Repo

// PortalRuleRepo stores entry rules. List and the lookups return entities the
// caller owns.
type PortalRuleRepo interface {
	List() []*PortalRule
	GetById(id int) (*PortalRule, bool)
	GetByName(name string) (*PortalRule, bool)
	Save(rule *PortalRule)
	Remove(id int) bool
}

// Core

type PortalRuleCore struct {
	PortalRuleRepo  PortalRuleRepo   `inject:""`
	PortalCertRepo  PortalCertRepo   `inject:""`
	PortalEntryCore *PortalEntryCore `inject:""`
}

func (m *PortalRuleCore) List() []*PortalRule {
	rules := m.PortalRuleRepo.List()
	ret := make([]*PortalRule, 0, len(rules))
	for _, rule := range rules {
		if rule.BuiltIn {
			continue
		}
		ret = append(ret, rule)
	}
	return ret
}

// FindByName returns the rule with the name.
func (m *PortalRuleCore) FindByName(name string) (*PortalRule, bool) {
	return m.PortalRuleRepo.GetByName(name)
}

func (m *PortalRuleCore) Get(id int) *PortalRule {
	rule, ok := m.PortalRuleRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	return rule
}

func (m *PortalRuleCore) Create(creation PortalRuleCreation) *PortalRule {
	_, ok := m.PortalRuleRepo.GetByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("entry rule %q already exists", creation.Name))

	// The entry owns the access the rule matches, so a rule joins an entry Hub
	// already stores instead of declaring an access of its own.
	entry, ok := m.PortalEntryCore.FindByName(creation.EntryName)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", creation.EntryName))
	rule := PortalRule{
		Name:                    creation.Name,
		EntryId:                 entry.Id,
		MatchScheme:             entry.Scheme,
		MatchHost:               entry.Host,
		MatchPort:               entry.Port,
		MatchPathPrefix:         creation.MatchPathPrefix,
		RouteType:               creation.RouteType,
		RouteSiteName:           creation.RouteSiteName,
		RouteRedirectionPattern: creation.RouteRedirectionPattern,
		RoutePathPrefix:         creation.RoutePathPrefix,
		Enabled:                 EnabledOrDefault(creation.Enabled),
	}
	rule = m.Validate(rule)
	m.checkMatchesUnique(&rule)
	m.PortalRuleRepo.Save(&rule)
	return &rule
}

func (m *PortalRuleCore) Update(id int, update PortalRuleUpdate) *PortalRule {
	rule, ok := m.PortalRuleRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	ex.PanicNewIfNot(!rule.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be updated", rule.Name))

	next := *rule
	next.FieldSources = cloneFieldSources(rule.FieldSources)
	if update.Name != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/name")
		if *update.Name != rule.Name {
			_, exists := m.PortalRuleRepo.GetByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("entry rule %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.MatchPathPrefix != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/matchPathPrefix")
		next.MatchPathPrefix = *update.MatchPathPrefix
	}
	if update.RouteType != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/routeType")
		next.RouteType = *update.RouteType
	}
	if update.RouteSiteName != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/routeSiteName")
		next.RouteSiteName = *update.RouteSiteName
	}
	if update.RouteRedirectionPattern != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/routeRedirectionPattern")
		next.RouteRedirectionPattern = *update.RouteRedirectionPattern
	}
	if update.RoutePathPrefix != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/routePathPrefix")
		next.RoutePathPrefix = *update.RoutePathPrefix
	}
	if update.Enabled != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/enabled")
		next.Enabled = *update.Enabled
	}

	next = m.Validate(next)
	return m.saveToEntry(next)
}

// normalizePortalRuleRoutePathPrefix validates a site-relative escaped path prefix.
// Empty and root prefixes preserve the legacy prefix-stripping behavior.
func normalizePortalRuleRoutePathPrefix(routeType string, routePathPrefix string) string {
	if routePathPrefix == "" {
		return ""
	}
	ex.PanicNewIfNot(routeType == PortalRuleRouteTypeSite, ex.OperationFailed, "routePathPrefix is only supported for SITE rules")
	ex.PanicNewIfNot(httputil.ValidatePathPrefix(routePathPrefix) == nil, ex.OperationFailed, "routePathPrefix is invalid")
	u, err := url.ParseRequestURI(routePathPrefix)
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, "routePathPrefix must be a valid absolute path")
	return strings.TrimRight(u.EscapedPath(), "/")
}

func (m *PortalRuleCore) Remove(id int) {
	rule, ok := m.PortalRuleRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
	ex.PanicNewIfNot(!rule.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be removed", rule.Name))

	ok = m.PortalRuleRepo.Remove(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
}

func (m *PortalRuleCore) UpdateDashboardAccess(scheme string, host string, port int, pathPrefix string) []*PortalRule {
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

	access := normalizePortalEntry(PortalEntry{Scheme: scheme, Host: host, Port: port, BuiltIn: true})
	adminRule.MatchScheme = access.Scheme
	adminRule.MatchHost = access.Host
	adminRule.MatchPort = access.Port
	webRule.MatchScheme = access.Scheme
	webRule.MatchHost = access.Host
	webRule.MatchPort = access.Port
	webRule.MatchPathPrefix = pathPrefix

	// Check the new access before Hub stores it, so a rejected update leaves both
	// the entry and its rules untouched.
	adminRule.normalizeAndValidate()
	webRule.normalizeAndValidate()
	m.checkMatchesUnique(adminRule, webRule)

	entry := m.PortalEntryCore.EnsureBuiltInAccess(access.Scheme, access.Host, access.Port, true)
	adminRule.EntryId = entry.Id
	webRule.EntryId = entry.Id
	m.PortalRuleRepo.Save(adminRule)
	m.PortalRuleRepo.Save(webRule)

	return []*PortalRule{
		adminRule,
		webRule,
	}
}

func (m *PortalRuleCore) DashboardAccess() PortalDashboardAccess {
	adminRule := m.dashboardRule(DashboardAdminApiRuleName)
	webRule := m.dashboardRule(DashboardWebRuleName)
	entry := m.PortalEntryCore.Get(adminRule.EntryId)
	return PortalDashboardAccess{
		Scheme:     entry.Scheme,
		Host:       entry.Host,
		Port:       entry.Port,
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
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	for _, cert := range m.PortalCertRepo.List() {
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
	rule, ok := m.PortalRuleRepo.GetByName(name)
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
	if r.EntryName != "" {
		fail(r.MatchScheme == "" && r.MatchHost == "" && r.MatchPort == 0,
			"entryName cannot be mixed with matchScheme, matchHost, or matchPort")
	} else {
		fail(r.MatchScheme == "http" || r.MatchScheme == "https", "matchScheme must be http or https")
		fail(r.MatchPort >= 0 && r.MatchPort <= 65535, "matchPort must be between 0 and 65535")
	}
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
// Web mount paths override configured prefixes when Portal builds its routes;
// they are not copied into persisted rules or checked against stored sites here.
func (*PortalRuleCore) Validate(rule PortalRule) PortalRule {
	ex.PanicNewIfNot(rule.Name != DashboardAdminApiRuleName && rule.Name != DashboardWebRuleName,
		ex.OperationFailed, ex.F("built-in entry rule %q cannot be replaced", rule.Name))
	rule.normalizeAndValidate()
	return rule
}

// portalRuleMatchKey identifies the requests a rule matches: the access of its
// entry plus its match path prefix.
func portalRuleMatchKey(rule *PortalRule) string {
	return portalEntryMatchKey(portalRuleAccess(rule), rule.MatchPathPrefix)
}

// portalRuleAccess returns the access a rule resolves through its entry.
func portalRuleAccess(rule *PortalRule) PortalEntry {
	return PortalEntry{
		Scheme: rule.MatchScheme,
		Host:   rule.MatchHost,
		Port:   rule.MatchPort,
	}
}

// portalEntryMatchKey identifies the requests one access and path prefix match.
func portalEntryMatchKey(access PortalEntry, matchPathPrefix string) string {
	return access.Scheme + "\x00" + access.Host + "\x00" + strconv.Itoa(access.Port) + "\x00" + matchPathPrefix
}

// portalEntryMatchText renders the request an access and path prefix match for
// error messages.
func portalEntryMatchText(access PortalEntry, matchPathPrefix string) string {
	host := access.Host
	if host == "" {
		host = "*"
	}
	return fmt.Sprintf("%s://%s:%d%s", access.Scheme, host, access.Port, matchPathPrefix)
}

// portalRuleMatchConflict returns the stored rule that already matches the
// request the candidate matches at matchPathPrefix. A rule replaces the stored
// rule with the same id or name, so it never conflicts with itself.
func portalRuleMatchConflict(stored []*PortalRule, candidate PortalRule, matchPathPrefix string) *PortalRule {
	key := portalEntryMatchKey(portalRuleAccess(&candidate), matchPathPrefix)
	for _, rule := range stored {
		if rule.Name == candidate.Name || (rule.Id != 0 && rule.Id == candidate.Id) {
			continue
		}
		if portalRuleMatchKey(rule) == key {
			return rule
		}
	}
	return nil
}

// checkPortalRuleMatchesUnique rejects candidate rules that match the same
// request as a stored rule. A rule replaces the stored rule with the same id or
// name, so the rules of one update do not conflict with themselves.
func checkPortalRuleMatchesUnique(stored []*PortalRule, candidates ...*PortalRule) {
	for _, candidate := range candidates {
		conflict := portalRuleMatchConflict(stored, *candidate, candidate.MatchPathPrefix)
		if conflict != nil {
			ex.PanicNew(ex.OperationFailed, ex.F("portal rule %q already matches %s",
				conflict.Name, portalEntryMatchText(portalRuleAccess(candidate), candidate.MatchPathPrefix)))
		}
	}
}

// Save creates or replaces a complete user rule by name, preserving an existing
// ID. A rule that names an entry joins that entry, while a rule that declares an
// access joins the entry serving that access.
func (m *PortalRuleCore) Save(rule PortalRule) *PortalRule {
	if rule.EntryName != "" {
		// Validate the shape the caller declared before Hub resolves the entry, so
		// a rule that mixes an entry name with an access fails on that.
		rule = m.Validate(rule)
		entry, ok := m.PortalEntryCore.FindByName(rule.EntryName)
		ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", rule.EntryName))
		rule.EntryId = entry.Id
		rule.MatchScheme = entry.Scheme
		rule.MatchHost = entry.Host
		rule.MatchPort = entry.Port
		// The entry owns the access from here on.
		rule.EntryName = ""
	}
	rule = m.Validate(rule)
	rule.Id = 0
	rule.BuiltIn = false
	if current, ok := m.PortalRuleRepo.GetByName(rule.Name); ok {
		ex.PanicNewIfNot(!current.BuiltIn, ex.OperationFailed, ex.F("built-in entry rule %q cannot be replaced", rule.Name))
		rule.Id = current.Id
	}
	return m.saveToEntry(rule)
}

// saveToEntry stores a validated rule under the entry that serves its access,
// creating the entry when no rule has used that access yet.
func (m *PortalRuleCore) saveToEntry(rule PortalRule) *PortalRule {
	entry := m.PortalEntryCore.EnsureAccess(rule.MatchScheme, rule.MatchHost, rule.MatchPort)
	rule.EntryId = entry.Id
	rule.MatchScheme = entry.Scheme
	rule.MatchHost = entry.Host
	rule.MatchPort = entry.Port
	m.checkMatchesUnique(&rule)
	m.PortalRuleRepo.Save(&rule)
	return &rule
}

// checkMatchesUnique rejects a rule that matches the same request as a stored
// rule, including a built-in Dashboard rule. Portal resolves matching rules by
// their longest path prefix, so two rules that match identically have no defined
// order. Only the access migration separates such rules on its own, because an
// upgraded database cannot be corrected by editing stored data.
func (m *PortalRuleCore) checkMatchesUnique(rules ...*PortalRule) {
	checkPortalRuleMatchesUnique(m.PortalRuleRepo.List(), rules...)
}

// EnsureDashboardRule provisions a built-in rule under Hub's own entry. The rule
// names that entry and never declares an access: the built-in entry owns the
// access, which Seeder ensures from the Dashboard URL. Without refresh the rule
// keeps the path prefix it already stores.
func (m *PortalRuleCore) EnsureDashboardRule(rule PortalRule, refreshAccess bool) {
	ex.PanicNewIfNot(rule.Name == DashboardAdminApiRuleName || rule.Name == DashboardWebRuleName, ex.OperationFailed, "not a dashboard rule")
	ex.PanicNewIfNot(rule.EntryName == PortalEntryBuiltInName, ex.OperationFailed,
		ex.F("dashboard rule %q must name the built-in portal entry", rule.Name))
	rule.BuiltIn = true
	// Hub's own rules are always published.
	rule.Enabled = true
	if old, ok := m.PortalRuleRepo.GetByName(rule.Name); ok {
		rule.Id = old.Id
		if !refreshAccess {
			rule.MatchPathPrefix = old.MatchPathPrefix
		}
	}
	entry := m.PortalEntryCore.BuiltIn()
	rule.EntryId = entry.Id
	rule.MatchScheme = entry.Scheme
	rule.MatchHost = entry.Host
	rule.MatchPort = entry.Port
	// The entry owns the access from here on.
	rule.EntryName = ""
	rule.normalizeAndValidate()
	m.checkMatchesUnique(&rule)
	m.PortalRuleRepo.Save(&rule)
}
