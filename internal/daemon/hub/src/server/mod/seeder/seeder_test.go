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
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vfile"
	"gorm.io/gorm"
)

func TestSeederLoadsYAMLIntoSQLiteRepos(t *testing.T) {
	for _, inline := range []bool{false, true} {
		t.Run(fmt.Sprintf("inline=%t", inline), func(t *testing.T) {
			configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, watchServer := newTestSeederRepos(t)
			entryRepo.SchemaRepo.SaveDomainSchemas("demo", "instance", []*skel.DomainSchema{{
				Domain: "demo", Hash: "mounted-web", Webs: []*skel.WebSchema{{SkelName: "demo.AdminWeb", MountPath: "/mounted"}},
			}})
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
				EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
				RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
				CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
				SiteCore:      newTestSiteCore(entryRepo),
			}
			seeder.DIInit()

			item, ok := configRepo.GetByName("feature.flag")
			require.True(t, ok)
			assert.Equal(t, `{"enabled":true}`, item.Value)
			assert.Equal(t, 1, item.Version)

			rule, ok := ruleRepo.GetByName("admin")
			require.True(t, ok)
			assert.Equal(t, "/admin", rule.MatchPathPrefix)
			assert.Equal(t, "admin@demo.app", rule.RouteSiteName)

			entry, ok := entryRepo.GetByName("admin@demo.app")
			require.True(t, ok)
			assert.Equal(t, "demo.AdminActor", entry.ActorSkelName)
			assert.Equal(t, "demo.AdminWeb", entry.WebName)
			assert.Equal(t, "/mounted", entry.WebMountPath)
			published, exists := watchServer.Get(watched.FormatPortalSiteKey(entry.Name))
			require.True(t, exists)
			assert.Equal(t, "/mounted", vcode.MustUnmarshalJsonS[watched.PortalSite](published).WebgwConfig.MountPath)

			cert, ok := certRepo.GetByName("admin-cert")
			require.True(t, ok)
			assert.Equal(t, []string{"admin.local"}, cert.Domains)
			assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), cert.ValidFrom)

			_, ok = watchServer.Get(watched.FormatPortalRuleKey("admin"))
			assert.True(t, ok)
			_, ok = watchServer.Get(watched.FormatPortalSiteKey("admin@demo.app"))
			assert.True(t, ok)
			_, ok = watchServer.Get(watched.FormatPortalCertKey("admin-cert"))
			assert.True(t, ok)
			assert.True(t, metadataRepo.IsSeeded())
		})
	}
}

