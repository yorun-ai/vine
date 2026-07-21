package seeder

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vfile"
	"gorm.io/gorm"
)

func TestSeederLoadsYAMLIntoSQLiteRepos(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, redisServer := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
appConfigs:
  - name: feature.flag
    value: '{"enabled":true}'
portalSites:
  - name: admin@demo.app
    type: WEBGW
    actorSkelName: demo.AdminActor
    actorVia: client
    webName: demo.AdminWeb
    builtIn: true
portalRules:
  - name: admin
    scheme: https
    host: demo.local
    port: 443
    pathPrefix: /admin
    targetType: SITE
    siteName: admin@demo.app
    redirectionPattern: ""
    builtIn: true
portalCerts:
  - name: admin-cert
    issuer: manual
    domains:
      - admin.local
    publicKeyBase64: pub
    privateKeyBase64: pri
    validFrom: 2026-01-01T00:00:00Z
    validTo: 2027-01-01T00:00:00Z
`))

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	item, ok := configRepo.GetItemByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, item.Value)
	assert.Equal(t, 1, item.Version)

	rule, ok := ruleRepo.GetRuleByName("admin")
	require.True(t, ok)
	assert.Equal(t, "/admin", rule.PathPrefix)
	assert.Equal(t, "admin@demo.app", rule.SiteName)
	assert.False(t, rule.BuiltIn)

	entry, ok := entryRepo.GetEntryByName("admin@demo.app")
	require.True(t, ok)
	assert.Equal(t, "demo.AdminActor", entry.ActorSkelName)
	assert.Equal(t, "demo.AdminWeb", entry.WebName)
	assert.False(t, entry.BuiltIn)

	cert, ok := certRepo.GetCertByName("admin-cert")
	require.True(t, ok)
	assert.Equal(t, []string{"admin.local"}, cert.Domains)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), cert.ValidFrom)

	_, ok = redisServer.Get(redised.FormatPortalRuleKey("admin"))
	assert.True(t, ok)
	_, ok = redisServer.Get(redised.FormatPortalSiteKey("admin@demo.app"))
	assert.True(t, ok)
	_, ok = redisServer.Get(redised.FormatPortalCertKey("admin-cert"))
	assert.True(t, ok)
	assert.True(t, metadataRepo.IsSeeded())
}

func testSyncer(redisServer *redisserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}

func newTestSeederFlag(seedYAMLPath string) *flag.Flag {
	flags := &flag.Flag{
		SourceType:   flag.SourceSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
		SeedYAMLPath: seedYAMLPath,
	}
	flags.Normalize(true)
	return flags
}

func TestSeederMarksSeededWhenSeedYAMLPathIsEmpty(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)

	seeder := &Seeder{
		Flag:          newTestSeederFlag(""),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}

	seeder.DIInit()

	assert.True(t, metadataRepo.IsSeeded())
	_, ok := configRepo.GetItemByName("feature.flag")
	assert.False(t, ok)

	rule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "http", rule.Scheme)
	assert.Equal(t, "", rule.Host)
	assert.Equal(t, 7099, rule.Port)
	assert.Equal(t, "/api", rule.PathPrefix)
	entry, ok := entryRepo.GetEntryByName(DashboardRpcCoreEntry.Name)
	require.True(t, ok)
	assert.Equal(t, DashboardRpcCoreEntry.ActorSkelName, entry.ActorSkelName)
}

func TestSeederSkipsEmptySeedYAMLPathWhenApplied(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(""),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	_, ok := configRepo.GetItemByName("feature.flag")
	assert.False(t, ok)
}

func TestSeederSkipsWhenSeedYAMLWasApplied(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
appConfigs:
  - name: feature.flag
    value: '{"enabled":false}'
`))

	configRepo.SaveItem(&core.AppConfig{
		Name:    "feature.flag",
		Value:   `{"enabled":true}`,
		Version: 7,
	})
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	item, ok := configRepo.GetItemByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, item.Value)
	assert.Equal(t, 7, item.Version)
}

func TestSeederAppliesOverrideItemsWhenSeedYAMLWasApplied(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
appConfigs:
  - name: feature.flag
    value: '{"enabled":false}'
    override: true
  - name: feature.keep
    value: '{"enabled":false}'
portalSites:
  - name: admin@demo.app
    type: WEBGW
    actorSkelName: demo.AdminActor
    actorVia: client
    webName: demo.AdminWeb
    override: true
portalRules:
  - name: admin
    scheme: https
    host: demo.local
    port: 443
    pathPrefix: /admin
    targetType: SITE
    siteName: admin@demo.app
    redirectionPattern: ""
    override: true
portalCerts:
  - name: admin-cert
    issuer: manual
    domains:
      - admin.local
    publicKeyBase64: pub
    privateKeyBase64: pri
    validFrom: 2026-01-01T00:00:00Z
    validTo: 2027-01-01T00:00:00Z
    override: true
`))

	configRepo.SaveItem(&core.AppConfig{Name: "feature.flag", Value: `{"enabled":true}`, Version: 7})
	configRepo.SaveItem(&core.AppConfig{Name: "feature.keep", Value: `{"enabled":true}`, Version: 3})
	entryRepo.SaveEntry(&core.PortalSite{Name: "admin@demo.app", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "old.Actor", ActorVia: "client", WebName: "old.Web"})
	ruleRepo.SaveRule(&core.PortalRule{Name: "admin", Scheme: "http", Port: 80, PathPrefix: "/old", TargetType: "SITE", SiteName: "old-site"})
	certRepo.SaveCert(&core.PortalCert{Name: "admin-cert", Issuer: "old", Domains: []string{"old.local"}, PublicKeyBase64: "old-pub", PrivateKeyBase64: "old-pri"})
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	item, ok := configRepo.GetItemByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":false}`, item.Value)
	assert.Equal(t, 8, item.Version)
	kept, ok := configRepo.GetItemByName("feature.keep")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, kept.Value)
	assert.Equal(t, 3, kept.Version)

	entry, ok := entryRepo.GetEntryByName("admin@demo.app")
	require.True(t, ok)
	assert.Equal(t, "demo.AdminActor", entry.ActorSkelName)
	assert.Equal(t, "demo.AdminWeb", entry.WebName)
	rule, ok := ruleRepo.GetRuleByName("admin")
	require.True(t, ok)
	assert.Equal(t, "https", rule.Scheme)
	assert.Equal(t, "/admin", rule.PathPrefix)
	cert, ok := certRepo.GetCertByName("admin-cert")
	require.True(t, ok)
	assert.Equal(t, "manual", cert.Issuer)
	assert.Equal(t, []string{"admin.local"}, cert.Domains)
}

