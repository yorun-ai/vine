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

type PortalEntryAccessUpdate struct {
	Scheme string
	Host   string
	Port   int
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
	// GetByAccess returns the entry rules belong to.
	GetByAccess(scheme string, host string, port int) (*PortalEntry, bool)
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
	if current, ok := m.PortalEntryRepo.GetByAccess(entry.Scheme, entry.Host, entry.Port); ok {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAccessText(entry)))
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

// Save creates or replaces a complete user entry by name, preserving an existing
// ID, the way a seed applies its entities. Rules reference the entry, so replacing
// an entry republishes the rules it routes with the access it now serves.
func (m *PortalEntryCore) Save(entry PortalEntry) *PortalEntry {
	entry = m.Validate(entry)
	entry.Id = 0
	if current, ok := m.PortalEntryRepo.GetByName(entry.Name); ok {
		entry.Id = current.Id
	}
	if current, ok := m.PortalEntryRepo.GetByAccess(entry.Scheme, entry.Host, entry.Port); ok && current.Id != entry.Id {
		ex.PanicNew(ex.OperationFailed,
			ex.F("portal entry %q already serves %s", current.Name, portalEntryAccessText(entry)))
	}
	// Replacing an entry moves the access of every rule it routes, so a request
	// another rule already serves fails here as well.
	m.checkAccessChangeMatchesUnique(entry.Id, &entry)
	m.PortalEntryRepo.Save(&entry)
	m.saveRules(entry.Id, &entry, entry.Id)
	return &entry
}

// Remove deletes the user entry for an access that routes no rule. Hub keeps the
// rules of an entry, so the operator moves or removes them first.
func (m *PortalEntryCore) Remove(scheme string, host string, port int) {
	access := normalizePortalEntry(PortalEntry{
		Scheme: scheme,
		Host:   host,
		Port:   port,
	})
	entry, ok := m.PortalEntryRepo.GetByAccess(access.Scheme, access.Host, access.Port)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", portalEntryAccessText(access)))

	rules := 0
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId == entry.Id {
			rules++
		}
	}
	ex.PanicNewIfNot(rules == 0, ex.OperationFailed,
		ex.F("portal entry %q still routes %d rules; remove them first", entry.Name, rules))
	ex.PanicNewIfNot(m.PortalEntryRepo.Remove(entry.Id), ex.OperationFailed, ex.F("portal entry %s not found", portalEntryAccessText(access)))
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

// EnsureAccess returns the user entry rules with this access belong to, and
// creates it when no entry serves that access yet.
func (m *PortalEntryCore) EnsureAccess(scheme string, host string, port int) *PortalEntry {
	normalized := normalizePortalEntry(PortalEntry{
		Scheme: scheme,
		Host:   host,
		Port:   port,
		// A rule Hub aggregates into a new entry stays published, the way a rule
		// that names an entry Hub already stores does.
		Enabled: true,
	})
	if current, ok := m.PortalEntryRepo.GetByAccess(normalized.Scheme, normalized.Host, normalized.Port); ok {
		return current
	}
	normalized.Name = PortalEntryName(normalized.Scheme, normalized.Host, normalized.Port)
	m.PortalEntryRepo.Save(&normalized)
	return &normalized
}

