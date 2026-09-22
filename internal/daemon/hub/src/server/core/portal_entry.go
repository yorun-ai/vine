package core

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/util/vslice"
)

const (
	portalEntryDefaultHTTPPort  = 80
	portalEntryDefaultHTTPSPort = 443
)

const (
	PortalRuleRouteTypeSite              = "SITE"
	PortalRuleRouteTypePermanentRedirect = "PERMANENT_REDIRECT"
	PortalRuleRouteTypeTemporaryRedirect = "TEMPORARY_REDIRECT"
)

// PortalEntry is a Portal access entry: the name Hub shows for it and the
// scheme, host, and port Portal serves. Hub stores an entry for every access
// user rules use, so changing an entry changes the access of all the rules it
// routes at once.
type PortalEntry struct {
	Id int
	// Name is the entry label. Hub derives the name from the access of an entry
	// it creates on its own, and a seed or an operator can name an entry instead.
	Name   string
	Scheme string
	Host   string
	Port   int
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
	Name *string
	// Scheme, Host, and Port are optional and keep the stored access when they
	// are nil.
	Scheme *string
	Host   *string
	Port   *int
	// Enabled is optional and keeps the stored switch when it is nil.
	Enabled *bool
}

type PortalEntryCreation struct {
	Name   string
	Scheme string
	Host   string
	Port   int
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
	// GetBySchemeHostPort returns the entry that serves the scheme, host, and
	// port, which is how a rule joins the entry it belongs to.
	GetBySchemeHostPort(scheme string, host string, port int) (*PortalEntry, bool)
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
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		if a.Scheme != b.Scheme {
			return cmpString(a.Scheme, b.Scheme) < 0
		}
		return cmpString(a.Host, b.Host) < 0
	})
}

// Create adds the named user entry for an access no user entry serves yet.
func (m *PortalEntryCore) Create(creation PortalEntryCreation) PortalEntryView {
	entry := m.Validate(PortalEntry{
		Name:    creation.Name,
		Scheme:  creation.Scheme,
		Host:    creation.Host,
		Port:    creation.Port,
		Enabled: EnabledOrDefault(creation.Enabled),
	})
	_, ok := m.PortalEntryRepo.GetByName(entry.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("portal entry %q already exists", entry.Name))
	if current, ok := m.PortalEntryRepo.GetBySchemeHostPort(entry.Scheme, entry.Host, entry.Port); ok {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAddress(entry)))
	}
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
// without accessing storage, so a seed fails on a bad scheme, host, or port
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
	if current, ok := m.PortalEntryRepo.GetBySchemeHostPort(entry.Scheme, entry.Host, entry.Port); ok && current.Id != entry.Id {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAddress(entry)))
	}
	m.validateWildcardRules(entry, entry.Id)
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

// EnsureEntry returns the entry rules with this scheme, host, and port belong
// to, and creates it when no entry serves them yet.
func (m *PortalEntryCore) EnsureEntry(scheme string, host string, port int) *PortalEntry {
	normalized := normalizePortalEntry(PortalEntry{
		Scheme: scheme,
		Host:   host,
		Port:   port,
		// A rule Hub aggregates into a new entry stays published, the way a rule
		// that names an entry Hub already stores does.
		Enabled: true,
	})
	if current, ok := m.PortalEntryRepo.GetBySchemeHostPort(normalized.Scheme, normalized.Host, normalized.Port); ok {
		return current
	}
	normalized.Name = PortalEntryName(normalized.Scheme, normalized.Host, normalized.Port)
	m.PortalEntryRepo.Save(&normalized)
	return &normalized
}

// Update changes the label and the access of the entry and republishes the rules
// it routes. When another entry already serves the target access, the rules move
// to that entry and the emptied one is removed, so one access never has two
// entries.
func (m *PortalEntryCore) Update(id int, update PortalEntryUpdate) PortalEntryView {
	current, ok := m.PortalEntryRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", id))

	next := *current
	if update.Name != nil {
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
	accessChanged := next.Scheme != current.Scheme || next.Host != current.Host || next.Port != current.Port
	if accessChanged {
		if target, ok := m.PortalEntryRepo.GetBySchemeHostPort(next.Scheme, next.Host, next.Port); ok && target.Id != current.Id {
			m.saveRules(current.Id, target.Id)
			m.PortalEntryRepo.Remove(current.Id)
			return m.view(*target)
		}
	}
	if !accessChanged && next.Name == current.Name && next.Enabled == current.Enabled {
		return m.view(*current)
	}

	next.Id = current.Id
	m.PortalEntryRepo.Save(&next)
	// Portal reaches the entry through its rules, so only an access change has to
	// republish them.
	if accessChanged {
		m.saveRules(next.Id, next.Id)
	}
	return m.view(next)
}

// PortalEntryName returns the name Hub derives for an entry it creates on its
// own, such as the entry a rule joins when no entry serves its access yet.
func PortalEntryName(scheme string, host string, port int) string {
	if host == "" {
		return fmt.Sprintf("%s:%d", scheme, port)
	}
	return fmt.Sprintf("%s:%s:%d", scheme, host, port)
}

// portalEntryAddress renders the access an entry serves for error messages.
func portalEntryAddress(entry PortalEntry) string {
	if entry.Host == "" {
		return fmt.Sprintf("%s:%d", entry.Scheme, entry.Port)
	}
	return fmt.Sprintf("%s:%s:%d", entry.Scheme, entry.Host, entry.Port)
}

// normalizePortalEntry returns the entry the way Hub stores it: a trimmed name,
// a lowercase scheme, a trimmed host, and the port Portal listens on. The name
// stays empty when the caller does not name the entry.
func normalizePortalEntry(entry PortalEntry) PortalEntry {
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Scheme = strings.ToLower(strings.TrimSpace(entry.Scheme))
	entry.Host = strings.TrimSpace(entry.Host)
	if strings.HasPrefix(entry.Host, "*.") {
		entry.Host = strings.ToLower(entry.Host)
	}
	ex.PanicNewIfNot(entry.Scheme == "http" || entry.Scheme == "https", ex.OperationFailed, ex.F("unknown portal entry scheme: %s", entry.Scheme))
	ex.PanicNewIfNot(entry.Port >= 0 && entry.Port <= 65535, ex.OperationFailed, "portal entry port must be between 0 and 65535")
	ex.PanicNewIfNot(portalEntryHostAccepted(entry.Host), ex.OperationFailed, "portal entry host must be a hostname, IP or leading *. wildcard without a port")
	entry.Port = portalEntrySchemePort(entry.Scheme, entry.Port)
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

// portalEntrySchemePort returns the port Portal listens on. An unset port means
// the default port of the scheme.
func portalEntrySchemePort(scheme string, port int) int {
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

// validateWildcardRules also covers entry access changes and entry merges.
func (m *PortalEntryCore) validateWildcardRules(entry PortalEntry, ruleEntryId int) {
	if !strings.HasPrefix(entry.Host, "*.") {
		return
	}
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId == ruleEntryId {
			rule.ValidateWildcardTarget(entry, m.portalRuleSite(rule))
		}
	}
}
