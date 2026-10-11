package core

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/listenip"
	"go.yorun.ai/vine/util/vslice"
)

const (
	PortalRuleRouteTypeSite              = "SITE"
	PortalRuleRouteTypePermanentRedirect = "PERMANENT_REDIRECT"
	PortalRuleRouteTypeTemporaryRedirect = "TEMPORARY_REDIRECT"
)

// PortalEntry is a Portal access entry: the name Hub shows for it and the
// protocol and its listener configuration. Hub stores an entry for every access
// user rules use, so changing an entry changes the access of all the rules it
// routes at once.
type PortalEntry struct {
	Id int
	// Name is the entry label supplied by a seed or an operator.
	Name     string
	Protocol string
	Http     *PortalEntryHTTP
	// ListenIPs restricts listener addresses; empty preserves legacy wildcard TCP.
	ListenIPs []string
	// Enabled decides whether Hub publishes the rules of the entry to Portal.
	Enabled bool
}

// PortalEntryView is one entry together with the rules Portal routes through
// it, in the order Portal resolves them.
type PortalEntryView struct {
	PortalEntry
	Rules []PortalEntryRule
}

type PortalEntryUpdate struct {
	// Name is optional and keeps the stored label when it is nil.
	Name      *string
	Protocol  *string
	Http      *PortalEntryHTTPUpdate
	ListenIPs *[]string
	// Enabled is optional and keeps the stored switch when it is nil.
	Enabled *bool
}

type PortalEntryCreation struct {
	Name     string
	Protocol string
	Http     *PortalEntryHTTP
	// ListenIPs restricts listener addresses; empty preserves legacy wildcard TCP.
	ListenIPs []string
	// Enabled is optional and defaults to true.
	Enabled *bool
}

type PortalEntryRule struct {
	Rule *PortalRule
	Site *PortalSite
}

// PortalEntryRepo stores Portal access entries. List and the lookups return
// entities the caller owns.
type PortalEntryRepo interface {
	List() []*PortalEntry
	GetById(id int) (*PortalEntry, bool)
	// GetByName returns the entry Hub labels with the name.
	GetByName(name string) (*PortalEntry, bool)
	Save(entry *PortalEntry)
	Remove(id int) bool
}

type PortalEntryCore struct {
	PortalEntryRepo PortalEntryRepo `inject:""`
	PortalRuleRepo  PortalRuleRepo  `inject:""`
	PortalSiteRepo  PortalSiteRepo  `inject:""`
}

// List returns the entries with the rules they route, ordered by port, scheme,
// and host. An entry that routes no rule is still listed, because an operator
// creates entries before the rules that use them.
func (m *PortalEntryCore) List() []PortalEntryView {
	rulesByEntry := m.rulesByEntry()
	views := make([]PortalEntryView, 0, len(rulesByEntry))
	for _, stored := range m.PortalEntryRepo.List() {
		entry := normalizePortalEntry(*stored)
		views = append(views, PortalEntryView{
			PortalEntry: entry,
			Rules:       sortedPortalEntryRules(rulesByEntry[entry.Id]),
		})
	}

	return vslice.SortBy(views, func(a PortalEntryView, b PortalEntryView) bool {
		aa, ba := a.Accesses()[0], b.Accesses()[0]
		if aa.Port != ba.Port {
			return aa.Port < ba.Port
		}
		if aa.Scheme != ba.Scheme {
			return aa.Scheme < ba.Scheme
		}
		if a.Http.Host != b.Http.Host {
			return a.Http.Host < b.Http.Host
		}
		return a.Name < b.Name
	})
}

// Create adds the named user entry for an access no user entry serves yet.
func (m *PortalEntryCore) Create(creation PortalEntryCreation) PortalEntryView {
	entry := m.Validate(PortalEntry{
		Name:      creation.Name,
		Protocol:  creation.Protocol,
		Http:      creation.Http,
		ListenIPs: creation.ListenIPs,
		Enabled:   EnabledOrDefault(creation.Enabled),
	})
	_, ok := m.PortalEntryRepo.GetByName(entry.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("portal entry %q already exists", entry.Name))
	if current, ok := m.sameAccess(entry); ok {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAddress(entry)))
	}
	m.validateListeners(entry)
	m.PortalEntryRepo.Save(&entry)
	return PortalEntryView{PortalEntry: entry}
}

