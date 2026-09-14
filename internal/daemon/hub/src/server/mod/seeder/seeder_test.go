package seeder

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
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
	for _, inline := range []bool{false, true} {
		t.Run(fmt.Sprintf("inline=%t", inline), func(t *testing.T) {
			configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, redisServer := newTestSeederRepos(t)
			seedPath := filepath.Join(t.TempDir(), "hub.yaml")
			seedYAML := `
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
    issuer: ignored
    domains:
      - ignored.local
    publicKeyBase64: ` + testSeederCertificate(t) + `
    privateKeyBase64: pri
    validFrom: 2026-01-01T00:00:00Z
    validTo: 2027-01-01T00:00:00Z
`

			flags := newTestSeederFlag(seedPath)
			if inline {
				flags = new(flag.Flag{SeedHubData: seedYAML})
				flags.Normalize(true)
			} else {
				require.NoError(t, vfile.WriteString(seedPath, seedYAML))
			}

			seeder := &Seeder{
				Flag:          flags,
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
				MetadataRepo:  metadataRepo,
				Logger:        logger.New("vine:test"),
				RuleRepo:      ruleRepo,
				RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
				CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
				SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
			}
			seeder.DIInit()

			item, ok := configRepo.GetItemByName("feature.flag")
			require.True(t, ok)
			assert.Equal(t, `{"enabled":true}`, item.Value)
			assert.Equal(t, 1, item.Version)

			rule, ok := ruleRepo.GetRuleByName("admin")
			require.True(t, ok)
			assert.Equal(t, "/admin", rule.MatchPathPrefix)
			assert.Equal(t, "admin@demo.app", rule.RouteSiteName)
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
		})
	}
}

func testSyncer(redisServer *redisserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}

func newTestSeederFlag(seedYAMLPath string) *flag.Flag {
	flags := &flag.Flag{
		Store:           flag.StoreSQLite,
		DBSQLiteFile:    "/tmp/hub.sqlite",
		SeedHubDataFile: seedYAMLPath,
	}
	flags.Normalize(true)
	return flags
}

func newTestSeederMTLSFlag() *flag.Flag {
	flags := &flag.Flag{
		MTLS: mtls.Files{
			CAFile:   "ca.pem",
			CertFile: "cert.pem",
			KeyFile:  "key.pem",
		},
		Store:        flag.StoreSQLite,
		DBSQLiteFile: "/tmp/hub.sqlite",
	}
	flags.Normalize(true)
	return flags
}

func TestSeederMarksSeededWhenSeedHubDataFileIsEmpty(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)

	seeder := &Seeder{
		Flag:          newTestSeederFlag(""),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}

	seeder.DIInit()

	assert.True(t, metadataRepo.IsSeeded())
	_, ok := configRepo.GetItemByName("feature.flag")
	assert.False(t, ok)

	rule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "http", rule.MatchScheme)
	assert.Equal(t, "", rule.MatchHost)
	assert.Equal(t, 7099, rule.MatchPort)
	assert.Equal(t, "/api", rule.MatchPathPrefix)
	entry, ok := entryRepo.GetEntryByName(DashboardRpcCoreEntry.Name)
	require.True(t, ok)
	assert.Equal(t, DashboardRpcCoreEntry.ActorSkelName, entry.ActorSkelName)
}

