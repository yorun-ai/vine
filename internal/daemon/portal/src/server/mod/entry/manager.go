package entry

import (
	"context"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/util/vcode"
)

const (
	defaultHTTPEntryPort  = 80
	defaultHTTPSEntryPort = 443
	entryShutdownTimeout  = 10 * time.Second
)

type Manager struct {
	app.BaseModule

	Context     context.Context  `inject:""`
	SiteManager *site.Manager    `inject:""`
	Vault       *vault.Vault     `inject:""`
	Redis       *hubredis.Client `inject:""`

	mutex            sync.Mutex
	entryRulesByName map[string]redised.PortalRule
	entriesByKey     map[_Key]*_Entry
	started          bool
}

func (e *Manager) DIInit() {
	e.entryRulesByName = map[string]redised.PortalRule{}
	e.entriesByKey = map[_Key]*_Entry{}
	e.loadPortalRules()
}

func (e *Manager) AfterAppStart() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.started = true
	e.reconcileEntriesLocked()
	for _, entry := range e.entriesByKey {
		entry.Start()
	}
}

func (e *Manager) AfterAppStop() {
	e.mutex.Lock()
	entries := e.entriesByKey
	e.entriesByKey = map[_Key]*_Entry{}
	e.started = false
	e.mutex.Unlock()

	for _, entry := range entries {
		entry.Stop()
	}
}

func (e *Manager) loadPortalRules() {
	valuesByKey := e.Redis.LoadListAndSubscribe(e.Context, redised.FormatPortalRulePrefix(), e.handlePortalRuleEvent)

	e.mutex.Lock()
	defer e.mutex.Unlock()

	for key, value := range valuesByKey {
		rule := vcode.MustUnmarshalJsonS[*redised.PortalRule](value)
		e.entryRulesByName[key] = *rule
	}
	e.reconcileEntriesLocked()
}

func (e *Manager) handlePortalRuleEvent(event hubapiredis.Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if event.Kind == hubapiredis.EventKindDelete {
		delete(e.entryRulesByName, event.Key)
		e.reconcileEntriesLocked()
		return
	}

	rule := vcode.MustUnmarshalJsonS[*redised.PortalRule](event.Value)
	e.entryRulesByName[event.Key] = *rule
	e.reconcileEntriesLocked()
}

func (e *Manager) reconcileEntriesLocked() {
	nextRules := e.buildRulesLocked()
	removedEntries, rulesToUpdate, rulesToCreate := e.diffEntriesLocked(nextRules)

	for _, entry := range removedEntries {
		delete(e.entriesByKey, entry.Key())
		entry.Stop()
	}

	for key, rules := range rulesToUpdate {
		e.entriesByKey[key].SetOrUpdateRules(rules)
	}

	for key, rules := range rulesToCreate {
		entry := newEntry(key.scheme, key.port, e.Vault)
		entry.SetOrUpdateRules(rules)
		e.entriesByKey[key] = entry
		if e.started {
			entry.Start()
		}
	}
}

func (e *Manager) buildRulesLocked() map[_Key][]*_Rule {
	rulesByKey := map[_Key][]*_Rule{}
	for _, item := range e.entryRulesByName {
		if rule, ok := newRule(item, e.SiteManager); ok {
			key := rule.Key()
			rulesByKey[key] = append(rulesByKey[key], rule)
		}
	}
	return rulesByKey
}

func (e *Manager) diffEntriesLocked(nextRules map[_Key][]*_Rule) ([]*_Entry, map[_Key][]*_Rule, map[_Key][]*_Rule) {
	removedEntries := make([]*_Entry, 0)
	rulesToUpdate := map[_Key][]*_Rule{}
	rulesToCreate := map[_Key][]*_Rule{}

	for key, entry := range e.entriesByKey {
		rules, ok := nextRules[key]
		if ok {
			rulesToUpdate[key] = rules
			continue
		}
		removedEntries = append(removedEntries, entry)
	}

	for key, rules := range nextRules {
		if _, ok := e.entriesByKey[key]; ok {
			continue
		}
		rulesToCreate[key] = rules
	}

	return removedEntries, rulesToUpdate, rulesToCreate
}