// Validate checks and normalizes a complete user entry without accessing storage.
func (*PortalEntryCore) Validate(entry PortalEntry) PortalEntry {
	entry = normalizePortalEntry(entry)
	ex.PanicNewIfNot(entry.Name != "", ex.OperationFailed, "portal entry name is required")
	return entry
}

// Normalize checks and normalizes the fields of an entry a caller declares
// without accessing storage, so a seed fails on an invalid protocol configuration
// before Hub applies the rule that declares them. The name stays optional here,
// because Hub derives the name of an entry it creates on its own.
func (*PortalEntryCore) Normalize(entry PortalEntry) PortalEntry {
	return normalizePortalEntry(entry)
}

// Save creates or replaces a complete user entry by name, preserving an existing
// ID, the way a seed applies its entities. Rules reference the entry, so replacing
// an entry republishes the rules it routes with the access it now serves.
func (m *PortalEntryCore) Save(entry PortalEntry) *PortalEntry {
	entry = m.Validate(entry)
	entry.Id = 0
	if current, ok := m.PortalEntryRepo.GetByName(entry.Name); ok {
		entry.Id = current.Id
	}
	if current, ok := m.sameAccess(entry); ok && current.Id != entry.Id {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAddress(entry)))
	}
	m.validateWildcardRules(entry, entry.Id)
	m.validateListeners(entry)
	m.PortalEntryRepo.Save(&entry)
	m.saveRules(entry.Id, entry.Id)
	return &entry
}

// Remove deletes the entry that routes no rule. Hub keeps the rules of an entry,
// so the operator moves or removes them first.
func (m *PortalEntryCore) Remove(id int) {
	entry, ok := m.PortalEntryRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	rules := 0
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId == entry.Id {
			rules++
		}
	}
	ex.PanicNewIfNot(rules == 0, ex.OperationFailed,
		ex.F("portal entry %q still routes %d rules; remove them first", entry.Name, rules))
	ex.PanicNewIfNot(m.PortalEntryRepo.Remove(entry.Id), ex.OperationFailed, ex.F("portal entry %d not found", id))
}

// Get returns the entry with the id.
func (m *PortalEntryCore) Get(id int) *PortalEntry {
	entry, ok := m.PortalEntryRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))
	return entry
}

// FindByName returns the entry Hub labels with the name.
func (m *PortalEntryCore) FindByName(name string) (*PortalEntry, bool) {
	return m.PortalEntryRepo.GetByName(name)
}

// FindById returns the entry with the id.
func (m *PortalEntryCore) FindById(id int) (*PortalEntry, bool) {
	return m.PortalEntryRepo.GetById(id)
}

func (m *PortalEntryCore) Update(id int, update PortalEntryUpdate) PortalEntryView {
	current, ok := m.PortalEntryRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))

	next := normalizePortalEntry(*current)
	if update.Protocol != nil {
		next.Protocol = *update.Protocol
		ex.PanicNewIfNot(strings.TrimSpace(next.Protocol) != "", ex.OperationFailed, "protocol is empty")
	}
	if update.Http != nil {
		config := update.Http.Apply(*next.Http)
		next.Http = &config
	}
	if update.Name != nil {
		next.Name = *update.Name
	}
	if update.ListenIPs != nil {
		next.ListenIPs = *update.ListenIPs
	}
	if update.Enabled != nil {
		next.Enabled = *update.Enabled
	}
	next = normalizePortalEntry(m.Validate(next))
	if current.Name != next.Name {
		if other, ok := m.PortalEntryRepo.GetByName(next.Name); ok && other.Id != current.Id {
			ex.PanicNew(ex.OperationFailed, ex.F("portal entry %q already exists", next.Name))
		}
	}
	m.validateWildcardRules(next, current.Id)
	accessChanged := next.Protocol != current.Protocol || *next.Http != *normalizePortalEntry(*current).Http
	if accessChanged {
		if target, ok := m.sameAccess(next); ok && target.Id != current.Id {
			ex.PanicNew(ex.OperationFailed, ex.F("portal entry %q already serves this access", target.Name))
		}
	}
	listenersChanged := !slices.Equal(next.ListenIPs, current.ListenIPs)
	if !accessChanged && !listenersChanged && next.Name == current.Name && next.Enabled == current.Enabled {
		return m.view(*current)
	}

	next.Id = current.Id
	m.validateListeners(next)
	m.PortalEntryRepo.Save(&next)
	// Save republishes listener changes through Syncer. An access change also
	// saves the attached rules to refresh their access-dependent domain state.
	if accessChanged {
		m.saveRules(next.Id, next.Id)
	}
	return m.view(next)
}

