package admin

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

// The Admin services the Dashboard calls read and write the same entities, so
// their tests share these map-backed repositories.

// _PortalSiteRepoSpy is a map-backed Portal site repository.
type _PortalSiteRepoSpy struct {
	items map[string]*core.PortalSite
}

func (r *_PortalSiteRepoSpy) List() []*core.PortalSite {
	items := make([]*core.PortalSite, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalSite, b *core.PortalSite) bool {
		return a.Id < b.Id
	})
}

func (r *_PortalSiteRepoSpy) GetById(id int) (*core.PortalSite, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_PortalSiteRepoSpy) GetByName(name string) (*core.PortalSite, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_PortalSiteRepoSpy) Save(entry *core.PortalSite) {
	value := *entry
	r.items[value.Name] = &value
}

func (r *_PortalSiteRepoSpy) Remove(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}

// _PortalRuleRepoSpy is a map-backed Portal rule repository.
type _NamedPortalRuleRepoSpy struct {
	items map[string]*core.PortalRule
}

func (r *_NamedPortalRuleRepoSpy) List() []*core.PortalRule {
	items := make([]*core.PortalRule, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalRule, b *core.PortalRule) bool {
		return a.Id < b.Id
	})
}

func (r *_NamedPortalRuleRepoSpy) GetById(id int) (*core.PortalRule, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_NamedPortalRuleRepoSpy) GetByName(name string) (*core.PortalRule, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_NamedPortalRuleRepoSpy) Save(rule *core.PortalRule) {
	value := *rule
	r.items[value.Name] = &value
}

func (r *_NamedPortalRuleRepoSpy) Remove(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}