func TestSeederRejectsSeedYAMLConflictingWithBuiltInItems(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalSites:
  - name: vine.hub.AdminActor-client-rpc
    type: WEBGW
portalRules:
  - name: vine.hub.admin-api
    scheme: http
`))
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}

	assert.Panics(t, seeder.DIInit)
}

func TestSeederRefreshesDashboardWhenSeeded(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	ruleRepo.SaveRule(&core.PortalRule{
		Name:       dashboardApiRuleName,
		Scheme:     "https",
		Host:       "hub.example.com",
		Port:       8088,
		PathPrefix: "/old-api",
		TargetType: "SITE",
		SiteName:   "old-entry",
		BuiltIn:    true,
	})
	entryRepo.SaveEntry(&core.PortalSite{
		Name:          DashboardRpcCoreEntry.Name,
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "old.Actor",
		ActorVia:      "client",
		BuiltIn:       true,
	})
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(""),
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	rule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", rule.Scheme)
	assert.Equal(t, "hub.example.com", rule.Host)
	assert.Equal(t, 8088, rule.Port)
	assert.Equal(t, "/old-api", rule.PathPrefix)
	assert.Equal(t, DashboardRpcCoreEntry.Name, rule.SiteName)

	entry, ok := entryRepo.GetEntryByName(DashboardRpcCoreEntry.Name)
	require.True(t, ok)
	assert.Equal(t, DashboardRpcCoreEntry.ActorSkelName, entry.ActorSkelName)
}

func TestSeederAppliesExplicitDashboardURLToExistingDashboardRules(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	ruleRepo.SaveRule(&core.PortalRule{
		Name:       dashboardApiRuleName,
		Scheme:     "http",
		Port:       7099,
		PathPrefix: "/api",
		TargetType: "SITE",
		SiteName:   DashboardRpcCoreEntry.Name,
		BuiltIn:    true,
	})
	ruleRepo.SaveRule(&core.PortalRule{
		Name:       dashboardWebRuleName,
		Scheme:     "http",
		Port:       7099,
		PathPrefix: "/",
		TargetType: "SITE",
		SiteName:   DashboardWebCoreEntry.Name,
		BuiltIn:    true,
	})
	metadataRepo.MarkSeeded()

	flags := &flag.Flag{SourceType: flag.SourceSQLite, DBSQLiteFile: "/tmp/hub.sqlite", DashboardURLRaw: "https://hub.example.com:8443/admin"}
	flags.Normalize(true)

	seeder := &Seeder{
		Flag:          flags,
		AppConfigRepo: configRepo,
		MetadataRepo:  metadataRepo,
		Logger:        logger.NewLogger(logger.GlobalOption()),
		RuleRepo:      ruleRepo,
		CertRepo:      certRepo,
		EntryRepo:     entryRepo,
	}
	seeder.DIInit()

	apiRule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", apiRule.Scheme)
	assert.Equal(t, "hub.example.com", apiRule.Host)
	assert.Equal(t, 8443, apiRule.Port)
	assert.Equal(t, "/api", apiRule.PathPrefix)

	webRule, ok := ruleRepo.GetRuleByName(dashboardWebRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", webRule.Scheme)
	assert.Equal(t, "hub.example.com", webRule.Host)
	assert.Equal(t, 8443, webRule.Port)
	assert.Equal(t, "/admin", webRule.PathPrefix)
}

func newTestSeederRepos(t *testing.T) (*repo.DBAppConfigRepo, *repo.DBPortalRuleRepo, *repo.DBPortalCertRepo, *repo.DBPortalSiteRepo, *repo.DBMetadataRepo, *redisserver.Server) {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hub.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&model.AppConfig{}, &model.PortalRule{}, &model.PortalCert{}, &model.PortalSite{}, &model.Metadata{}))

	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	return &repo.DBAppConfigRepo{
			Dao:    &model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](gdb)},
			Syncer: testSyncer(redisServer),
		}, &repo.DBPortalRuleRepo{
			Dao:    &model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](gdb)},
			Syncer: testSyncer(redisServer),
		}, &repo.DBPortalCertRepo{
			Dao:    &model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](gdb)},
			Syncer: testSyncer(redisServer),
		}, &repo.DBPortalSiteRepo{
			Dao:        &model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](gdb)},
			SchemaRepo: new(schema.MemorySchemaRepo),
			Syncer:     testSyncer(redisServer),
		}, &repo.DBMetadataRepo{
			Dao: &model.MetadataDao{Dao: rdb.NewDao[*model.Metadata](gdb)},
		}, redisServer
}