// portalEntryAddress renders the access an entry serves for error messages.
func portalEntryAddress(entry PortalEntry) string {
	addresses := []string{}
	for _, access := range entry.Accesses() {
		if entry.Http.Host == "" {
			addresses = append(addresses, fmt.Sprintf("%s:%d", access.Scheme, access.Port))
		} else {
			addresses = append(addresses, fmt.Sprintf("%s:%s:%d", access.Scheme, entry.Http.Host, access.Port))
		}
	}
	return strings.Join(addresses, ", ")
}

// normalizePortalEntry returns the entry the way Hub stores it: a trimmed name,
// a lowercase protocol and a normalized HTTP configuration. The name
// stays empty when the caller does not name the entry.
func normalizePortalEntry(entry PortalEntry) PortalEntry {
	entry.Name = strings.TrimSpace(entry.Name)
	entry = normalizePortalEntryHTTP(entry)
	entry.Http.Host = strings.TrimSpace(entry.Http.Host)
	if strings.HasPrefix(entry.Http.Host, "*.") {
		entry.Http.Host = strings.ToLower(entry.Http.Host)
	}
	ex.PanicNewIfNot(portalEntryHostAccepted(entry.Http.Host), ex.OperationFailed, "portal entry host must be a hostname, IP or leading *. wildcard without a port")
	ips, err := listenip.Normalize(entry.ListenIPs)
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, ex.F("%v", err))
	entry.ListenIPs = ips
	return entry
}

// portalEntryHostAccepted reports whether a host Hub stores is a hostname or IP
// address without a port, so the access an entry declares is a request Portal
// can match.
func portalEntryHostAccepted(host string) bool {
	if host == "" {
		return true
	}
	if strings.HasPrefix(host, "*.") {
		suffix := host[2:]
		if suffix == "" || net.ParseIP(suffix) != nil {
			return false
		}
		for label := range strings.SplitSeq(suffix, ".") {
			if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return false
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
					return false
				}
			}
		}
		return len(suffix) <= 253
	}
	if strings.ContainsAny(host, "/?#@*\\") || strings.IndexFunc(host, unicode.IsSpace) >= 0 || strings.IndexFunc(host, unicode.IsControl) >= 0 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	parsed, err := url.Parse("//" + host)
	return err == nil && parsed.Hostname() == host && !strings.ContainsAny(host, ":[]")
}

func isPortalEntryRuleRouteType(value string) bool {
	return value == PortalRuleRouteTypeSite ||
		value == PortalRuleRouteTypePermanentRedirect ||
		value == PortalRuleRouteTypeTemporaryRedirect
}

func (m *PortalEntryCore) rulesByEntry() map[int][]PortalEntryRule {
	rulesByEntry := map[int][]PortalEntryRule{}
	for _, rule := range m.PortalRuleRepo.List() {
		if !isPortalEntryRuleRouteType(rule.RouteType) {
			continue
		}
		rulesByEntry[rule.EntryId] = append(rulesByEntry[rule.EntryId], PortalEntryRule{
			Rule: rule,
			Site: m.portalRuleSite(rule),
		})
	}
	return rulesByEntry
}