func TestSeederUsesHTTPSForDefaultDashboardWithMTLS(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seeder := &Seeder{
		Flag:          newTestSeederMTLSFlag(),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}

	seeder.DIInit()

	apiRule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", apiRule.MatchScheme)
	assert.Equal(t, 7099, apiRule.MatchPort)
	webRule, ok := ruleRepo.GetRuleByName(dashboardWebRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", webRule.MatchScheme)
	assert.Equal(t, 7099, webRule.MatchPort)
}

func TestSeederMigratesLegacyDashboardDefaultsToHTTPSWithMTLS(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	ruleRepo.SaveRule(&core.PortalRule{
		Name:            dashboardApiRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/api",
		RouteType:       "SITE",
		RouteSiteName:   DashboardRpcCoreEntry.Name,
		BuiltIn:         true,
	})
	ruleRepo.SaveRule(&core.PortalRule{
		Name:            dashboardWebRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/",
		RouteType:       "SITE",
		RouteSiteName:   DashboardWebCoreEntry.Name,
		BuiltIn:         true,
	})
	metadataRepo.MarkSeeded()
	seeder := &Seeder{
		Flag:          newTestSeederMTLSFlag(),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}

	seeder.DIInit()

	apiRule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", apiRule.MatchScheme)
	webRule, ok := ruleRepo.GetRuleByName(dashboardWebRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", webRule.MatchScheme)
}

func TestSeederPreservesCustomDashboardAccessWithMTLSDefault(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	for _, rule := range []*core.PortalRule{
		{
			Name:            dashboardApiRuleName,
			MatchScheme:     "https",
			MatchHost:       "hub.example.com",
			MatchPort:       8443,
			MatchPathPrefix: "/custom-api",
			RouteType:       "SITE",
			RouteSiteName:   DashboardRpcCoreEntry.Name,
			BuiltIn:         true,
		},
		{
			Name:            dashboardWebRuleName,
			MatchScheme:     "https",
			MatchHost:       "hub.example.com",
			MatchPort:       8443,
			MatchPathPrefix: "/custom",
			RouteType:       "SITE",
			RouteSiteName:   DashboardWebCoreEntry.Name,
			BuiltIn:         true,
		},
	} {
		ruleRepo.SaveRule(rule)
	}
	metadataRepo.MarkSeeded()
	seeder := &Seeder{
		Flag:          newTestSeederMTLSFlag(),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}

	seeder.DIInit()

	apiRule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", apiRule.MatchScheme)
	assert.Equal(t, "hub.example.com", apiRule.MatchHost)
	assert.Equal(t, 8443, apiRule.MatchPort)
	assert.Equal(t, "/custom-api", apiRule.MatchPathPrefix)
	webRule, ok := ruleRepo.GetRuleByName(dashboardWebRuleName)
	require.True(t, ok)
	assert.Equal(t, "/custom", webRule.MatchPathPrefix)
}

func TestSeederSkipsEmptySeedHubDataFileWhenApplied(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(""),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
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
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}
	seeder.DIInit()

	item, ok := configRepo.GetItemByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, item.Value)
	assert.Equal(t, 7, item.Version)
	// After metadata is set, none of the deployment inputs are read again.
	seeder.Flag.SeedHubVarsFile = filepath.Join(t.TempDir(), "missing-vars.yaml")
	seeder.Flag.SeedHubSourceFile = filepath.Join(t.TempDir(), "missing-source.yaml")
	require.NotPanics(t, seeder.DIInit)
	require.NoError(t, vfile.WriteString(seedPath, "appConfigs: [{name: feature.flag, value: '${missing}'}]"))
	require.NotPanics(t, seeder.DIInit)
	require.NoError(t, vfile.WriteString(seedPath, "[invalid YAML"))
	require.NotPanics(t, seeder.DIInit)
	seeder.Flag.SeedHubDataFile = filepath.Join(t.TempDir(), "missing-seed.yaml")
	require.NotPanics(t, seeder.DIInit)

}

