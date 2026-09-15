package admin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type _MaintenanceServiceAppConfigRepo struct {
	items map[string]*core.AppConfig
}

func (r *_MaintenanceServiceAppConfigRepo) List() []*core.AppConfig {
	var items []*core.AppConfig
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

func (r *_MaintenanceServiceAppConfigRepo) ListSlots() []*core.AppConfig {
	return r.List()
}

func (r *_MaintenanceServiceAppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	return r.GetByName(name)
}

func (r *_MaintenanceServiceAppConfigRepo) GetById(id int) (*core.AppConfig, bool) {
	for _, item := range r.items {
		if item.Id == id {
			return item, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServiceAppConfigRepo) GetByName(name string) (*core.AppConfig, bool) {
	item, ok := r.items[name]
	return item, ok
}

func (r *_MaintenanceServiceAppConfigRepo) Save(item *core.AppConfig) {
	r.items[item.Name] = item
}

func (r *_MaintenanceServiceAppConfigRepo) Remove(id int) bool {
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
	service := &MaintenanceApiServiceServerImpl{}

	preview := service.PreviewSeedYaml("unknown_items: []")

	if preview.Items == nil {
		t.Fatal("expected non-nil preview items")
	}
	if len(preview.Items) != 0 {
		t.Fatalf("unexpected preview items: %d", len(preview.Items))
	}
}

func TestMaintenanceServiceEmptySeedDocumentPreviewsAndAppliesNothing(t *testing.T) {
	service := &MaintenanceApiServiceServerImpl{}

	require.Empty(t, service.PreviewSeedYaml("").Items)
	require.Empty(t, service.ApplySeedYaml("", []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "demo.Config"}}).Items)
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
	service := &MaintenanceApiServiceServerImpl{
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
	}

	service.ApplySeedYaml(`
appConfigs:
  - name: demo.Config
    value: '{"old":false}'
`, []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "demo.Config"}})

	item, ok := configRepo.GetByName("demo.Config")
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
		core.DashboardWebRuleName: {
			Id:              1,
			Name:            core.DashboardWebRuleName,
			MatchScheme:     "http",
			MatchPort:       80,
			MatchPathPrefix: "/old",
			RouteType:       "SITE",
			RouteSiteName:   "old-site",
			BuiltIn:         true,
		},
	}}
	service := &MaintenanceApiServiceServerImpl{
		SiteCore: newTestPortalSiteCore(entryRepo),
		RuleCore: newTestPortalRuleCore(ruleRepo),
		CertCore: newTestPortalCertCore(),
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
  - name: vine.hub.dashboard-web
    scheme: https
    host: demo.local
    port: 443
    pathPrefix: /admin
    targetType: SITE
    siteName: admin@demo.app
    redirectionPattern: ""
    builtIn: false
`

	preview := service.PreviewSeedYaml(strings.ReplaceAll(content, core.DashboardWebRuleName, "preview"))
	require.Panics(t, func() { service.PreviewSeedYaml(content) })
	for _, item := range preview.Items {
		for _, field := range item.Fields {
			if field.Name == "builtIn" {
				t.Fatalf("did not expect builtIn field in preview: %#v", item)
			}
		}
	}

	require.Panics(t, func() {
		service.ApplySeedYaml(content, []skeled.SeedItemSelection{
			{Kind: seedKindPortalSite, Name: "admin@demo.app"},
			{Kind: seedKindPortalRule, Name: core.DashboardWebRuleName},
		})
	})
	require.Equal(t, "old.Web", entryRepo.items["admin@demo.app"].WebName, "preflight must reject before any writes")
	require.Equal(t, "/old", ruleRepo.items[core.DashboardWebRuleName].MatchPathPrefix)

	entry, ok := entryRepo.GetByName("admin@demo.app")
	if !ok {
		t.Fatal("expected portal site")
	}
	if !entry.BuiltIn {
		t.Fatal("expected existing portal site built-in flag to be preserved")
	}
	rule, ok := ruleRepo.GetByName(core.DashboardWebRuleName)
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

func (r *_MaintenanceServicePortalSiteRepo) List() []*core.PortalSite {
	items := make([]*core.PortalSite, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalSite, b *core.PortalSite) bool {
		return a.Id < b.Id
	})
}

func (r *_MaintenanceServicePortalSiteRepo) GetById(id int) (*core.PortalSite, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServicePortalSiteRepo) GetByName(name string) (*core.PortalSite, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_MaintenanceServicePortalSiteRepo) Save(entry *core.PortalSite) {
	value := *entry
	r.items[value.Name] = &value
}

func (r *_MaintenanceServicePortalSiteRepo) Remove(id int) bool {
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

func (r *_MaintenanceServicePortalRuleRepo) List() []*core.PortalRule {
	items := make([]*core.PortalRule, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return vslice.SortBy(items, func(a *core.PortalRule, b *core.PortalRule) bool {
		return a.Id < b.Id
	})
}

func (r *_MaintenanceServicePortalRuleRepo) GetById(id int) (*core.PortalRule, bool) {
	for _, item := range r.items {
		if item.Id == id {
			value := *item
			return &value, true
		}
	}
	return nil, false
}

func (r *_MaintenanceServicePortalRuleRepo) GetByName(name string) (*core.PortalRule, bool) {
	item, ok := r.items[name]
	if !ok {
		return nil, false
	}
	value := *item
	return &value, true
}

func (r *_MaintenanceServicePortalRuleRepo) Save(rule *core.PortalRule) {
	value := *rule
	r.items[value.Name] = &value
}

func (r *_MaintenanceServicePortalRuleRepo) Remove(id int) bool {
	for name, item := range r.items {
		if item.Id == id {
			delete(r.items, name)
			return true
		}
	}
	return false
}

func TestMaintenanceTargetPathSeedRoundTrip(t *testing.T) {
	repo := &_MaintenanceServicePortalRuleRepo{items: map[string]*core.PortalRule{}}
	service := &MaintenanceApiServiceServerImpl{RuleCore: newTestPortalRuleCore(repo)}
	payload := service.parseSeed("portalRules:\n  - name: mapped\n    scheme: http\n    targetType: SITE\n    siteName: web\n    pathPrefix: /api\n    targetPath: /internal/\n")
	rule := payload.PortalRules[0]
	if rule.RoutePathPrefix != "/internal" {
		t.Fatalf("unexpected target path: %q", rule.RoutePathPrefix)
	}
	service.applyPortalRules(payload.PortalRules, map[_SeedSelectionKey]struct{}{{kind: seedKindPortalRule, name: "mapped"}: {}})
	stored, ok := repo.GetByName("mapped")
	if !ok || stored.RoutePathPrefix != "/internal" {
		t.Fatalf("target path not persisted: %+v", stored)
	}
	fields := stored.SeedFields()
	if fields["routePathPrefix"] != "/internal" {
		t.Fatalf("target path not exported: %v", fields)
	}
}

func TestMaintenanceRuleFieldNames(t *testing.T) {
	service := &MaintenanceApiServiceServerImpl{RuleCore: newTestPortalRuleCore(&_MaintenanceServicePortalRuleRepo{items: map[string]*core.PortalRule{}})}
	payload := service.parseSeed("portalRules:\n  - name: example\n    matchScheme: http\n    routeType: SITE\n    routeSiteName: web\n    routePathPrefix: /internal")
	require.Equal(t, "http", payload.PortalRules[0].MatchScheme)
	require.Equal(t, "/internal", payload.PortalRules[0].RoutePathPrefix)
	require.Panics(t, func() { service.parseSeed("portalRules:\n  - scheme: http\n    routeType: SITE") })
}

func TestMaintenanceSeedPreviewComparesEverySeedFieldOfASite(t *testing.T) {
	entryRepo := &_MaintenanceServicePortalSiteRepo{items: map[string]*core.PortalSite{
		"demo.site": {
			Id:            1,
			Name:          "demo.site",
			Type:          core.PortalSiteTypeRPCGW,
			ActorSkelName: "demo.Actor",
			ActorVia:      "client",
			Cors:          core.PortalCors{Mode: core.PortalCorsModeDisabled},
		},
	}}
	service := &MaintenanceApiServiceServerImpl{SiteCore: newTestPortalSiteCore(entryRepo)}

	preview := service.PreviewSeedYaml(`
portalSites:
  - name: demo.site
    type: RPCGW
    actorSkelName: demo.Actor
    actorVia: client
    cors:
      mode: STRICT
      allowedOrigins: ["https://demo.local"]
`)

	require.Len(t, preview.Items, 1)
	fields := map[string]skeled.SeedFieldDiff{}
	for _, field := range preview.Items[0].Fields {
		fields[field.Name] = field
	}
	require.False(t, fields["type"].Changed)
	require.True(t, fields["corsMode"].Changed)
	require.Equal(t, "DISABLED", fields["corsMode"].CurrentValue)
	require.Equal(t, "STRICT", fields["corsMode"].SeedValue)
	require.True(t, fields["corsOrigins"].Changed)
	require.Equal(t, "[]", fields["corsOrigins"].CurrentValue)
	require.Equal(t, `["https://demo.local"]`, fields["corsOrigins"].SeedValue)
}

func TestMaintenanceUsesDomainValidationForBothYAMLVocabularies(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		content := "portalRules:\n  - name: invalid\n    matchScheme: http\n    matchPort: -1\n    routeType: SITE\n    routeSiteName: web\n"
		if legacy {
			content = strings.NewReplacer("matchScheme:", "scheme:", "matchPort:", "port:", "routeType:", "targetType:", "routeSiteName:", "siteName:").Replace(content)
		}
		repo := &_MaintenanceServicePortalRuleRepo{items: map[string]*core.PortalRule{}}
		service := &MaintenanceApiServiceServerImpl{RuleCore: newTestPortalRuleCore(repo)}
		require.Panics(t, func() { service.PreviewSeedYaml(content) })
		require.Panics(t, func() {
			service.ApplySeedYaml(content, []skeled.SeedItemSelection{{Kind: seedKindPortalRule, Name: "invalid"}})
		})
		require.Empty(t, repo.items)
	}
}

func TestMaintenancePreflightsSitesAndCertificatesBeforeWriting(t *testing.T) {
	for name, invalid := range map[string]string{
		"site":        "portalSites:\n  - name: invalid\n    type: WEBGW\n    actorSkelName: demo.Actor\n    actorVia: client\n",
		"certificate": "portalCerts:\n  - name: invalid\n    publicKeyBase64: invalid\n",
	} {
		t.Run(name, func(t *testing.T) {
			configs := &_MaintenanceServiceAppConfigRepo{items: map[string]*core.AppConfig{}}
			target := &MaintenanceApiServiceServerImpl{AppConfigCore: &core.AppConfigCore{AppConfigRepo: configs}, SiteCore: &core.PortalSiteCore{}, CertCore: &core.PortalCertCore{}}
			content := "appConfigs:\n  - name: pending\n    value: test\n" + invalid
			require.Panics(t, func() { target.PreviewSeedYaml(content) })
			require.Panics(t, func() {
				target.ApplySeedYaml(content, []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "pending"}})
			})
			require.Empty(t, configs.items)
		})
	}
}

func TestMaintenanceStructuredAppConfigYAML(t *testing.T) {
	configRepo := new(_MaintenanceServiceAppConfigRepo{
		items: map[string]*core.AppConfig{},
	})
	service := new(MaintenanceApiServiceServerImpl{
		AppConfigCore: new(core.AppConfigCore{AppConfigRepo: configRepo}),
	})
	const content = `appConfigs:
  - name: demo.Config
    value:
      statuses: {EAST: ACTIVE}
      openingDate: 2026-09-13
`
	preview := service.PreviewSeedYaml(content)
	require.Len(t, preview.Items, 1)
	service.ApplySeedYaml(content, []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "demo.Config"}})
	item, ok := configRepo.GetByName("demo.Config")
	require.True(t, ok)
	require.Equal(t, `{"openingDate":"2026-09-13","statuses":{"EAST":"ACTIVE"}}`, item.Value)
	version := item.Version
	service.ApplySeedYaml(content, []skeled.SeedItemSelection{{Kind: seedKindAppConfig, Name: "demo.Config"}})
	require.Equal(t, version, configRepo.items["demo.Config"].Version)
}

func TestMaintenanceSeedRejectsYAMLReferencesBeforePreviewOrApply(t *testing.T) {
	for _, content := range []string{
		"&seed {appConfigs: []}",
		"unknown: &unused text",
		"<<: {appConfigs: []}",
		"appConfigs: [{name: demo.Config, value: &value {enabled: true}}]",
		"portalSites: [{name: site, urls: &urls []}]",
		"portalRules: [{name: rule, <<: {routeType: SITE}}]",
		"portalCerts: [{name: &name cert, cert: *name}]",
	} {
		t.Run(content, func(t *testing.T) {
			service := new(MaintenanceApiServiceServerImpl)
			for _, call := range []func(){
				func() { service.PreviewSeedYaml(content) },
				func() { service.ApplySeedYaml(content, nil) },
			} {
				func() {
					defer func() {
						failure := recover()
						require.NotNil(t, failure)
						require.Contains(t, fmt.Sprint(failure), "not supported")
						require.Contains(t, fmt.Sprint(failure), "line ")
					}()
					call()
				}()
			}
		})
	}
}
