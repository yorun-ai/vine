package entry

import (
	"context"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

const entryShutdownTimeout = 10 * time.Second

type Manager struct {
	app.BaseModule

	Context     context.Context       `inject:""`
	SiteManager *site.Manager         `inject:""`
	Vault       *vault.Vault          `inject:""`
	Watch       hubapiwatch.ClientOps `inject:""`

	mutex              sync.Mutex
	entryRulesByName   map[string]watched.PortalRule
	entryConfigsByName map[string]watched.PortalEntry
	entriesByKey       map[_Key]*_Entry
	started            bool
}

func (e *Manager) DIInit() {
	e.entryRulesByName = map[string]watched.PortalRule{}
	e.entriesByKey = map[_Key]*_Entry{}
	e.entryConfigsByName = map[string]watched.PortalEntry{}
	e.loadPortalEntries()
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

func (e *Manager) loadPortalEntries() {
	values, subscription := e.Watch.LoadListAndSubscribe(e.Context, watched.FormatPortalEntryPrefix(), e.handlePortalEntryEvent)
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for key, value := range values {
		e.entryConfigsByName[key] = vcode.MustUnmarshalJsonS[watched.PortalEntry](value)
	}
	subscription.Start()
}
func (e *Manager) handlePortalEntryEvent(event hubapiwatch.Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if event.Kind == hubapiwatch.EventKindDelete {
		delete(e.entryConfigsByName, event.Key)
	} else {
		e.entryConfigsByName[event.Key] = vcode.MustUnmarshalJsonS[watched.PortalEntry](event.Value)
	}
	if err := e.reconcileEntriesLocked(); err != nil {
		entryLogger.Error("vine.portal listener update failed", "error", err)
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
		config, ok := e.entryConfigsByName[watched.FormatPortalEntryKey(item.EntryName)]
		if !ok {
			continue
		}
		for _, access := range entryTransports(config) {
			if rule, ok := newRule(item, config, access.scheme, access.port, e.SiteManager); ok {
				for _, key := range rule.Keys() {
					rulesByKey[key] = append(rulesByKey[key], rule)
				}
			}
		}
	}
	for _, config := range e.entryConfigsByName {
		for _, access := range entryTransports(config) {
			rule := &_Rule{name: config.Name, listenIPs: config.ListenIPs, matchScheme: access.scheme, matchPort: access.port, matchHost: config.Host}
			for _, key := range rule.Keys() {
				if _, ok := rulesByKey[key]; !ok {
					rulesByKey[key] = nil
				}
				if access.scheme == spec.SchemeHTTP && config.Http.AutoHTTPS {
					rule.autoHTTPSPort = config.Http.HttpsPort
					rule.certificateVault = e.Vault
					rulesByKey[key] = append(rulesByKey[key], rule)
				}
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

type _Transport struct {
	scheme spec.Scheme
	port   int
}

func entryTransports(config watched.PortalEntry) []_Transport {
	transports := []_Transport{}
	if config.Http.HttpEnabled {
		transports = append(transports, _Transport{scheme: spec.SchemeHTTP, port: config.Http.HttpPort})
	}
	if config.Http.HttpsEnabled {
		transports = append(transports, _Transport{scheme: spec.SchemeHTTPS, port: config.Http.HttpsPort})
	}
	return transports
}
