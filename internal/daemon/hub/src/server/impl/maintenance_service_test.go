package impl

import (
	"testing"

	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type _MaintenanceServiceAppConfigRepo struct {
	items map[string]*core.AppConfig
}

func (r *_MaintenanceServiceAppConfigRepo) ListItems() []*core.AppConfig {
	var items []*core.AppConfig
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

func (r *_MaintenanceServiceAppConfigRepo) GetItemById(id int) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return item, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServiceAppConfigRepo) GetItemByName(name string) (*core.AppConfig, bool) {
	item, ok := r.items[name]
	return item, ok
}

func (r *_MaintenanceServiceAppConfigRepo) SaveItem(item *core.AppConfig) {
	r.items[item.Name] = item
}

func (r *_MaintenanceServiceAppConfigRepo) RemoveItem(id int) bool {
	for name, item := range r.items {
		if item.Id != id {
			continue
		}
		delete(r.items, name)
		return true
	}
	return false
}

func TestMaintenanceServicePreviewSeedYamlReturnsEmptyItems(t *testing.T) {
	service := &MaintenanceServiceServerImpl{}

	preview := service.PreviewSeedYaml("unknown_items: []")

	if preview.Items == nil {
		t.Fatal("expected non-nil preview items")
	}
	if len(preview.Items) != 0 {
		t.Fatalf("unexpected preview items: %d", len(preview.Items))
	}
}

func TestMaintenanceServiceApplySeedYamlUpdatesSelectedItem(t *testing.T) {
	configRepo := &_MaintenanceServiceAppConfigRepo{items: map[string]*core.AppConfig{
		"demo.Config": {
			Id:      3,
			Name:    "demo.Config",
			Value:   `{"old":true}`,
			Version: 1,
		},
	}}
	service := &MaintenanceServiceServerImpl{
		AppConfigRepo: configRepo,
	}

	service.ApplySeedYaml(`
appConfigs:
  - name: demo.Config
    value: '{"old":false}'
`, []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "demo.Config"}})

	item, ok := configRepo.GetItemByName("demo.Config")
	if !ok {
		t.Fatal("expected config item")
	}
	if item.Value != `{"old":false}` {
		t.Fatalf("unexpected value: %s", item.Value)
	}
	if item.Version != 2 {
		t.Fatalf("unexpected version: %d", item.Version)
	}
}

func TestMaintenanceServiceSeedYamlDoesNotExposeVineField(t *testing.T) {
	entryRepo := &_MaintenanceServicePortalSiteRepo{items: map[string]*core.PortalSite{
		"admin@demo.app": {
			Id:            1,
			Name:          "admin@demo.app",
			Type:          core.PortalSiteTypeWEBGW,
			ActorSkelName: "old.Actor",
			ActorVia:      "client",
			WebName:       "old.Web",
			BuiltIn:       true,
		},
	}}
	ruleRepo := &_MaintenanceServicePortalRuleRepo{items: map[string]*core.PortalRule{
		"admin": {
			Id:         1,
			Name:       "admin",
			Scheme:     "http",
			Port:       80,
			PathPrefix: "/old",
			TargetType: "SITE",
			SiteName:   "old-site",
			BuiltIn:    true,
		},
	}}
	service := &MaintenanceServiceServerImpl{
		EntryRepo: entryRepo,
		RuleRepo:  ruleRepo,
	}
	content := `
portalSites:
  - name: admin@demo.app
    type: WEBGW
    actorSkelName: demo.AdminActor
    actorVia: client
    webName: demo.AdminWeb
    builtIn: false
portalRules:
  - name: admin
    scheme: https
    host: demo.local
    port: 443
    pathPrefix: /admin
    targetType: SITE
    siteName: admin@demo.app
    redirectionPattern: ""
    builtIn: false
`

	preview := service.PreviewSeedYaml(content)
	for _, item := range preview.Items {
		for _, field := range item.Fields {
			if field.Name == "builtIn" {
				t.Fatalf("did not expect builtIn field in preview: %#v", item)
			}
		}
	}

	service.ApplySeedYaml(content, []skeled.SeedItemSelection{
		{Kind: seedKindPortalSite, Name: "admin@demo.app"},
		{Kind: seedKindPortalRule, Name: "admin"},
	})

	entry, ok := entryRepo.GetEntryByName("admin@demo.app")
	if !ok {
		t.Fatal("expected portal site")
	}
	if !entry.BuiltIn {
		t.Fatal("expected existing portal site built-in flag to be preserved")
	}
	rule, ok := ruleRepo.GetRuleByName("admin")
	if !ok {
		t.Fatal("expected portal rule")
	}
	if !rule.BuiltIn {
		t.Fatal("expected existing portal rule built-in flag to be preserved")
	}
}

type _MaintenanceServicePortalSiteRepo struct {
	items map[string]*core.PortalSite
}

func (r *_MaintenanceServicePortalSiteRepo) ListEntries() []core.PortalSite {
	items := make([]core.PortalSite, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, *item)
	}
	return vslice.SortBy(items, func(a core.PortalSite, b core.PortalSite) bool {
		return a.Id < b.Id
	})
}

func (r *_MaintenanceServicePortalSiteRepo) GetEntryById(id int) (*core.PortalSite, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServicePortalSiteRepo) GetEntryByName(name string) (*core.PortalSite, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_MaintenanceServicePortalSiteRepo) SaveEntry(entry *core.PortalSite) {
	value := *entry
	r.items[value.Name] = &value
}

func (r *_MaintenanceServicePortalSiteRepo) RemoveEntry(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}

type _MaintenanceServicePortalRuleRepo struct {
	items map[string]*core.PortalRule
}

func (r *_MaintenanceServicePortalRuleRepo) ListRules() []core.PortalRule {
	items := make([]core.PortalRule, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, *item)
	}
	return vslice.SortBy(items, func(a core.PortalRule, b core.PortalRule) bool {
		return a.Id < b.Id
	})
}

func (r *_MaintenanceServicePortalRuleRepo) GetRuleById(id int) (*core.PortalRule, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServicePortalRuleRepo) GetRuleByName(name string) (*core.PortalRule, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_MaintenanceServicePortalRuleRepo) SaveRule(rule *core.PortalRule) {
	value := *rule
	r.items[value.Name] = &value
}

func (r *_MaintenanceServicePortalRuleRepo) RemoveRule(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}
