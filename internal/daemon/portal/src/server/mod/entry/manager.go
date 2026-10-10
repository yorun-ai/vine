package entry

import (
	"context"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

const (
	defaultHTTPEntryPort  = 80
	defaultHTTPSEntryPort = 443
	entryShutdownTimeout  = 10 * time.Second
)

type Manager struct {
	app.BaseModule

	Context     context.Context       `inject:""`
	SiteManager *site.Manager         `inject:""`
	Vault       *vault.Vault          `inject:""`
	Watch       hubapiwatch.ClientOps `inject:""`

	mutex            sync.Mutex
	entryRulesByName map[string]watched.PortalRule
	entriesByKey     map[_Key]*_Entry
	started          bool
}

func (e *Manager) DIInit() {
	e.entryRulesByName = map[string]watched.PortalRule{}
	e.entriesByKey = map[_Key]*_Entry{}
	e.loadPortalRules()
}

func (e *Manager) AfterAppStart() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.started = true
	err := e.reconcileEntriesLocked()
	if err == nil {
		for _, entry := range e.entriesByKey {
			if err = entry.Start(); err != nil {
				break
			}
		}
	}
	if err != nil {
		for _, entry := range e.entriesByKey {
			entry.Stop()
		}
		e.started = false
		vpre.Panicf("portal entry startup failed: %v", err)
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
	valuesByKey, subscription := e.Watch.LoadListAndSubscribe(e.Context, watched.FormatPortalRulePrefix(), e.handlePortalRuleEvent)

	e.mutex.Lock()
	defer e.mutex.Unlock()

	for key, value := range valuesByKey {
		rule := vcode.MustUnmarshalJsonS[*watched.PortalRule](value)
		e.entryRulesByName[key] = *rule
	}
	if err := e.reconcileEntriesLocked(); err != nil {
		entryLogger.Error("vine.portal listener update failed", "error", err)
	}
	subscription.Start()
}

func (e *Manager) handlePortalRuleEvent(event hubapiwatch.Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if event.Kind == hubapiwatch.EventKindDelete {
		delete(e.entryRulesByName, event.Key)
		if err := e.reconcileEntriesLocked(); err != nil {
			entryLogger.Error("vine.portal listener update failed", "error", err)
		}
		return
	}

	rule := vcode.MustUnmarshalJsonS[*watched.PortalRule](event.Value)
	e.entryRulesByName[event.Key] = *rule
	if err := e.reconcileEntriesLocked(); err != nil {
		entryLogger.Error("vine.portal listener update failed", "error", err)
	}
}

// reconcileEntriesLocked applies a listener change only after every new binding
// succeeds. Removed listeners are stopped first to allow wildcard/IP transitions;
// on failure, new bindings are closed and the previous listeners are restored.
func (e *Manager) reconcileEntriesLocked() error {
	nextRules := e.buildRulesLocked()
	removedEntries, rulesToUpdate, rulesToCreate := e.diffEntriesLocked(nextRules)
	for _, entry := range removedEntries {
		entry.Stop()
	}
	created := map[_Key]*_Entry{}
	for key, rules := range rulesToCreate {
		entry := newEntry(key.scheme, key.port, e.Vault)
		entry.listenIP = key.listenIP
		entry.SetOrUpdateRules(rules)
		if e.started {
			if err := entry.Start(); err != nil {
				for _, previous := range created {
					previous.Stop()
				}
				for _, previous := range removedEntries {
					if restoreErr := previous.Start(); restoreErr != nil {
						entryLogger.Error("vine.portal listener restore failed", "error", restoreErr)
					}
				}
				return err
			}
		}
		created[key] = entry
	}
	for _, entry := range removedEntries {
		delete(e.entriesByKey, entry.Key())
	}
	for key, rules := range rulesToUpdate {
		e.entriesByKey[key].SetOrUpdateRules(rules)
	}
	for key, entry := range created {
		e.entriesByKey[key] = entry
	}
	return nil
}

func (e *Manager) buildRulesLocked() map[_Key][]*_Rule {
	rulesByKey := map[_Key][]*_Rule{}
	for _, item := range e.entryRulesByName {
		if rule, ok := newRule(item, e.SiteManager); ok {
			for _, key := range rule.Keys() {
				rulesByKey[key] = append(rulesByKey[key], rule)
			}
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
