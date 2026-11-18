package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vmap"
)

// All rule indexes are owned by Syncer.namesMutex. Keep references to disabled
// and unresolved rules so an entry or site change can make them eligible again.
type _PortalRuleIndex struct {
	byID    map[int]*_IndexedRule
	byEntry _RuleRefs[int]
	bySite  _RuleRefs[string]
	groups  map[string]*_RuleGroup
}

type _IndexedRule struct {
	rule  core.PortalRule
	match string
}

type _RuleGroup struct {
	members       map[*_IndexedRule]struct{}
	winner        *_IndexedRule
	publishedName string
}

type _RuleRefs[K comparable] map[K]map[*_IndexedRule]struct{}

func (refs _RuleRefs[K]) add(key K, rule *_IndexedRule) {
	if refs[key] == nil {
		refs[key] = make(map[*_IndexedRule]struct{})
	}
	refs[key][rule] = struct{}{}
}

func (refs _RuleRefs[K]) remove(key K, rule *_IndexedRule) {
	delete(refs[key], rule)
	if len(refs[key]) == 0 {
		delete(refs, key)
	}
}

func newPortalRuleIndex() _PortalRuleIndex {
	return _PortalRuleIndex{
		byID:    make(map[int]*_IndexedRule),
		byEntry: make(_RuleRefs[int]),
		bySite:  make(_RuleRefs[string]),
		groups:  make(map[string]*_RuleGroup),
	}
}

func (s *Syncer) SyncPortalRule(rule *core.PortalRule) {
	s.SyncPortalRules([]*core.PortalRule{rule})
}

// SyncPortalRules applies a set of rule writes and publishes its final winners
// together. Entries and sites may arrive before or after their rules.
func (s *Syncer) SyncPortalRules(rules []*core.PortalRule) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	changed := make(map[string]struct{})
	for _, rule := range rules {
		s.upsertRule(rule, changed)
	}
	batch := s.WatchServer.NotifyBatch()
	s.publishRuleGroups(batch, changed)
	batch.Notify()
}

func (s *Syncer) upsertRule(rule *core.PortalRule, changed map[string]struct{}) {
	state := s.rules.byID[rule.Id]
	if state == nil {
		state = new(_IndexedRule)
		s.rules.byID[rule.Id] = state
	} else {
		old := &state.rule
		if old.EntryId == rule.EntryId && old.RouteSiteName == rule.RouteSiteName &&
			old.RouteType == rule.RouteType && old.MatchPathPrefix == rule.MatchPathPrefix &&
			old.Enabled == rule.Enabled && old.Name == rule.Name {
			payloadChanged := old.RouteRedirectionPattern != rule.RouteRedirectionPattern || old.RoutePathPrefix != rule.RoutePathPrefix
			state.rule = *rule
			if group := s.rules.groups[state.match]; payloadChanged && group != nil && group.winner == state {
				changed[state.match] = struct{}{}
			}
			return
		}
		s.rules.detach(state, changed)
	}

	state.rule = *rule
	s.rules.byEntry.add(rule.EntryId, state)
	if rule.RouteType == core.PortalRuleRouteTypeSite {
		s.rules.bySite.add(rule.RouteSiteName, state)
	}
	s.indexRule(state, changed)
}

func (s *Syncer) RemovePortalRule(rule *core.PortalRule) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()
	state := s.rules.byID[rule.Id]
	if state == nil {
		return
	}
	changed := make(map[string]struct{})
	s.rules.detach(state, changed)
	delete(s.rules.byID, rule.Id)
	batch := s.WatchServer.NotifyBatch()
	s.publishRuleGroups(batch, changed)
	batch.Notify()
}

func (idx *_PortalRuleIndex) detach(state *_IndexedRule, changed map[string]struct{}) {
	idx.byEntry.remove(state.rule.EntryId, state)
	if state.rule.RouteType == core.PortalRuleRouteTypeSite {
		idx.bySite.remove(state.rule.RouteSiteName, state)
	}
	idx.leaveGroup(state, changed)
}

func (idx *_PortalRuleIndex) leaveGroup(state *_IndexedRule, changed map[string]struct{}) {
	if state.match == "" {
		return
	}
	group := idx.groups[state.match]
	delete(group.members, state)
	if group.winner == state {
		group.winner = nil
	}
	changed[state.match] = struct{}{}
	state.match = ""
}

func (s *Syncer) indexRule(state *_IndexedRule, changed map[string]struct{}) {
	rule := &state.rule
	match := core.PortalRuleMatchKey(rule, s.portalEntriesById[rule.EntryId], s.portalSiteOfRuleLocked(rule))
	if match == state.match {
		if group := s.rules.groups[match]; group != nil && group.winner == state {
			changed[match] = struct{}{}
		}
		return
	}
	s.rules.leaveGroup(state, changed)
	if match == "" {
		return
	}
	group := s.rules.groups[match]
	if group == nil {
		group = &_RuleGroup{members: make(map[*_IndexedRule]struct{})}
		s.rules.groups[match] = group
	}
	// A nil winner in a nonempty group is re-elected once after the batch.
	if len(group.members) == 0 || (group.winner != nil && rule.Name < group.winner.rule.Name) {
		group.winner = state
	}
	group.members[state] = struct{}{}
	state.match = match
	changed[match] = struct{}{}
}

func (s *Syncer) reindexRules(refs map[*_IndexedRule]struct{}, changed map[string]struct{}) {
	for state := range refs {
		s.indexRule(state, changed)
	}
}

func (s *Syncer) publishRuleGroups(batch *watchserver.NotifyBatch, changed map[string]struct{}) {
	// Watch keys use rule names. A rule can move between two changed groups
	// while keeping its name, so resolve withdrawals before recording winners.
	updates := make(map[string]*_IndexedRule, len(changed))
	for match := range changed {
		if name := s.rules.groups[match].publishedName; name != "" {
			updates[name] = nil
		}
	}
	for match := range changed {
		group := s.rules.groups[match]
		if len(group.members) == 0 {
			delete(s.rules.groups, match)
			continue
		}
		if group.winner == nil {
			for member := range group.members {
				if group.winner == nil || member.rule.Name < group.winner.rule.Name {
					group.winner = member
				}
			}
		}
		group.publishedName = group.winner.rule.Name
		updates[group.publishedName] = group.winner
	}
	for _, name := range vmap.SortedKeys(updates) {
		key := watched.FormatPortalRuleKey(name)
		if state := updates[name]; state != nil {
			rule := &state.rule
			batch.Set(key, vcode.MustMarshalJsonS(s.toWatchedPortalRule(rule, s.portalSiteOfRuleLocked(rule))))
		} else {
			batch.Delete(key)
		}
	}
}