func testSyncer(watchServer *watchserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{WatchServer: watchServer}
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

func TestSeederSkipsWhenSeedYAMLWasApplied(t *testing.T) {
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
appConfigs:
  - name: feature.flag
    value: '{"enabled":false}'
`))

	configRepo.Save(&core.AppConfig{
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
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
		RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(entryRepo),
	}
	seeder.DIInit()

	item, ok := configRepo.GetByName("feature.flag")
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

	configRepo.Save(&core.AppConfig{Name: "feature.flag", Value: `{"enabled":true}`, Version: 7})
	configRepo.Save(&core.AppConfig{Name: "feature.keep", Value: `{"enabled":true}`, Version: 3})
	entryRepo.Save(&core.PortalSite{Name: "admin@demo.app", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "old.Actor", ActorVia: "client", WebName: "old.Web"})
	saveTestPortalRule(t, ruleRepo, &core.PortalRule{Name: "admin", MatchPathPrefix: "/old", RouteType: "SITE", RouteSiteName: "old-site"}, core.PortalEntry{Scheme: "http", Port: 80})
	certRepo.Save(&core.PortalCert{Name: "admin-cert", Issuer: "old", Domains: []string{"old.local"}, PublicKeyBase64: "old-pub", PrivateKeyBase64: "old-pri"})
	metadataRepo.MarkSeeded()

	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
		RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(entryRepo),
	}
	seeder.DIInit()

	item, ok := configRepo.GetByName("feature.flag")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, item.Value)
	assert.Equal(t, 7, item.Version)
	kept, ok := configRepo.GetByName("feature.keep")
	require.True(t, ok)
	assert.Equal(t, `{"enabled":true}`, kept.Value)
	assert.Equal(t, 3, kept.Version)

	entry, ok := entryRepo.GetByName("admin@demo.app")
	require.True(t, ok)
	assert.Equal(t, "old.Actor", entry.ActorSkelName)
	assert.Equal(t, "old.Web", entry.WebName)
	rule, ok := ruleRepo.GetByName("admin")
	require.True(t, ok)
	assert.Equal(t, "/old", rule.MatchPathPrefix)
	cert, ok := certRepo.GetByName("admin-cert")
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
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
		RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(entryRepo),
	}

	assert.Panics(t, seeder.DIInit)
}

func TestSeederRejectsRuleReferencingUndeclaredEntry(t *testing.T) {
	// A seed is self-contained: a rule only references an entry the same document
	// declares, so Hub never completes the relationship from stored data.
	for name, content := range map[string]string{
		"no entries": `
portalRules:
  - name: demo.web
    entryName: web
    routeType: SITE
    routeSiteName: demo.Web
`,
		"other entry": `
portalEntries:
  - name: api
    scheme: http
    port: 8099
portalRules:
  - name: demo.web
    entryName: web
    routeType: SITE
    routeSiteName: demo.Web
`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseSeedEntities(content)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `portal rule "demo.web" references portal entry "web" that the seed does not declare`)
		})
	}
}

func TestSeederRejectsAccessRuleWithDeclaredEntries(t *testing.T) {
	// A seed that declares portalEntries names them: Hub never guesses whether a
	// rule meant a declared entry or the access it serves.
	content := `
portalEntries:
  - name: web
    scheme: http
    port: 8099
portalRules:
  - name: demo.web
    matchScheme: http
    matchPort: 8099
    routeType: SITE
    routeSiteName: demo.Web
`
	_, err := parseSeedEntities(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		`portal rule "demo.web" declares an access while the seed declares portalEntries; name the entry with entryName instead`)
}

func TestSeederRejectsMixedPortalRuleStyles(t *testing.T) {
	// One seed document uses one way for a rule to join an entry.
	content := `
portalEntries:
  - name: web
    scheme: http
    port: 8099
portalRules:
  - name: demo.web
    entryName: web
    routeType: SITE
    routeSiteName: demo.Web
  - name: demo.api
    matchScheme: https
    matchPort: 8443
    routeType: SITE
    routeSiteName: demo.Web
`
	_, err := parseSeedEntities(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `portal rule "demo.api" declares an access while portal rule "demo.web" names an entry`)

	// The startup seed fails the same way before Hub writes anything.
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, content))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	require.PanicsWithError(t,
		`portal rule "demo.api" declares an access while portal rule "demo.web" names an entry; a seed declares one or the other, never both`,
		seeder.DIInit)
	assert.False(t, metadataRepo.IsSeeded())
	// The document is rejected before Hub writes any of its rules.
	_, ok := ruleRepo.GetByName("demo.web")
	assert.False(t, ok)
	_, ok = ruleRepo.GetByName("demo.api")
	assert.False(t, ok)
}

func TestSeederPublishesRuleItAggregatesIntoANewEntry(t *testing.T) {
	// A seed written the 0.19.0 way declares the access on the rule and declares
	// no entry. Hub aggregates the access into an entry and keeps publishing the
	// rule, the way a rule that names an entry Hub already stores stays
	// published. A seed declares the switch as disabled, so leaving it at the
	// default keeps the rule published.
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, watchServer := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalSites:
  - name: demo.Web
    type: WEBGW
    actorSkelName: demo.Actor
    actorVia: client
    webName: demo.Web
portalRules:
  - name: demo.web
    disabled: false
    matchScheme: http
    matchPort: 8099
    matchPathPrefix: /
    routeType: SITE
    routeSiteName: demo.Web
`))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	seeder.DIInit()

	rule, ok := ruleRepo.GetByName("demo.web")
	require.True(t, ok)
	entry, ok := ruleRepo.PortalEntryRepo.GetById(rule.EntryId)
	require.True(t, ok)
	assert.Equal(t, "http", entry.Scheme)
	assert.Equal(t, 8099, entry.Port)
	assert.True(t, entry.Enabled)
	_, published := watchServer.Get(watched.FormatPortalRuleKey("demo.web"))
	assert.True(t, published)
}

