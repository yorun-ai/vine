package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/util/httputil"
)

// Structs

type PortalRule struct {
	FieldSources FieldSources
	Id           int
	Name         string
	// EntryId identifies the Portal entry a rule belongs to: the entry stores the
	// scheme, host, and port, so a rule never carries them itself.
	EntryId int

	MatchPathPrefix         string
	RouteType               string
	RouteSiteName           string
	RouteRedirectionPattern string
	RoutePathPrefix         string
	// Enabled decides whether Hub publishes the rule to Portal.
	Enabled bool
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
	PortalSiteRepo  PortalSiteRepo   `inject:""`
}

func (m *PortalRuleCore) List() []*PortalRule {
	rules := m.PortalRuleRepo.List()
	ret := make([]*PortalRule, 0, len(rules))
	for _, rule := range rules {
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

// Create stores a new rule under the entry it belongs to. The caller resolves
// the entry, because only it knows whether the rule names an entry or declares
// the scheme, host, and port of the entry Hub ensures.
func (m *PortalRuleCore) Create(rule PortalRule) *PortalRule {
	_, ok := m.PortalRuleRepo.GetByName(rule.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("entry rule %q already exists", rule.Name))
	rule.Id = 0
	return m.save(rule)
}

func (m *PortalRuleCore) Update(id int, update PortalRuleUpdate) *PortalRule {
	rule, ok := m.PortalRuleRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))

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
		// A seed declares the switch as disabled, so it owns that source path.
		next.FieldSources = overrideFieldSource(next.FieldSources, "/disabled")
		next.Enabled = *update.Enabled
	}

	return m.save(next)
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
	_, ok := m.PortalRuleRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))

	ok = m.PortalRuleRepo.Remove(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", id))
}

// normalizeAndValidate enforces the constraints of a complete rule before persistence.
func (r *PortalRule) normalizeAndValidate() {
	fail := func(ok bool, message string) {
		ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal rule %q: %s", r.Name, message))
	}
	fail(strings.TrimSpace(r.Name) != "", "name is required")
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
	rule.normalizeAndValidate()
	return rule
}

// portalEntryMatchKey identifies the requests one entry and path prefix match.
func portalEntryMatchKey(entry PortalEntry, matchPathPrefix string) string {
	return entry.Scheme + "\x00" + entry.Host + "\x00" + strconv.Itoa(entry.Port) + "\x00" + matchPathPrefix
}

// PortalRuleConflict names two published rules that match the same request: the
// entry they belong to and the match path prefix their sites resolve.
type PortalRuleConflict struct {
	Entry           PortalEntry
	MatchPathPrefix string
	Rule            string
	RuleId          int
	Conflict        string
	ConflictId      int
	// Published and Suppressed name the rules Hub publishes and leaves out:
	// Portal resolves matching rules by their longest path prefix, so two rules
	// that match identically have no defined order, and Hub keeps the rule whose
	// name sorts first.
	Published  string
	Suppressed string
}

// PortalRuleConflictWinner returns the rule Hub publishes when two rules match
// the same request: the rule whose name sorts first, so the choice never depends
// on the order Hub applied them.
func PortalRuleConflictWinner(rule string, conflict string) (published string, suppressed string) {
	if conflict < rule {
		return conflict, rule
	}
	return rule, conflict
}

// MatchText renders the request both rules match.
func (c PortalRuleConflict) MatchText() string {
	host := c.Entry.Host
	if host == "" {
		host = "*"
	}
	return fmt.Sprintf("%s://%s:%d%s", c.Entry.Scheme, host, c.Entry.Port, c.MatchPathPrefix)
}

// Save creates or replaces a complete user rule by name, preserving an existing
// ID, the way a seed applies its entities. The caller resolves the entry the
// rule belongs to.
func (m *PortalRuleCore) Save(rule PortalRule) *PortalRule {
	rule.Id = 0
	if current, ok := m.PortalRuleRepo.GetByName(rule.Name); ok {
		rule.Id = current.Id
	}
	return m.save(rule)
}

// save stores a validated rule under the entry it belongs to.
func (m *PortalRuleCore) save(rule PortalRule) *PortalRule {
	ex.PanicNewIfNot(rule.EntryId != 0, ex.OperationFailed,
		ex.F("portal rule %q: the entry it belongs to is required", rule.Name))
	m.PortalEntryCore.Get(rule.EntryId)
	rule = m.Validate(rule)
	m.PortalRuleRepo.Save(&rule)
	return &rule
}

// Conflicts returns the requests more than one published rule matches, once the
// Web mount path of each site decides its prefix. Portal resolves matching rules
// by their longest path prefix, so two rules that match identically have no
// defined order. Hub applies a seed before applications register their schemas,
// so no write can answer this question: only a caller that reads the registered
// schemas can report what Hub cannot resolve on its own.
func (m *PortalRuleCore) Conflicts() []PortalRuleConflict {
	type _Match struct {
		name string
		id   int
	}
	matched := map[string]_Match{}
	conflicts := []PortalRuleConflict{}
	for _, rule := range m.PortalRuleRepo.List() {
		if !rule.Enabled {
			continue
		}
		entry, ok := m.PortalEntryCore.FindById(rule.EntryId)
		if !ok || !entry.Enabled {
			continue
		}
		site := m.ruleSite(rule)
		if rule.RouteType == PortalRuleRouteTypeSite && site != nil && !site.Enabled {
			continue
		}
		matchPathPrefix, _ := ResolvePortalRulePaths(rule, site)
		key := portalEntryMatchKey(*entry, matchPathPrefix)
		matchedRule, found := matched[key]
		if !found {
			matched[key] = _Match{name: rule.Name, id: rule.Id}
			continue
		}
		published, suppressed := PortalRuleConflictWinner(rule.Name, matchedRule.name)
		conflicts = append(conflicts, PortalRuleConflict{
			Entry:           *entry,
			MatchPathPrefix: matchPathPrefix,
			Rule:            rule.Name,
			RuleId:          rule.Id,
			Conflict:        matchedRule.name,
			ConflictId:      matchedRule.id,
			Published:       published,
			Suppressed:      suppressed,
		})
	}
	return conflicts
}

// ruleSite returns the site a rule targets, with the Web mount path its schema
// declares.
func (m *PortalRuleCore) ruleSite(rule *PortalRule) *PortalSite {
	if rule.RouteType != PortalRuleRouteTypeSite || rule.RouteSiteName == "" {
		return nil
	}
	site, ok := m.PortalSiteRepo.GetByName(rule.RouteSiteName)
	if !ok {
		return nil
	}
	return site
}