func TestSeederPreservesAllItemsWhenAlreadySeeded(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
appConfigs:
  - name: feature.flag
    value: '{"enabled":false}'
  - name: feature.keep
    value: '{"enabled":false}'
portalSites:
  - name: admin@demo.app
    type: WEBGW
    actorSkelName: demo.AdminActor
    actorVia: client
    webName: demo.AdminWeb
portalRules:
  - name: admin
    scheme: https
    host: demo.local
    port: 443
    pathPrefix: /admin
    targetType: SITE
    siteName: admin@demo.app
    redirectionPattern: ""
portalCerts:
  - name: admin-cert
    issuer: ignored
    domains:
      - ignored.local
    publicKeyBase64: `+testSeederCertificate(t)+`
    privateKeyBase64: pri
    validFrom: 2026-01-01T00:00:00Z
    validTo: 2027-01-01T00:00:00Z
`))

	configRepo.SaveItem(&core.AppConfig{Name: "feature.flag", Value: `{"enabled":true}`, Version: 7})
	configRepo.SaveItem(&core.AppConfig{Name: "feature.keep", Value: `{"enabled":true}`, Version: 3})
	entryRepo.SaveEntry(&core.PortalSite{Name: "admin@demo.app", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "old.Actor", ActorVia: "client", WebName: "old.Web"})
	ruleRepo.SaveRule(&core.PortalRule{Name: "admin", MatchScheme: "http", MatchPort: 80, MatchPathPrefix: "/old", RouteType: "SITE", RouteSiteName: "old-site"})
	certRepo.SaveCert(&core.PortalCert{Name: "admin-cert", Issuer: "old", Domains: []string{"old.local"}, PublicKeyBase64: "old-pub", PrivateKeyBase64: "old-pri"})
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}
	seeder.DIInit()

	item, ok := configRepo.GetItemByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, item.Value)
	assert.Equal(t, 7, item.Version)
	kept, ok := configRepo.GetItemByName("feature.keep")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, kept.Value)
	assert.Equal(t, 3, kept.Version)

	entry, ok := entryRepo.GetEntryByName("admin@demo.app")
	require.True(t, ok)
	assert.Equal(t, "old.Actor", entry.ActorSkelName)
	assert.Equal(t, "old.Web", entry.WebName)
	rule, ok := ruleRepo.GetRuleByName("admin")
	require.True(t, ok)
	assert.Equal(t, "http", rule.MatchScheme)
	assert.Equal(t, "/old", rule.MatchPathPrefix)
	cert, ok := certRepo.GetCertByName("admin-cert")
	require.True(t, ok)
	assert.Equal(t, "old", cert.Issuer)
	assert.Equal(t, []string{"old.local"}, cert.Domains)
}

func TestSeederRejectsSeedYAMLConflictingWithBuiltInItems(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalSites:
  - name: vine.hub.admin.AdminActor-client-rpc
    type: WEBGW
portalRules:
  - name: vine.hub.admin-api
    scheme: http
`))

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}

	assert.Panics(t, seeder.DIInit)
}

