package site

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/runtime"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/rpcgw"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/webgw"
	"go.yorun.ai/vine/util/vcode"
)

const (
	siteTypeRpcgw = "RPCGW"
	siteTypeWebgw = "WEBGW"
)

type Manager struct {
	app.BaseModule

	Context context.Context  `inject:""`
	App     runtime.App      `inject:""`
	Redis   *hubredis.Client `inject:""`
	Access  *access.Access   `inject:""`
	Epmgr   *epmgr.Manager   `inject:""`

	mutex       sync.RWMutex
	sitesByKey  map[string]spec.Site
	sitesByName map[string]spec.Site
}

func (m *Manager) DIInit() {
	m.sitesByKey = map[string]spec.Site{}
	m.sitesByName = map[string]spec.Site{}
	m.loadSites()
}

func (m *Manager) Site(siteName string) (spec.Site, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	site, ok := m.sitesByName[siteName]
	return site, ok
}

func (m *Manager) loadSites() {
	valuesByKey := m.Redis.LoadListAndSubscribe(m.Context, redised.FormatPortalSitePrefix(), m.handleSiteEvent)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range valuesByKey {
		site := m.newSite(m.decodeSite(value))
		m.sitesByKey[key] = site
		m.sitesByName[site.Name()] = site
	}
}

func (m *Manager) handleSiteEvent(event hubapiredis.Event) {
	if event.Kind == hubapiredis.EventKindDelete {
		stopSite(m.removeSite(event.Key))
		return
	}

	config := m.decodeSite(event.Value)
	stopSite(m.updateSite(event.Key, config))
}

func (m *Manager) removeSite(key string) spec.Site {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	toRemove := m.sitesByKey[key]
	if toRemove != nil {
		delete(m.sitesByKey, key)
		delete(m.sitesByName, toRemove.Name())
	}
	return toRemove
}

func (m *Manager) replaceSite(key string, site spec.Site) spec.Site {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	toRemove := m.sitesByKey[key]
	m.sitesByKey[key] = site
	m.sitesByName[site.Name()] = site
	return toRemove
}

func (m *Manager) updateSite(key string, config redised.PortalSite) spec.Site {
	m.mutex.RLock()
	current := m.sitesByKey[key]
	m.mutex.RUnlock()

	if current != nil && current.Update(config) {
		return nil
	}
	return m.replaceSite(key, m.newSite(config))
}

func stopSite(site spec.Site) {
	if site != nil {
		site.Stop()
	}
}

func (m *Manager) decodeSite(value string) redised.PortalSite {
	return *vcode.MustUnmarshalJsonS[*redised.PortalSite](value)
}

func (m *Manager) newSite(config redised.PortalSite) spec.Site {
	switch config.Type {
	case siteTypeRpcgw:
		return rpcgw.New(m.Context, m.App, m.Access, m.Epmgr, config)
	case siteTypeWebgw:
		return webgw.New(m.Context, m.Access, m.Epmgr, config)
	default:
		return newUnknownSite(config.Name, config.Type)
	}
}