func TestSeederRejectsStoredSwitchName(t *testing.T) {
	// Hub stores the switch as enabled, and a seed declares it as disabled.
	// Decoding ignores a field the payload does not declare, so a seed that
	// still writes enabled would silently keep the entity published: Hub names
	// the field the seed should declare instead.
	for _, testCase := range []struct {
		name    string
		content string
	}{
		{
			name:    "portal entry",
			content: "portalEntries:\n  - name: web\n    scheme: http\n    port: 8099\n    enabled: false\n",
		},
		{
			name:    "portal site",
			content: "portalSites:\n  - name: demo.Web\n    enabled: false\n",
		},
		{
			name:    "portal rule",
			content: "portalRules:\n  - name: demo.web\n    entryName: web\n    enabled: false\n",
		},
		{
			name:    "portal certificate",
			content: "portalCerts:\n  - name: demo-cert\n    enabled: false\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseSeedEntities(testCase.content)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `declares "enabled"; a seed turns configuration off with "disabled: true"`)
		})
	}
}

func TestSeederRejectsUnknownPortalField(t *testing.T) {
	// Portal sections are fully typed, so a field Hub does not know fails
	// instead of silently leaving the entity at its default.
	for _, testCase := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "portal entry",
			content: "portalEntries:\n  - name: web\n    scheme: http\n    port: 8099\n    webname: demo.Web\n",
			want:    `portal entry "web" declares unknown field "webname"`,
		},
		{
			name:    "portal site",
			content: "portalSites:\n  - name: demo.Web\n    webname: demo.Web\n",
			want:    `portal site "demo.Web" declares unknown field "webname"`,
		},
		{
			name:    "portal site cors",
			content: "portalSites:\n  - name: demo.Web\n    cors:\n      allowOrigins: [https://demo.local]\n",
			want:    `portal site cors "demo.Web" declares unknown field "allowOrigins"`,
		},
		{
			name:    "portal rule",
			content: "portalRules:\n  - name: demo.web\n    matchPathPrefix: /\n    routeSiteNam: demo.Web\n",
			want:    `portal rule "demo.web" declares unknown field "routeSiteNam"`,
		},
		{
			name:    "portal certificate",
			content: "portalCerts:\n  - name: demo-cert\n    issuers: demo\n",
			want:    `portal certificate "demo-cert" declares unknown field "issuers"`,
		},
		{
			name:    "Hub-owned built-in marker",
			content: "portalSites:\n  - name: demo.Web\n    webName: demo.Web\n    builtIn: true\n",
			want:    `portal site "demo.Web" declares "builtIn"; Hub owns the built-in entities, so a seed cannot declare it`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseSeedEntities(testCase.content)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.want)
		})
	}
}