func TestSeederRefreshesDashboardWhenSeeded(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	ruleRepo.SaveRule(&core.PortalRule{
		Name:            dashboardApiRuleName,
		MatchScheme:     "https",
		MatchHost:       "hub.example.com",
		MatchPort:       8088,
		MatchPathPrefix: "/old-api",
		RouteType:       "SITE",
		RouteSiteName:   "old-entry",
		BuiltIn:         true,
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
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}
	seeder.DIInit()

	rule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", rule.MatchScheme)
	assert.Equal(t, "hub.example.com", rule.MatchHost)
	assert.Equal(t, 8088, rule.MatchPort)
	assert.Equal(t, "/old-api", rule.MatchPathPrefix)
	assert.Equal(t, DashboardRpcCoreEntry.Name, rule.RouteSiteName)

	entry, ok := entryRepo.GetEntryByName(DashboardRpcCoreEntry.Name)
	require.True(t, ok)
	assert.Equal(t, DashboardRpcCoreEntry.ActorSkelName, entry.ActorSkelName)
}

func TestSeederAppliesExplicitDashboardURLToExistingDashboardRules(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	ruleRepo.SaveRule(&core.PortalRule{
		Name:            dashboardApiRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/api",
		RouteType:       "SITE",
		RouteSiteName:   DashboardRpcCoreEntry.Name,
		BuiltIn:         true,
	})
	ruleRepo.SaveRule(&core.PortalRule{
		Name:            dashboardWebRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/",
		RouteType:       "SITE",
		RouteSiteName:   DashboardWebCoreEntry.Name,
		BuiltIn:         true,
	})
	metadataRepo.MarkSeeded()

	flags := &flag.Flag{Store: flag.StoreSQLite, DBSQLiteFile: "/tmp/hub.sqlite", DashboardURLRaw: "https://hub.example.com:8443/admin"}
	flags.Normalize(true)

	seeder := &Seeder{
		Flag:          flags,
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		RuleRepo:      ruleRepo,
		RuleCore:      &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      &core.PortalSiteCore{PortalSiteRepo: entryRepo},
	}
	seeder.DIInit()

	apiRule, ok := ruleRepo.GetRuleByName(dashboardApiRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", apiRule.MatchScheme)
	assert.Equal(t, "hub.example.com", apiRule.MatchHost)
	assert.Equal(t, 8443, apiRule.MatchPort)
	assert.Equal(t, "/api", apiRule.MatchPathPrefix)

	webRule, ok := ruleRepo.GetRuleByName(dashboardWebRuleName)
	require.True(t, ok)
	assert.Equal(t, "https", webRule.MatchScheme)
	assert.Equal(t, "hub.example.com", webRule.MatchHost)
	assert.Equal(t, 8443, webRule.MatchPort)
	assert.Equal(t, "/admin", webRule.MatchPathPrefix)
}

func newTestSeederRepos(t *testing.T) (*repo.DBAppConfigRepo, *repo.DBPortalRuleRepo, *repo.DBPortalCertRepo, *repo.DBPortalSiteRepo, *repo.DBMetadataRepo, *redisserver.Server) {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hub.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&model.Metadata{}))
	(&model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](gdb)}).InitSchema()
	(&model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](gdb)}).InitSchema()
	(&model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](gdb)}).InitSchema()
	(&model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](gdb)}).InitSchema()

	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	return &repo.DBAppConfigRepo{
		Dao:    &model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](gdb)},
		Syncer: testSyncer(redisServer),
		Access: new(configaccess.Access),
	}, &repo.DBPortalRuleRepo{
		Dao:    &model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](gdb)},
		Syncer: testSyncer(redisServer),
		Access: new(configaccess.Access),
	}, &repo.DBPortalCertRepo{
		Dao:    &model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](gdb)},
		Syncer: testSyncer(redisServer),
		Access: new(configaccess.Access),
	}, &repo.DBPortalSiteRepo{
		Dao:        &model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](gdb)},
		SchemaRepo: new(schema.MemorySchemaRepo),
		Syncer:     testSyncer(redisServer),
		Access:     new(configaccess.Access),
	}, &repo.DBMetadataRepo{
		Dao: &model.MetadataDao{Dao: rdb.NewDao[*model.Metadata](gdb)},
	}, redisServer
}

func TestSeederPreflightsAllRulesBeforeImporting(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
			content := `appConfigs:
  - name: pending
    value: test
portalRules:
  - name: valid
    matchScheme: http
    routeType: SITE
    routeSiteName: web
  - name: invalid
    matchScheme: ftp
    routeType: SITE
    routeSiteName: web
`
			if legacy {
				content = strings.NewReplacer("matchScheme:", "scheme:", "routeType:", "targetType:", "routeSiteName:", "siteName:").Replace(content)
			}
			path := filepath.Join(t.TempDir(), "hub.yaml")
			require.NoError(t, vfile.WriteString(path, content))
			seeder := &Seeder{Flag: newTestSeederFlag(path), Logger: logger.New("vine:test"),
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo}, RuleRepo: ruleRepo, RuleCore: &core.PortalRuleCore{PortalRuleRepo: ruleRepo},
				CertCore: &core.PortalCertCore{PortalCertRepo: certRepo},
				SiteCore: &core.PortalSiteCore{PortalSiteRepo: entryRepo}, MetadataRepo: metadataRepo}
			require.Panics(t, seeder.DIInit)
			_, exists := configRepo.GetItemByName("pending")
			require.False(t, exists)
			_, exists = ruleRepo.GetRuleByName("valid")
			require.False(t, exists)
			require.False(t, metadataRepo.IsSeeded())
		})
	}
}