func (m *PortalEntryCore) view(entry PortalEntry) PortalEntryView {
	return PortalEntryView{
		PortalEntry: entry,
		Rules:       sortedPortalEntryRules(m.rulesByEntry()[entry.Id]),
	}
}

// saveRules points the rules of one entry at another entry and saves them, so
// Hub republishes them with the access they now resolve through the entry.
func (m *PortalEntryCore) saveRules(from int, to int) {
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId != from {
			continue
		}
		rule.EntryId = to
		m.PortalRuleRepo.Save(rule)
	}
}

func (m *PortalEntryCore) portalRuleSite(rule *PortalRule) *PortalSite {
	if rule.RouteType != PortalRuleRouteTypeSite {
		return nil
	}
	site, ok := m.PortalSiteRepo.GetByName(rule.RouteSiteName)
	if !ok {
		return nil
	}
	return site
}

func sortedPortalEntryRules(rules []PortalEntryRule) []PortalEntryRule {
	return vslice.SortBy(rules, func(a PortalEntryRule, b PortalEntryRule) bool {
		if len(a.Rule.MatchPathPrefix) != len(b.Rule.MatchPathPrefix) {
			return len(a.Rule.MatchPathPrefix) > len(b.Rule.MatchPathPrefix)
		}
		return cmpString(a.Rule.Name, b.Rule.Name) < 0
	})
}

// validateWildcardRules also covers entry access changes.
func (m *PortalEntryCore) validateWildcardRules(entry PortalEntry, ruleEntryId int) {
	if !strings.HasPrefix(entry.Http.Host, "*.") {
		return
	}
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId == ruleEntryId {
			rule.ValidateWildcardTarget(entry, m.portalRuleSite(rule))
		}
	}
}

// validateListeners rejects bind overlaps while allowing virtual hosts to share
// an identical listener. Entry identity is its protocol configuration.
func (m *PortalEntryCore) validateListeners(entry PortalEntry) {
	if !entry.Enabled {
		return
	}
	entries := []PortalEntry{entry}
	for _, other := range m.PortalEntryRepo.List() {
		if other.Id != entry.Id {
			entries = append(entries, *other)
		}
	}
	ValidatePortalEntryListeners(entries)
}

// ValidatePortalEntryListeners checks a complete configuration for overlapping
// bindings before a seed writes it. Identical listeners of the same scheme share
// a socket; wildcard overlaps and different schemes on the same socket conflict.
func ValidatePortalEntryListeners(entries []PortalEntry) {
	for i, entry := range entries {
		entry = normalizePortalEntry(entry)
		if !entry.Enabled {
			continue
		}
		for _, other := range entries[i+1:] {
			other = normalizePortalEntry(other)
			if !other.Enabled {
				continue
			}
			for _, access := range entry.Accesses() {
				for _, otherAccess := range other.Accesses() {
					if entry.Http.Host == other.Http.Host && access.Scheme == otherAccess.Scheme && access.Port == otherAccess.Port && *entry.Http != *other.Http {
						ex.PanicNew(ex.OperationFailed, ex.F("portal entry %q shares an access with entry %q; configure the transports on one entry", entry.Name, other.Name))
					}
					if access.Port != otherAccess.Port {
						continue
					}
					for _, left := range listenip.Addresses(entry.ListenIPs) {
						for _, right := range listenip.Addresses(other.ListenIPs) {
							if listenip.Overlap(left, right) && (left != right || access.Scheme != otherAccess.Scheme) {
								ex.PanicNew(ex.OperationFailed, ex.F("portal entry %q listener conflicts with entry %q on port %d", entry.Name, other.Name, access.Port))
							}
						}
					}
				}
			}
		}
	}
}

func (m *PortalEntryCore) sameAccess(entry PortalEntry) (*PortalEntry, bool) {
	for _, other := range m.PortalEntryRepo.List() {
		if other.Id != entry.Id && other.Protocol == entry.Protocol && *normalizePortalEntry(*other).Http == *entry.Http {
			return other, true
		}
	}
	return nil, false
}
