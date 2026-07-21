package core

import (
	"fmt"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/util/vslice"
)

const (
	portalEntryDefaultHTTPPort  = 80
	portalEntryDefaultHTTPSPort = 443
)

const (
	PortalRuleTargetTypeSite              = "SITE"
	PortalRuleTargetTypePermanentRedirect = "PERMANENT_REDIRECT"
	PortalRuleTargetTypeTemporaryRedirect = "TEMPORARY_REDIRECT"
)

type PortalEntry struct {
	Name    string
	Scheme  string
	Host    string
	Port    int
	Rules   []PortalEntryRule
	BuiltIn bool
}

type PortalEntryAccessUpdate struct {
	Scheme string
	Host   string
	Port   int
}

type PortalEntryRule struct {
	Rule *PortalRule
	Site *PortalSite
}

type PortalEntryCore struct {
	PortalRuleRepo PortalRuleRepo `inject:""`
	PortalSiteRepo PortalSiteRepo `inject:""`
}

type _PortalEntryKey struct {
	Scheme string
	Host   string
	Port   int
}

func (m *PortalEntryCore) List() []PortalEntry {
	rules := m.PortalRuleRepo.ListRules()
	entriesByKey := map[_PortalEntryKey]*PortalEntry{}
	for _, rule := range rules {
		if rule.BuiltIn {
			continue
		}
		if !isPortalEntryRuleTargetType(rule.TargetType) {
			continue
		}

		key := _PortalEntryKey{
			Scheme: rule.Scheme,
			Host:   rule.Host,
			Port:   portalEntryRulePort(rule.Scheme, rule.Port),
		}
		entry, ok := entriesByKey[key]
		if !ok {
			entry = &PortalEntry{
				Name:   portalEntryName(key),
				Scheme: key.Scheme,
				Host:   key.Host,
				Port:   key.Port,
			}
			entriesByKey[key] = entry
		}
		entry.Rules = append(entry.Rules, PortalEntryRule{
			Rule: &rule,
			Site: m.portalRuleSite(rule),
		})
		entry.BuiltIn = entry.BuiltIn || rule.BuiltIn
	}

	entries := make([]PortalEntry, 0, len(entriesByKey))
	for _, entry := range entriesByKey {
		item := *entry
		item.Rules = sortedPortalEntryRules(entry.Rules)
		entries = append(entries, item)
	}

	return vslice.SortBy(entries, func(a PortalEntry, b PortalEntry) bool {
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		if a.Scheme != b.Scheme {
			return cmpString(a.Scheme, b.Scheme) < 0
		}
		return cmpString(a.Host, b.Host) < 0
	})
}

func (m *PortalEntryCore) UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate) PortalEntry {
	currentKey := normalizePortalEntryKey(scheme, host, port)
	nextKey := normalizePortalEntryKey(update.Scheme, update.Host, update.Port)

	rules := m.PortalRuleRepo.ListRules()
	updated := false
	for _, rule := range rules {
		if rule.BuiltIn || !isPortalEntryRuleTargetType(rule.TargetType) {
			continue
		}
		if !portalEntryRuleMatchesKey(rule, currentKey) {
			continue
		}

		rule.Scheme = nextKey.Scheme
		rule.Host = nextKey.Host
		rule.Port = nextKey.Port
		m.PortalRuleRepo.SaveRule(&rule)
		updated = true
	}

	ex.PanicNewIfNot(updated, ex.OperationFailed, ex.F("portal entry %s not found", portalEntryName(currentKey)))

	for _, entry := range m.List() {
		if portalEntryMatchesKey(entry, nextKey) {
			return entry
		}
	}
	ex.PanicNew(ex.OperationFailed, ex.F("portal entry %s not found", portalEntryName(nextKey)))
	return PortalEntry{}
}

func isPortalEntryRuleTargetType(value string) bool {
	return value == PortalRuleTargetTypeSite ||
		value == PortalRuleTargetTypePermanentRedirect ||
		value == PortalRuleTargetTypeTemporaryRedirect
}

func portalEntryRulePort(scheme string, port int) int {
	switch scheme {
	case "http":
		if port == 0 {
			return portalEntryDefaultHTTPPort
		}
	case "https":
		if port == 0 {
			return portalEntryDefaultHTTPSPort
		}
	default:
		ex.PanicNew(ex.OperationFailed, ex.F("unknown portal entry scheme: %s", scheme))
	}
	return port
}

func portalEntryName(key _PortalEntryKey) string {
	if key.Host == "" {
		return fmt.Sprintf("%s:%d", key.Scheme, key.Port)
	}
	return fmt.Sprintf("%s:%s:%d", key.Scheme, key.Host, key.Port)
}

func normalizePortalEntryKey(scheme string, host string, port int) _PortalEntryKey {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	host = strings.TrimSpace(host)
	ex.PanicNewIfNot(scheme == "http" || scheme == "https", ex.OperationFailed, "portal entry scheme must be http or https")
	ex.PanicNewIfNot(port >= 0 && port <= 65535, ex.OperationFailed, "portal entry port must be between 0 and 65535")
	return _PortalEntryKey{
		Scheme: scheme,
		Host:   host,
		Port:   portalEntryRulePort(scheme, port),
	}
}

func portalEntryRuleMatchesKey(rule PortalRule, key _PortalEntryKey) bool {
	return rule.Scheme == key.Scheme &&
		rule.Host == key.Host &&
		portalEntryRulePort(rule.Scheme, rule.Port) == key.Port
}

func portalEntryMatchesKey(entry PortalEntry, key _PortalEntryKey) bool {
	return entry.Scheme == key.Scheme && entry.Host == key.Host && entry.Port == key.Port
}

func (m *PortalEntryCore) portalRuleSite(rule PortalRule) *PortalSite {
	if rule.TargetType != PortalRuleTargetTypeSite {
		return nil
	}
	site, ok := m.PortalSiteRepo.GetEntryByName(rule.SiteName)
	if !ok {
		return nil
	}
	return site
}

func sortedPortalEntryRules(rules []PortalEntryRule) []PortalEntryRule {
	return vslice.SortBy(rules, func(a PortalEntryRule, b PortalEntryRule) bool {
		if len(a.Rule.PathPrefix) != len(b.Rule.PathPrefix) {
			return len(a.Rule.PathPrefix) > len(b.Rule.PathPrefix)
		}
		return cmpString(a.Rule.Name, b.Rule.Name) < 0
	})
}