func TestSeederStoresDisabledConfiguration(t *testing.T) {
	// A seed can turn the switch off: Hub stores the entity and does not publish
	// it to Portal.
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, watchServer := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalEntries:
  - name: web
    scheme: http
    port: 8099
    disabled: true
portalSites:
  - name: demo.Web
    type: WEBGW
    actorSkelName: demo.Actor
    actorVia: client
    webName: demo.Web
    disabled: true
portalRules:
  - name: demo.web
    entryName: web
    matchPathPrefix: /
    routeType: SITE
    routeSiteName: demo.Web
    disabled: true
portalCerts:
  - name: demo-cert
    publicKeyBase64: `+testSeederCertificate(t)+`
    privateKeyBase64: pri
    disabled: true
`))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	seeder.DIInit()

	entry, ok := ruleRepo.PortalEntryRepo.GetByName("web")
	require.True(t, ok)
	assert.False(t, entry.Enabled)
	rule, ok := ruleRepo.GetByName("demo.web")
	require.True(t, ok)
	assert.False(t, rule.Enabled)
	site, ok := siteRepo.GetByName("demo.Web")
	require.True(t, ok)
	assert.False(t, site.Enabled)
	cert, ok := certRepo.GetByName("demo-cert")
	require.True(t, ok)
	assert.False(t, cert.Enabled)

	for _, key := range []string{
		watched.FormatPortalRuleKey("demo.web"),
		watched.FormatPortalSiteKey("demo.Web"),
		watched.FormatPortalCertKey("demo-cert"),
	} {
		_, published := watchServer.Get(key)
		assert.False(t, published, "disabled configuration is not published: %s", key)
	}
}

func TestSeederPortalRuleJoinsNamedEntry(t *testing.T) {
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalEntries:
  - name: web
    scheme: http
    port: 8099
portalRules:
  - name: demo.web
    entryName: web
    matchPathPrefix: /
    routeType: SITE
    routeSiteName: demo.Web
`))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	seeder.DIInit()

	// A rule that names an entry joins it instead of declaring an access.
	entry, ok := ruleRepo.PortalEntryRepo.GetByName("web")
	require.True(t, ok)
	rule, ok := ruleRepo.GetByName("demo.web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, rule.EntryId)
	assert.Equal(t, "http", entry.Scheme)
	assert.Equal(t, 8099, entry.Port)
}

func TestSeederRejectsPortalRuleMixingEntryNameAndAccess(t *testing.T) {
	// The entry owns the access, so a rule declares either the entry name or the
	// access the entry serves, never both.
	for _, field := range []string{"matchScheme: http", "matchHost: demo.local", "matchPort: 8099"} {
		t.Run(field, func(t *testing.T) {
			_, err := parseSeedEntities("portalRules:\n  - name: demo.web\n    entryName: web\n    " + field + "\n    routeType: SITE\n    routeSiteName: demo.Web\n")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "entryName cannot be mixed with")
		})
	}
}

