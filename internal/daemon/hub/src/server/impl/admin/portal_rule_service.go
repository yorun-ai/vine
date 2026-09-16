package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalRuleApiServiceServerImpl struct {
	skeled.DefaultPortalRuleApiServiceServer

	PortalRuleCore  *core.PortalRuleCore  `inject:""`
	PortalEntryCore *core.PortalEntryCore `inject:""`
	PortalSiteRepo  core.PortalSiteRepo   `inject:""`
}

func (s *PortalRuleApiServiceServerImpl) List() []skeled.PortalRuleListItem {
	rules := s.PortalRuleCore.List()
	ret := make([]skeled.PortalRuleListItem, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, toServerPortalRuleListItem(s.ruleEntry(rule), rule, s.PortalSiteRepo))
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

func toServerPortalRule(rule *core.PortalRule, entry *core.PortalEntry, fieldSources []skeled.FieldSource) skeled.PortalRule {
	resolvedMatchPathPrefix, resolvedRoutePathPrefix := resolvePortalRulePaths(rule, nil)
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
		ret.MatchScheme = entry.Scheme
		ret.MatchHost = entry.Host
		ret.MatchPort = entry.Port
	}
	return ret
}

func (s *PortalRuleApiServiceServerImpl) toServerPortalRule(rule *core.PortalRule, fieldSources []skeled.FieldSource) skeled.PortalRule {
	ret := toServerPortalRule(rule, s.ruleEntry(rule), fieldSources)
	if s.PortalSiteRepo != nil {
		if site, ok := s.PortalSiteRepo.GetByName(rule.RouteSiteName); ok {
			ret.ResolvedMatchPathPrefix, ret.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, site)
		}
	}
	return ret
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

func resolvePortalRulePaths(rule *core.PortalRule, site *core.PortalSite) (string, string) {
	return core.ResolvePortalRulePaths(rule, site)
}

// toServerPortalRuleListItem maps a rule for list responses, which carry the
// entity values without its seed provenance.
func toServerPortalRuleListItem(entry *core.PortalEntry, rule *core.PortalRule, siteRepo core.PortalSiteRepo, sites ...*core.PortalSite) skeled.PortalRuleListItem {
	detail := toServerPortalRule(rule, entry, nil)
	if len(sites) > 0 {
		detail.ResolvedMatchPathPrefix, detail.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, sites[0])
	} else if siteRepo != nil {
		if site, ok := siteRepo.GetByName(rule.RouteSiteName); ok {
			detail.ResolvedMatchPathPrefix, detail.ResolvedRoutePathPrefix = resolvePortalRulePaths(rule, site)
		}
	}
	return skeled.PortalRuleListItem{
		Enabled:                 detail.Enabled,
		Id:                      detail.Id,
		Name:                    detail.Name,
		MatchScheme:             detail.MatchScheme,
		MatchHost:               detail.MatchHost,
		MatchPort:               detail.MatchPort,
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
			Entry:            conflict.Access.Name,
			Match:            conflict.MatchText(),
			PublishedRuleId:  publishedId,
			PublishedRule:    conflict.Published,
			SuppressedRuleId: suppressedId,
			SuppressedRule:   conflict.Suppressed,
		})
	}
	return ret
}
