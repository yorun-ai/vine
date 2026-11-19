package admin

import (
	"fmt"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalRuleApiServiceServerImpl struct {
	skeled.DefaultPortalRuleApiServiceServer

	PortalRuleListRepo core.PortalRuleListRepo `inject:""`
	PortalRuleCore     *core.PortalRuleCore    `inject:""`
	PortalEntryCore    *core.PortalEntryCore   `inject:""`
	PortalSiteRepo     core.PortalSiteRepo     `inject:""`
}

func (s *PortalRuleApiServiceServerImpl) List() []skeled.PortalRuleListItem {
	views := s.PortalRuleListRepo.ListViews()
	ret := make([]skeled.PortalRuleListItem, 0, len(views))
	for _, view := range views {
		ret = append(ret, toServerPortalRuleListItem(view.Entry, view.Rule, view.Site))
	}
	return ret
}

func (s *PortalRuleApiServiceServerImpl) Get(id int) skeled.PortalRule {
	rule := s.PortalRuleCore.Get(id)
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Create(creation skeled.PortalRuleCreation) skeled.PortalRule {
	// The entry owns the access, so a rule joins an entry Hub already stores
	// instead of declaring an access of its own.
	entry, ok := s.PortalEntryCore.FindByName(creation.EntryName)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", creation.EntryName))
	var routePathPrefix string
	if creation.RoutePathPrefix != nil {
		routePathPrefix = *creation.RoutePathPrefix
	}
	rule := s.PortalRuleCore.Create(core.PortalRule{
		Name:                    creation.Name,
		EntryId:                 entry.Id,
		MatchPathPrefix:         creation.MatchPathPrefix,
		RouteType:               creation.RouteType,
		RouteSiteName:           creation.RouteSiteName,
		RouteRedirectionPattern: creation.RouteRedirectionPattern,
		RoutePathPrefix:         routePathPrefix,
		Enabled:                 core.EnabledOrDefault(creation.Enabled),
	})
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Update(id int, update skeled.PortalRuleUpdate) skeled.PortalRule {
	rule := s.PortalRuleCore.Update(id, core.PortalRuleUpdate{
		Name:                    update.Name,
		MatchPathPrefix:         update.MatchPathPrefix,
		RouteType:               update.RouteType,
		RouteSiteName:           update.RouteSiteName,
		RouteRedirectionPattern: update.RouteRedirectionPattern,
		RoutePathPrefix:         update.RoutePathPrefix,
		Enabled:                 update.Enabled,
	})
	return s.toServerPortalRule(rule, toServerFieldSources(rule.FieldSources))
}

func (s *PortalRuleApiServiceServerImpl) Remove(id int) {
	s.PortalRuleCore.Remove(id)
}

func toServerPortalRule(rule *core.PortalRule, entry *core.PortalEntry, site *core.PortalSite, fieldSources []skeled.FieldSource) skeled.PortalRule {
	resolvedMatchPathPrefix, resolvedRoutePathPrefix := core.ResolvePortalRulePaths(rule, site)
	ret := skeled.PortalRule{
		Enabled:                 rule.Enabled,
		Id:                      rule.Id,
		Name:                    rule.Name,
		MatchPathPrefix:         rule.MatchPathPrefix,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		RoutePathPrefix:         rule.RoutePathPrefix,
		ResolvedMatchPathPrefix: resolvedMatchPathPrefix,
		ResolvedRoutePathPrefix: resolvedRoutePathPrefix,
		FieldSources:            fieldSources,
	}
	if entry != nil {
		ret.EntryName = entry.Name
		ret.EntryAddresses = []string{}
		for _, access := range entry.EffectiveAccesses() {
			host := entry.Http.Host
			if host == "" {
				host = "<IP>"
			}
			if strings.Contains(host, ":") {
				host = "[" + host + "]"
			}
			ret.EntryAddresses = append(ret.EntryAddresses, fmt.Sprintf("%s://%s:%d", access.Scheme, host, access.Port))
		}
	}
	return ret
}

func (s *PortalRuleApiServiceServerImpl) toServerPortalRule(rule *core.PortalRule, fieldSources []skeled.FieldSource) skeled.PortalRule {
	var site *core.PortalSite
	if rule.RouteType == core.PortalRuleRouteTypeSite && s.PortalSiteRepo != nil {
		site, _ = s.PortalSiteRepo.GetByName(rule.RouteSiteName)
	}
	return toServerPortalRule(rule, s.ruleEntry(rule), site, fieldSources)
}

// ruleEntry returns the entry a rule belongs to. An entry Hub cannot resolve
// leaves the access of the response empty instead of failing the detail.
func (s *PortalRuleApiServiceServerImpl) ruleEntry(rule *core.PortalRule) *core.PortalEntry {
	if s.PortalEntryCore == nil {
		return nil
	}
	entry, ok := s.PortalEntryCore.FindById(rule.EntryId)
	if !ok {
		return nil
	}
	return entry
}

// toServerPortalRuleListItem maps a rule for list responses, which carry the
// entity values without its seed provenance.
func toServerPortalRuleListItem(entry *core.PortalEntry, rule *core.PortalRule, site *core.PortalSite) skeled.PortalRuleListItem {
	detail := toServerPortalRule(rule, entry, site, nil)
	return skeled.PortalRuleListItem{
		Enabled:                 detail.Enabled,
		Id:                      detail.Id,
		Name:                    detail.Name,
		EntryName:               detail.EntryName,
		EntryAddresses:          detail.EntryAddresses,
		MatchPathPrefix:         detail.MatchPathPrefix,
		RouteType:               detail.RouteType,
		RouteSiteName:           detail.RouteSiteName,
		RouteRedirectionPattern: detail.RouteRedirectionPattern,
		RoutePathPrefix:         detail.RoutePathPrefix,
		ResolvedMatchPathPrefix: detail.ResolvedMatchPathPrefix,
		ResolvedRoutePathPrefix: detail.ResolvedRoutePathPrefix,
	}
}

// ListConflicts returns the rules that match the same request, so the Dashboard
// shows the operator what Portal cannot order on its own and which rule Hub
// publishes for the request.
func (s *PortalRuleApiServiceServerImpl) ListConflicts() []skeled.PortalRuleConflict {
	conflicts := s.PortalRuleCore.Conflicts()
	ret := make([]skeled.PortalRuleConflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		publishedId, suppressedId := conflict.RuleId, conflict.ConflictId
		if conflict.Published == conflict.Conflict {
			publishedId, suppressedId = conflict.ConflictId, conflict.RuleId
		}
		ret = append(ret, skeled.PortalRuleConflict{
			RuleId:           conflict.RuleId,
			Rule:             conflict.Rule,
			ConflictRuleId:   conflict.ConflictId,
			ConflictRule:     conflict.Conflict,
			Entry:            conflict.Entry.Name,
			Match:            conflict.MatchText(),
			PublishedRuleId:  publishedId,
			PublishedRule:    conflict.Published,
			SuppressedRuleId: suppressedId,
			SuppressedRule:   conflict.Suppressed,
		})
	}
	return ret
}