func TestSeederAppliesPortalEntriesBeforeRules(t *testing.T) {
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalEntries:
  - name: web
    scheme: http
    port: 8099
  - name: idle
    scheme: https
    host: api.example.com
    port: 8443
portalRules:
  - name: demo.web
    entryName: web
    routeType: SITE
    routeSiteName: demo.Web
`))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	seeder.DIInit()

	// A seed names an entry, and a rule joins the entry it names.
	web, ok := ruleRepo.PortalEntryRepo.GetByName("web")
	require.True(t, ok)
	assert.Equal(t, "http", web.Scheme)
	assert.Equal(t, 8099, web.Port)
	// An entry may route no rule: the seed declares the entry Portal serves.
	idle, ok := ruleRepo.PortalEntryRepo.GetByName("idle")
	require.True(t, ok)
	assert.Equal(t, 8443, idle.Port)

	rule, ok := ruleRepo.GetByName("demo.web")
	require.True(t, ok)
	assert.Equal(t, web.Id, rule.EntryId)
}

func TestSeederRejectsUnnamedPortalEntry(t *testing.T) {
	configRepo, ruleRepo, certRepo, siteRepo, metadataRepo, _ := newTestSeederRepos(t)
	seedPath := filepath.Join(t.TempDir(), "hub.yaml")
	require.NoError(t, vfile.WriteString(seedPath, `
portalEntries:
  - scheme: http
    port: 8099
`))
	seeder := &Seeder{
		Flag:          newTestSeederFlag(seedPath),
		AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("vine:test"),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
		SiteCore:      newTestSiteCore(siteRepo),
	}

	require.PanicsWithError(t, "portal entry name is required type=APPLICATION code=OPERATION_FAILED", seeder.DIInit)
	assert.False(t, metadataRepo.IsSeeded())
}

// newTestEntryCore builds an entry core with the repositories Hub injects.
func newTestEntryCore(entryRepo core.PortalEntryRepo, ruleRepo core.PortalRuleRepo, siteRepo core.PortalSiteRepo) *core.PortalEntryCore {
	return &core.PortalEntryCore{
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  ruleRepo,
		PortalSiteRepo:  siteRepo,
	}
}

// A seed may declare the same request twice: once with the port left unset and
// once with the default port named explicitly. Hub stores both rules, because
// the request they match also depends on the Web mount paths the applications
// register after Hub starts. Hub reports the conflict once it knows those paths;
// a read-only Hub refuses to serve the seed instead of picking one rule.
func TestSeederStoresRulesThatShareOneRequest(t *testing.T) {
	implicit := `
  - name: demo.app
    matchScheme: http
    matchPathPrefix: /app
    routeType: SITE
    routeSiteName: demo.Web
`
	explicit := `
  - name: demo.app-explicit
    matchScheme: http
    matchPort: 80
    matchPathPrefix: /app
    routeType: SITE
    routeSiteName: demo.Web
`
	for name, test := range map[string]struct {
		rules string
	}{
		"implicit first": {rules: implicit + explicit},
		"explicit first": {rules: explicit + implicit},
	} {
		t.Run(name, func(t *testing.T) {
			configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
			seedPath := filepath.Join(t.TempDir(), "hub.yaml")
			require.NoError(t, vfile.WriteString(seedPath, "portalRules:\n"+test.rules))
			seeder := &Seeder{
				Flag:          newTestSeederFlag(seedPath),
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo},
				MetadataRepo:  metadataRepo,
				Logger:        logger.New("vine:test"),
				EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
				RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
				CertCore:      &core.PortalCertCore{PortalCertRepo: certRepo},
				SiteCore:      newTestSiteCore(entryRepo),
			}

			seeder.DIInit()

			assert.True(t, metadataRepo.IsSeeded())
			require.Len(t, ruleRepo.List(), 2)
			conflicts := seeder.RuleCore.Conflicts()
			require.Len(t, conflicts, 1)
			assert.ElementsMatch(t, []string{"demo.app", "demo.app-explicit"},
				[]string{conflicts[0].Rule, conflicts[0].Conflict})
			assert.Equal(t, "http://*:80/app", conflicts[0].MatchText())
		})
	}
}

func newTestSeederRepos(t *testing.T) (*repo.AppConfigRepo, *repo.PortalRuleRepo, *repo.PortalCertRepo, *repo.PortalSiteRepo, *repo.MetadataRepo, *watchserver.Server) {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hub.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&model.Metadata{}))
	(&model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](gdb)}).InitSchema()
	(&model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](gdb)}).InitSchema()
	(&model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](gdb)}).InitSchema()
	(&model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](gdb)}).InitSchema()
	(&model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](gdb)}).InitSchema()

	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	return &repo.AppConfigRepo{
		Dao:        &model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](gdb)},
		SchemaRepo: new(schema.SchemaRepo),
		Syncer:     testSyncer(watchServer),
		Access:     new(configaccess.Access),
	}, &repo.PortalRuleRepo{
		Dao:             &model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](gdb)},
		Syncer:          testSyncer(watchServer),
		Access:          new(configaccess.Access),
		PortalEntryRepo: &repo.PortalEntryRepo{Dao: &model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](gdb)}, Syncer: testSyncer(watchServer), Access: new(configaccess.Access)},
	}, &repo.PortalCertRepo{
		Dao:    &model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](gdb)},
		Syncer: testSyncer(watchServer),
		Access: new(configaccess.Access),
	}, &repo.PortalSiteRepo{
		Dao:        &model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](gdb)},
		SchemaRepo: new(schema.SchemaRepo),
		Syncer:     testSyncer(watchServer),
		Access:     new(configaccess.Access),
	}, &repo.MetadataRepo{
		Dao: &model.MetadataDao{Dao: rdb.NewDao[*model.Metadata](gdb)},
	}, watchServer
}

// saveTestPortalRule stores a rule under the entry that serves the access the
// caller declares, the way Core stores rules and the way the migration rebuilds
// entry storage for an upgraded database.
func saveTestPortalRule(t *testing.T, ruleRepo *repo.PortalRuleRepo, rule *core.PortalRule, access core.PortalEntry) {
	t.Helper()

	entry, ok := ruleRepo.PortalEntryRepo.GetBySchemeHostPort(access.Scheme, access.Host, access.Port)
	if !ok {
		entry = &core.PortalEntry{Scheme: access.Scheme, Host: access.Host, Port: access.Port, Enabled: true}
		ruleRepo.PortalEntryRepo.Save(entry)
	}
	rule.EntryId = entry.Id
	ruleRepo.Save(rule)
}

// newTestSiteCore builds a site core with the repositories Hub injects.
func newTestSiteCore(siteRepo core.PortalSiteRepo) *core.PortalSiteCore {
	return &core.PortalSiteCore{PortalSiteRepo: siteRepo, SchemaRepo: new(schema.SchemaRepo)}
}

// newTestRuleCore builds a rule core with the chosen rule repository and the
// entry repository rules resolve their access through.
func newTestRuleCore(ruleRepo *repo.PortalRuleRepo, siteRepo core.PortalSiteRepo) *core.PortalRuleCore {
	return &core.PortalRuleCore{
		PortalRuleRepo: ruleRepo,
		PortalSiteRepo: siteRepo,
		PortalEntryCore: &core.PortalEntryCore{
			PortalEntryRepo: ruleRepo.PortalEntryRepo,
			PortalRuleRepo:  ruleRepo,
			PortalSiteRepo:  siteRepo,
		},
	}
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
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configRepo}, EntryCore: newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
				RuleCore: newTestRuleCore(ruleRepo, entryRepo),
				CertCore: &core.PortalCertCore{PortalCertRepo: certRepo},
				SiteCore: newTestSiteCore(entryRepo), MetadataRepo: metadataRepo}
			require.Panics(t, seeder.DIInit)
			_, exists := configRepo.GetByName("pending")
			require.False(t, exists)
			_, exists = ruleRepo.GetByName("valid")
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
			target := &Seeder{Flag: newTestSeederFlag(path), Logger: logger.New("vine:test"), MetadataRepo: metadata,
				AppConfigCore: &core.AppConfigCore{AppConfigRepo: configs}, EntryCore: newTestEntryCore(rules.PortalEntryRepo, rules, sites),
				RuleCore: newTestRuleCore(rules, sites),
				CertCore: &core.PortalCertCore{PortalCertRepo: certs}, SiteCore: newTestSiteCore(sites)}
			require.Panics(t, target.DIInit)
			_, exists := configs.GetByName("pending")
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
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, siteRepo),
		RuleCore:      newTestRuleCore(ruleRepo, siteRepo),
		SiteCore:      newTestSiteCore(siteRepo),
		CertCore:      new(core.PortalCertCore{PortalCertRepo: certRepo}), MetadataRepo: metadataRepo, Logger: logger.New("seed-source-test"),
	})
	s.Flag.Normalize(true)
	s.DIInit()
	item, ok := configRepo.GetByName("second")
	require.True(t, ok)
	require.Equal(t, `"resolved"`, item.Value)
	require.Equal(t, core.FieldSource{Source: "app/default", Define: "domain/booker", Override: "app/default", Variables: []string{"value"}, Template: new(skel.JSON(`"${value}"`)), Bindings: []core.FieldSourceBinding{{Variable: "value", Reference: "${value}", Value: skel.JSON(`"\"resolved\""`)}}}, item.FieldSources["/value"])
	other, ok := configRepo.GetByName("first")
	require.True(t, ok)
	require.Empty(t, other.FieldSources)
	// Metadata is loaded from the database, not retained by the Seeder instance.
	reread, ok := configRepo.GetById(item.Id)
	require.True(t, ok)
	require.Equal(t, item.FieldSources, reread.FieldSources)
	s.AppConfigCore.Update(item.Id, core.AppConfigUpdate{Value: new(`"resolved"`)})
	updated, ok := configRepo.GetById(item.Id)
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
		EntryCore:     newTestEntryCore(rules.PortalEntryRepo, rules, sites),
		RuleCore:      newTestRuleCore(rules, sites),
		SiteCore:      newTestSiteCore(sites),
		CertCore:      new(core.PortalCertCore{PortalCertRepo: certs}),
		MetadataRepo:  metadata, Logger: logger.New("seed-key-sources-test"),
	})
	target.Flag.Normalize(true)
	target.DIInit()
	item, ok := configs.GetByName("user.AuthConfig")
	require.True(t, ok)
	require.JSONEq(t, `{"accessTokenTTL":"2h","refreshTokenTTL":"168h","nested":{"enabled":false}}`, item.Value)
	require.Equal(t, core.FieldSources{
		"/value/accessTokenTTL":  {Source: "profile/dev", Define: "domain/user", Override: "profile/dev", Variables: []string{"ttl"}, Template: new(skel.JSON(`"${ttl}"`)), Bindings: []core.FieldSourceBinding{{Variable: "ttl", Reference: "${ttl}", Value: skel.JSON(`"2h"`)}}},
		"/value/refreshTokenTTL": {Source: "domain/user", Define: "domain/user"},
		"/value/nested":          {Source: "app/default", Define: "domain/user", Override: "app/default"},
	}, item.FieldSources)
	reread, ok := configs.GetById(item.Id)
	require.True(t, ok)
	require.Equal(t, item.FieldSources, reread.FieldSources)
}

// A seed declares the access on the rule, and the entry owns it: the stored rule
// keeps the sources of the fields it owns, because an entry carries no field
// sources of its own.
func TestSeederKeepsOnlyRuleFieldSources(t *testing.T) {
	template := `
portalRules:
  - name: demo.app
    matchScheme: http
    matchHost: ""
    matchPort: 7088
    routeType: SITE
    routeSiteName: demo.Web
`
	source := fmt.Sprintf(`
version: 1
seedSha256: %x
fields:
  /portalRules/0/name: {source: app/default, define: domain/booker}
  /portalRules/0/matchScheme: {source: app/default, define: domain/booker}
  /portalRules/0/matchHost: {source: app/default, define: domain/booker}
  /portalRules/0/matchPort: {source: app/default, define: domain/booker}
`, sha256.Sum256([]byte(template)))
	configRepo, ruleRepo, certRepo, entryRepo, metadataRepo, _ := newTestSeederRepos(t)
	seeder := &Seeder{
		Flag:          new(flag.Flag{SeedHubData: template, SeedHubSource: source}),
		AppConfigCore: new(core.AppConfigCore{AppConfigRepo: configRepo}),
		EntryCore:     newTestEntryCore(ruleRepo.PortalEntryRepo, ruleRepo, entryRepo),
		RuleCore:      newTestRuleCore(ruleRepo, entryRepo),
		SiteCore:      newTestSiteCore(entryRepo),
		CertCore:      new(core.PortalCertCore{PortalCertRepo: certRepo}),
		MetadataRepo:  metadataRepo,
		Logger:        logger.New("seed-rule-sources-test"),
	}
	seeder.Flag.Normalize(true)

	seeder.DIInit()

	rule, ok := ruleRepo.GetByName("demo.app")
	require.True(t, ok)
	require.Equal(t, "app/default", rule.FieldSources["/name"].Source)
	for _, field := range []string{"/matchScheme", "/matchHost", "/matchPort"} {
		require.NotContains(t, rule.FieldSources, field)
	}
}

func writeSeedHubVarsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vars.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}