// UpdateAccess changes the access of the entry and republishes the rules it
// routes. When another entry already serves the target access, the rules move
// to that entry and the emptied one is removed, so one access never has two
// user entries.
func (m *PortalEntryCore) UpdateAccess(scheme string, host string, port int, update PortalEntryAccessUpdate) PortalEntryView {
	// Callers address the entry the way Admin reports it, so normalize the
	// lookup before matching stored access.
	access := normalizePortalEntry(PortalEntry{
		Name:   PortalEntryName(scheme, host, port),
		Scheme: scheme,
		Host:   host,
		Port:   port,
	})
	current, ok := m.PortalEntryRepo.GetByAccess(access.Scheme, access.Host, access.Port)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", portalEntryAccessText(access)))

	next := normalizePortalEntry(PortalEntry{
		Name:    current.Name,
		Scheme:  update.Scheme,
		Host:    update.Host,
		Port:    update.Port,
		Enabled: current.Enabled,
	})
	if update.Enabled != nil {
		next.Enabled = *update.Enabled
	}
	if target, ok := m.PortalEntryRepo.GetByAccess(next.Scheme, next.Host, next.Port); ok && target.Id != current.Id {
		m.checkAccessChangeMatchesUnique(current.Id, &next)
		m.saveRules(current.Id, &next, target.Id)
		m.PortalEntryRepo.Remove(current.Id)
		return m.view(*target)
	}

	if current.Scheme == next.Scheme && current.Host == next.Host && current.Port == next.Port {
		return m.view(*current)
	}

	m.checkAccessChangeMatchesUnique(current.Id, &next)
	next.Id = current.Id
	m.PortalEntryRepo.Save(&next)
	m.saveRules(next.Id, &next, next.Id)
	return m.view(next)
}

// checkAccessChangeMatchesUnique rejects an access change that would make a rule
// of the entry match the same request as a rule of another entry. Portal resolves
// matching rules by their longest path prefix, so two rules that match
// identically have no defined order.
func (m *PortalEntryCore) checkAccessChangeMatchesUnique(entryId int, access *PortalEntry) {
	stored := m.PortalRuleRepo.List()
	candidates := make([]*PortalRule, 0, len(stored))
	for _, rule := range stored {
		if rule.EntryId != entryId {
			continue
		}
		candidate := *rule
		candidate.MatchScheme = access.Scheme
		candidate.MatchHost = access.Host
		candidate.MatchPort = access.Port
		candidates = append(candidates, &candidate)
	}
	checkPortalRuleMatchesUnique(stored, candidates...)
}

// PortalEntryName returns the name Hub derives for an entry it creates on its
// own, such as the entry a rule joins when no entry serves its access yet.
func PortalEntryName(scheme string, host string, port int) string {
	if host == "" {
		return fmt.Sprintf("%s:%d", scheme, port)
	}
	return fmt.Sprintf("%s:%s:%d", scheme, host, port)
}

// portalEntryAccessText renders the access an entry serves for error messages.
func portalEntryAccessText(entry PortalEntry) string {
	if entry.Host == "" {
		return fmt.Sprintf("%s:%d", entry.Scheme, entry.Port)
	}
	return fmt.Sprintf("%s:%s:%d", entry.Scheme, entry.Host, entry.Port)
}

// normalizePortalEntry returns the entry with the name and access Hub stores: a
// trimmed name, a lowercase scheme, a trimmed host, and the port Portal listens
// on. The name stays empty when the caller does not name the entry.
func normalizePortalEntry(entry PortalEntry) PortalEntry {
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Scheme = strings.ToLower(strings.TrimSpace(entry.Scheme))
	entry.Host = strings.TrimSpace(entry.Host)
	ex.PanicNewIfNot(entry.Scheme == "http" || entry.Scheme == "https", ex.OperationFailed, ex.F("unknown portal entry scheme: %s", entry.Scheme))
	ex.PanicNewIfNot(entry.Port >= 0 && entry.Port <= 65535, ex.OperationFailed, "portal entry port must be between 0 and 65535")
	entry.Port = portalEntrySchemePort(entry.Scheme, entry.Port)
	return entry
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
// Hub republishes them with the access they now resolve.
func (m *PortalEntryCore) saveRules(from int, access *PortalEntry, to int) {
	for _, rule := range m.PortalRuleRepo.List() {
		if rule.EntryId != from {
			continue
		}
		rule.EntryId = to
		rule.MatchScheme = access.Scheme
		rule.MatchHost = access.Host
		rule.MatchPort = access.Port
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
