package entry

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
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
	Watch       *hubwatch.Client `inject:""`

	mutex                  sync.Mutex
	entryRulesByName       map[string]watched.PortalRule
	webMountPathsBySiteKey map[string]string
	entriesByKey           map[_Key]*_Entry
	started                bool
}

func (e *Manager) DIInit() {
	e.entryRulesByName = map[string]watched.PortalRule{}
	e.entriesByKey = map[_Key]*_Entry{}
	e.webMountPathsBySiteKey = map[string]string{}
	e.loadWebMountPaths()
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

// Entry owns the effective match and rewrite prefixes. Site changes rebuild
// rules from their original configuration, including precedence by path length.
func (e *Manager) loadWebMountPaths() {
	values, subscription := e.Watch.LoadListAndSubscribe(e.Context, watched.FormatPortalSitePrefix(), e.handlePortalSiteEvent)
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for key, value := range values {
		e.webMountPathsBySiteKey[key] = portalSiteMountPath(value)
	}
	subscription.Start()
}

func portalSiteMountPath(value string) string {
	site := vcode.MustUnmarshalJsonS[watched.PortalSite](value)
	if site.Type == "WEBGW" && site.WebgwConfig != nil {
		return site.WebgwConfig.MountPath
	}
	return ""
}

func (e *Manager) handlePortalSiteEvent(event hubapiwatch.Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	oldPath := e.webMountPathsBySiteKey[event.Key]
	nextPath := ""
	if event.Kind == hubapiwatch.EventKindDelete {
		delete(e.webMountPathsBySiteKey, event.Key)
	} else {
		nextPath = portalSiteMountPath(event.Value)
		e.webMountPathsBySiteKey[event.Key] = nextPath
	}
	if oldPath != nextPath {
		e.reconcileEntriesLocked()
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
	e.reconcileEntriesLocked()
	subscription.Start()
}

func (e *Manager) handlePortalRuleEvent(event hubapiwatch.Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if event.Kind == hubapiwatch.EventKindDelete {
		delete(e.entryRulesByName, event.Key)
		e.reconcileEntriesLocked()
		return
	}

	rule := vcode.MustUnmarshalJsonS[*watched.PortalRule](event.Value)
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
		if item.RouteType == routeTypeSite {
			if mountPath := e.webMountPathsBySiteKey[watched.FormatPortalSiteKey(item.RouteSiteName)]; mountPath != "" {
				// Trim for segment matching and rewriting; a declared root path
				// still overrides configured prefixes because the check is above.
				item.MatchPathPrefix = strings.TrimRight(mountPath, "/")
				item.RoutePathPrefix = item.MatchPathPrefix
				if item.MatchPathPrefix == "" {
					item.MatchPathPrefix = "/"
				}
			}
		}
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