func testSeederCertificate(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "manual"}, DNSNames: []string{"admin.local"}, NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
}

func TestSeederPreflightsSitesAndCertificatesBeforeWriting(t *testing.T) {
	for name, invalid := range map[string]string{
		"site":        "portalSites:\n  - name: invalid\n    type: WEBGW\n    actorSkelName: demo.Actor\n    actorVia: client\n",
		"certificate": "portalCerts:\n  - name: invalid\n    publicKeyBase64: invalid\n",
	} {
		t.Run(name, func(t *testing.T) {
			configs, rules, certs, sites, metadata, _ := newTestSeederRepos(t)
			path := filepath.Join(t.TempDir(), "hub.yaml")
			require.NoError(t, vfile.WriteString(path, "appConfigs:\n  - name: pending\n    value: test\n"+invalid))
			target := &Seeder{Flag: newTestSeederFlag(path), Logger: logger.New("vine:test"), MetadataRepo: metadata, RuleRepo: rules,
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configs}, RuleCore: &core.PortalRuleCore{PortalRuleRepo: rules},
				CertCore: &core.PortalCertCore{PortalCertRepo: certs}, SiteCore: &core.PortalSiteCore{PortalSiteRepo: sites}}
			require.Panics(t, target.DIInit)
			_, exists := configs.GetItemByName("pending")
			require.False(t, exists)
			require.False(t, metadata.IsSeeded())
		})
	}
}

func TestSeederRejectsInvalidInlineYAML(t *testing.T) {
	for _, source := range []string{" ", "null", "appConfigs: [", "[]"} {
		t.Run(source, func(t *testing.T) {
			seeder := new(Seeder{Flag: new(flag.Flag{SeedHubData: source})})
			require.Panics(t, seeder.loadSeedYAML)
		})
	}
}

func TestSeederAcceptsEmptyInlineMapping(t *testing.T) {
	seeder := new(Seeder{Flag: new(flag.Flag{SeedHubData: "{}"})})
	require.NotPanics(t, seeder.loadSeedYAML)
	require.NotNil(t, seeder.payload)
}

func TestSeederPersistsSourcesByEntityAndClearsOnRemoval(t *testing.T) {
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, _ := newTestSeederRepos(t)
	template := "appConfigs:\n- name: second\n  value: '${value}'\n- name: first\n  value: {}\n"
	source := fmt.Sprintf("version: 1\nseedSha256: %x\nfields:\n  /appConfigs/0/value:\n    source: app/default\n    define: domain/booker\n    override: app/default\n", sha256.Sum256([]byte(template)))
	s := new(Seeder{Flag: new(flag.Flag{SeedHubData: template, SeedHubSource: source, SeedHubVarsFile: writeSeedHubVarsFile(t, `value: '"resolved"'`)}),
		AppConfigCore: new(core.AppConfigCore{AppConfigRepo: configRepo}),
		RuleCore:      new(core.PortalRuleCore{PortalRuleRepo: ruleRepo}),
		SiteCore:      new(core.PortalSiteCore{PortalSiteRepo: siteRepo}),
		CertCore:      new(core.PortalCertCore{PortalCertRepo: certRepo}),
		RuleRepo:      ruleRepo, MetadataRepo: metadataRepo, Logger: logger.New("seed-source-test"),
	})
	s.Flag.Normalize(true)
	s.DIInit()
	item, ok := configRepo.GetItemByName("second")
	require.True(t, ok)
	require.Equal(t, `"resolved"`, item.Value)
	require.Equal(t, core.FieldSource{Source: "app/default", Define: "domain/booker", Override: "app/default", Variables: []string{"value"}, Template: new(skel.JSON(`"${value}"`)), Bindings: []core.FieldSourceBinding{{Variable: "value", Reference: "${value}", Value: skel.JSON(`"\"resolved\""`)}}}, item.FieldSources["/value"])
	other, ok := configRepo.GetItemByName("first")
	require.True(t, ok)
	require.Empty(t, other.FieldSources)
	// Metadata is loaded from the database, not retained by the Seeder instance.
	reread, ok := configRepo.GetItemById(item.Id)
	require.True(t, ok)
	require.Equal(t, item.FieldSources, reread.FieldSources)
	s.AppConfigCore.Update(item.Id, core.AppConfigUpdate{Value: new(`"resolved"`)})
	updated, ok := configRepo.GetItemById(item.Id)
	require.True(t, ok)
	require.Equal(t, "hub", updated.FieldSources["/value"].Override)
	require.Empty(t, updated.FieldSources["/value"].Variables)
	require.True(t, s.AppConfigCore.Remove(item.Id))
	var storeCount int64
	require.NoError(t, configRepo.Dao.GormDB().Table("field_source").Where("kind = ? AND entity_id = ?", "app_config", item.Id).Count(&storeCount).Error)
	require.Zero(t, storeCount)
	require.Empty(t, s.AppConfigCore.Save(core.AppConfig{Name: "third", Value: "{}"}).FieldSources)
}

func TestSeederPersistsConfigValueKeySources(t *testing.T) {
	configs, rules, certs, sites, metadata, _ := newTestSeederRepos(t)
	template := "appConfigs:\n- name: user.AuthConfig\n  value:\n    accessTokenTTL: ${ttl}\n    refreshTokenTTL: 168h\n    nested: {enabled: false}\n"
	source := fmt.Sprintf(`version: 1
seedSha256: %x
fields:
  /appConfigs/0/value/accessTokenTTL: {source: profile/dev, define: domain/user, override: profile/dev}
  /appConfigs/0/value/refreshTokenTTL: {source: domain/user, define: domain/user}
  /appConfigs/0/value/nested: {source: app/default, define: domain/user, override: app/default}
`, sha256.Sum256([]byte(template)))
	target := new(Seeder{
		Flag:          new(flag.Flag{SeedHubData: template, SeedHubSource: source, SeedHubVarsFile: writeSeedHubVarsFile(t, "ttl: 2h")}),
		AppConfigCore: new(core.AppConfigCore{AppConfigRepo: configs}),
		RuleCore:      new(core.PortalRuleCore{PortalRuleRepo: rules}),
		SiteCore:      new(core.PortalSiteCore{PortalSiteRepo: sites}),
		CertCore:      new(core.PortalCertCore{PortalCertRepo: certs}),
		RuleRepo:      rules, MetadataRepo: metadata, Logger: logger.New("seed-key-sources-test"),
	})
	target.Flag.Normalize(true)
	target.DIInit()
	item, ok := configs.GetItemByName("user.AuthConfig")
	require.True(t, ok)
	require.JSONEq(t, `{"accessTokenTTL":"2h","refreshTokenTTL":"168h","nested":{"enabled":false}}`, item.Value)
	require.Equal(t, core.FieldSources{
		"/value/accessTokenTTL":  {Source: "profile/dev", Define: "domain/user", Override: "profile/dev", Variables: []string{"ttl"}, Template: new(skel.JSON(`"${ttl}"`)), Bindings: []core.FieldSourceBinding{{Variable: "ttl", Reference: "${ttl}", Value: skel.JSON(`"2h"`)}}},
		"/value/refreshTokenTTL": {Source: "domain/user", Define: "domain/user"},
		"/value/nested":          {Source: "app/default", Define: "domain/user", Override: "app/default"},
	}, item.FieldSources)
	reread, ok := configs.GetItemById(item.Id)
	require.True(t, ok)
	require.Equal(t, item.FieldSources, reread.FieldSources)
}

func writeSeedHubVarsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vars.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}
