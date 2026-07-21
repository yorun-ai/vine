package initializer

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appcore "go.yorun.ai/vine/internal/app"
	coreskel "go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	_ "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
	"go.yorun.ai/vine/util/vcode"
)

type _RedisTestStore struct {
	server *redisserver.Server
}

func formatTestRedisListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}

func (s _RedisTestStore) Get(key string) (string, error) {
	value, _ := s.server.Get(key)
	return value, nil
}

func (s _RedisTestStore) ExecuteCommand(command string, args ...string) ([]byte, error) {
	pattern := "*"
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "MATCH" {
			pattern = args[i+1]
			break
		}
	}
	return []byte(strings.Join(s.server.Scan(pattern), "\n")), nil
}

type testAppConfigRepo struct {
	items []*core.AppConfig
}

type testPortalRuleRepo struct {
	rules []core.PortalRule
}

type testPortalCertRepo struct {
	certs []*core.PortalCert
}

type testPortalSiteRepo struct {
	entries []core.PortalSite
}

func (r *testAppConfigRepo) ListItems() []*core.AppConfig {
	return r.items
}

func (*testAppConfigRepo) GetItemById(int) (*core.AppConfig, bool) {
	return nil, false
}

func (*testAppConfigRepo) GetItemByName(string) (*core.AppConfig, bool) {
	return nil, false
}

func (*testAppConfigRepo) SaveItem(*core.AppConfig) {
}

func (*testAppConfigRepo) RemoveItem(int) bool {
	return false
}

func (r *testPortalRuleRepo) ListRules() []core.PortalRule {
	return r.rules
}

func (*testPortalRuleRepo) GetRuleById(int) (*core.PortalRule, bool) {
	return nil, false
}

func (r *testPortalRuleRepo) GetRuleByName(name string) (*core.PortalRule, bool) {
	for i := range r.rules {
		if r.rules[i].Name == name {
			return &r.rules[i], true
		}
	}
	return nil, false
}

func (r *testPortalRuleRepo) SaveRule(rule *core.PortalRule) {
	for i := range r.rules {
		if r.rules[i].Name == rule.Name {
			r.rules[i] = *rule
			return
		}
	}
	r.rules = append(r.rules, *rule)
}

func (*testPortalRuleRepo) RemoveRule(int) bool {
	return false
}

func (r *testPortalCertRepo) ListCerts() []*core.PortalCert {
	return r.certs
}

func (*testPortalCertRepo) GetCertById(int) (*core.PortalCert, bool) {
	return nil, false
}

func (*testPortalCertRepo) GetCertByName(string) (*core.PortalCert, bool) {
	return nil, false
}

func (*testPortalCertRepo) SaveCert(*core.PortalCert) {
}

func (*testPortalCertRepo) RemoveCert(int) bool {
	return false
}

func (r *testPortalSiteRepo) ListEntries() []core.PortalSite {
	return r.entries
}

func (*testPortalSiteRepo) GetEntryById(int) (*core.PortalSite, bool) {
	return nil, false
}

func (r *testPortalSiteRepo) GetEntryByName(name string) (*core.PortalSite, bool) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			return &r.entries[i], true
		}
	}
	return nil, false
}

func (r *testPortalSiteRepo) SaveEntry(entry *core.PortalSite) {
	for i := range r.entries {
		if r.entries[i].Name == entry.Name {
			r.entries[i] = *entry
			return
		}
	}
	r.entries = append(r.entries, *entry)
}

func (*testPortalSiteRepo) RemoveEntry(int) bool {
	return false
}

func testSyncer(redisServer *redisserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{RedisServer: redisServer}
	target.DIInit()
	return target
}

func testPortalRuleWithId(id int, rule core.PortalRule) core.PortalRule {
	rule.Id = id
	return rule
}

func testDashboardApiRule() core.PortalRule {
	return core.PortalRule{
		Name:       core.DashboardAdminApiRuleName,
		Scheme:     "http",
		Port:       7099,
		PathPrefix: "/api",
		TargetType: "SITE",
		SiteName:   seeder.DashboardRpcCoreEntry.Name,
		BuiltIn:    true,
	}
}

func testDashboardWebRule() core.PortalRule {
	return core.PortalRule{
		Name:       core.DashboardWebRuleName,
		Scheme:     "http",
		Port:       7099,
		PathPrefix: "/",
		TargetType: "SITE",
		SiteName:   seeder.DashboardWebCoreEntry.Name,
		BuiltIn:    true,
	}
}

func testPortalSiteWithId(id int, site core.PortalSite) core.PortalSite {
	site.Id = id
	return site
}

func TestInitializerDIInitWritesRepoItems(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	db := _RedisTestStore{redisServer}

	p := &Initializer{
		Syncer: testSyncer(redisServer),
		AppConfigRepo: &testAppConfigRepo{
			items: []*core.AppConfig{
				{Id: 1, Name: "demo.DatabaseConfig", Value: `{"dsn":"postgres://demo"}`},
				{Id: 2, Name: "demo.FeatureConfig", Value: `{"enabled":true}`},
			},
		},
		RuleRepo: &testPortalRuleRepo{
			rules: []core.PortalRule{
				{Id: 1, Name: "demo-entry", Scheme: "https", Host: "demo.local", PathPrefix: "/admin", TargetType: "SITE", SiteName: "admin@demo.app"},
				testPortalRuleWithId(2, testDashboardApiRule()),
				testPortalRuleWithId(3, testDashboardWebRule()),
			},
		},
		CertRepo: &testPortalCertRepo{
			certs: []*core.PortalCert{
				{Id: 1, Name: "demo-cert", Issuer: "letsencrypt", Domains: []string{"demo.local"}, PublicKeyBase64: "pub", PrivateKeyBase64: "pri"},
			},
		},
		EntryRepo: &testPortalSiteRepo{
			entries: []core.PortalSite{
				{Id: 1, Name: "demo-entry", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", ActorVia: "client", WebName: "demo.Web"},
				testPortalSiteWithId(2, seeder.DashboardRpcCoreEntry),
				testPortalSiteWithId(3, seeder.DashboardWebCoreEntry),
			},
		},
		SchemaRepo: &schema.MemorySchemaRepo{},
		InprocFlag: &appcore.InternalInprocFlag{},
		Flag:       &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	p.DIInit()

	value, err := db.Get(redised.FormatConfigKey("demo.DatabaseConfig"))
	assert.NoError(t, err)
	assert.Equal(t, marshalTestConfigValue("demo.DatabaseConfig", `{"dsn":"postgres://demo"}`), value)

	value, err = db.Get(redised.FormatConfigKey("demo.FeatureConfig"))
	assert.NoError(t, err)
	assert.Equal(t, marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`), value)

	ruleKey := redised.FormatPortalRuleKey("demo-entry")
	value, err = db.Get(ruleKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-entry"`)

	defaultRuleValue, err := db.Get(redised.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.NoError(t, err)
	assert.Contains(t, defaultRuleValue, `"name":"vine.hub.admin-api"`)

	defaultWebRuleValue, err := db.Get(redised.FormatPortalRuleKey(core.DashboardWebRuleName))
	assert.NoError(t, err)
	assert.Contains(t, defaultWebRuleValue, `"name":"vine.hub.dashboard-web"`)

	defaultSiteValue, err := db.Get(redised.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, defaultSiteValue, `"name":"vine.hub.AdminActor-client-rpc"`)

	defaultWebSiteValue, err := db.Get(redised.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, defaultWebSiteValue, `"name":"vine.hub.DashboardWeb-web"`)

	adminActorValue, err := db.Get(redised.FormatSchemaActorKey("vine.hub.AdminActor"))
	assert.NoError(t, err)
	assert.Contains(t, adminActorValue, `"skelName":"vine.hub.AdminActor"`)

	skeletonServiceValue, err := db.Get(redised.FormatSchemaServiceKey("vine.hub.SkeletonService"))
	assert.NoError(t, err)
	assert.Contains(t, skeletonServiceValue, `"authMode":"noauth"`)

	siteKey := redised.FormatPortalSiteKey("demo-entry")
	value, err = db.Get(siteKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-entry"`)

	defaultWebRegistrationKey := redised.FormatWebRegistrationKey(seeder.DashboardWebCoreEntry.WebName, dashboardAppName, dashboardAppInstanceId)
	defaultWebRegistrationValue, err := db.Get(defaultWebRegistrationKey)
	assert.NoError(t, err)
	expectedWebReg := dashboardWebRegistration
	expectedWebReg.Endpoint = p.dashboardWebEndpoint()
	assert.Equal(t, vcode.MustMarshalJsonS(expectedWebReg), defaultWebRegistrationValue)

	entryRuleScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "portal:rule:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(entryRuleScan), redised.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.Contains(t, string(entryRuleScan), redised.FormatPortalRuleKey(core.DashboardWebRuleName))
	siteScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "portal:site:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(siteScan), redised.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.Contains(t, string(siteScan), redised.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))

	webRegistrationScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "web:"+seeder.DashboardWebCoreEntry.WebName+":endpoint:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(webRegistrationScan), defaultWebRegistrationKey)

	for _, serviceName := range seeder.DashboardRpcServices {
		registrationKey := redised.FormatRpcServiceRegistrationKey(serviceName, dashboardAppName, dashboardAppInstanceId)
		registrationValue, err := db.Get(registrationKey)
		assert.NoError(t, err)
		expectedReg := dashboardRpcRegistrations[serviceName]
		expectedReg.Endpoint = p.dashboardRpcEndpoint()
		assert.Equal(t, vcode.MustMarshalJsonS(expectedReg), registrationValue)

		registrationScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", formatTestRedisListPattern(redised.FormatRpcServiceRegistrationPrefix(serviceName)), "COUNT", strconv.Itoa(1000))
		assert.NoError(t, err)
		assert.Contains(t, string(registrationScan), registrationKey)
	}

	certKey := redised.FormatPortalCertKey("demo-cert")
	value, err = db.Get(certKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-cert"`)
}

func TestInitializerDIInitWritesDashboardEntriesAndRulesFromRepo(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	db := _RedisTestStore{redisServer}

	existingApiRule := core.PortalRule{
		Id:         1,
		Name:       core.DashboardAdminApiRuleName,
		Scheme:     "http",
		Port:       8088,
		PathPrefix: "/custom-api",
		TargetType: "SITE",
		SiteName:   "custom-admin-entry",
		BuiltIn:    true,
	}
	existingWebRule := core.PortalRule{
		Id:         2,
		Name:       core.DashboardWebRuleName,
		Scheme:     "http",
		Port:       8088,
		PathPrefix: "/custom-web",
		TargetType: "SITE",
		SiteName:   "custom-web-entry",
		BuiltIn:    true,
	}
	existingRpcSite := core.PortalSite{
		Id:            1,
		Name:          seeder.DashboardRpcCoreEntry.Name,
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "custom.Actor",
		ActorVia:      "client",
		BuiltIn:       true,
	}
	existingWebSite := core.PortalSite{
		Id:            2,
		Name:          seeder.DashboardWebCoreEntry.Name,
		Type:          core.PortalSiteTypeWEBGW,
		ActorSkelName: "custom.Actor",
		ActorVia:      "client",
		WebName:       "custom.Web",
		BuiltIn:       true,
	}
	ruleRepo := &testPortalRuleRepo{rules: []core.PortalRule{existingApiRule, existingWebRule}}
	entryRepo := &testPortalSiteRepo{entries: []core.PortalSite{existingRpcSite, existingWebSite}}
	p := &Initializer{
		Syncer:        testSyncer(redisServer),
		AppConfigRepo: &testAppConfigRepo{},
		RuleRepo:      ruleRepo,
		CertRepo:      &testPortalCertRepo{},
		EntryRepo:     entryRepo,
		SchemaRepo:    &schema.MemorySchemaRepo{},
		InprocFlag:    &appcore.InternalInprocFlag{},
		Flag:          &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	p.DIInit()

	assert.Len(t, ruleRepo.rules, 2)
	assert.Len(t, entryRepo.entries, 2)

	value, err := db.Get(redised.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.NoError(t, err)
	assert.Contains(t, value, `"port":8088`)
	assert.Contains(t, value, "/custom-api")

	value, err = db.Get(redised.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, value, "custom.Actor")

	value, err = db.Get(redised.FormatPortalRuleKey(core.DashboardWebRuleName))
	assert.NoError(t, err)
	assert.Contains(t, value, `"port":8088`)
	assert.Contains(t, value, "/custom-web")

	value, err = db.Get(redised.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, value, "custom.Web")
}

func TestDashboardRpcServicesDerivedFromRegisteredSchema(t *testing.T) {
	assert.Equal(t, []string{
		"vine.hub.AppConfigService",
		"vine.hub.AppStatusService",
		"vine.hub.EventDebugService",
		"vine.hub.MaintenanceService",
		"vine.hub.PortalCertService",
		"vine.hub.PortalEntryService",
		"vine.hub.PortalRuleService",
		"vine.hub.PortalSiteService",
		"vine.hub.ServiceDebugService",
		"vine.hub.SkeletonService",
		"vine.hub.TaskDebugService",
	}, seeder.DashboardRpcServices)
}

func marshalTestConfigValue(name string, value string) string {
	return vcode.MustMarshalJsonS(redised.ConfigValue{
		Name:  name,
		Value: []byte(value),
	})
}

func TestDashboardRpcEndpointUsesInprocWhenEnabled(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	assert.Equal(t, "rpc+inproc://vine/hub/rpc/invoke", initializer.dashboardRpcEndpoint())
}

func TestDashboardWebEndpointUsesInprocWhenEnabled(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	assert.Equal(t, "web+inproc://vine/hub/web/access/vine.hub.DashboardWeb", initializer.dashboardWebEndpoint())
}

func TestDashboardRpcEndpointUsesHTTPListenOutsideInproc(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{},
		Flag:       &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	assert.Equal(t, "http://127.0.0.1:7071/rpc/invoke", initializer.dashboardRpcEndpoint())
}

func TestDashboardWebEndpointUsesHTTPListenOutsideInproc(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{},
		Flag:       &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	assert.Equal(t, "http://127.0.0.1:7071/web/access/vine.hub.DashboardWeb", initializer.dashboardWebEndpoint())
}

func TestInitializerDIInitLoadsRegisteredSchemasIntoMemoryRepoInInprocMode(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()

	domainSchema := &coreskel.DomainSchema{
		Domain:    "test.initializer.schema",
		Hash:      "test-initializer-schema-hash",
		Generated: &coreskel.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	coreskel.RegisterDomainSchema(domainSchema)

	schemaRepo := new(schema.MemorySchemaRepo)
	p := &Initializer{
		Syncer:        testSyncer(redisServer),
		AppConfigRepo: &testAppConfigRepo{},
		RuleRepo:      &testPortalRuleRepo{},
		CertRepo:      &testPortalCertRepo{},
		EntryRepo:     &testPortalSiteRepo{},
		SchemaRepo:    schemaRepo,
		InprocFlag:    &appcore.InternalInprocFlag{Enabled: true},
		Flag:          &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	p.DIInit()

	got, ok := findDomainSchemaByHash(schemaRepo.ListDomainSchemaViews(), domainSchema.Hash)
	assert.True(t, ok)
	assert.Same(t, domainSchema, got)
}

func TestInitializerDIInitLoadsHubSchemaIntoMemoryRepoInNormalMode(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()

	var hubSchema *coreskel.DomainSchema
	for _, schema := range coreskel.RegisteredDomainSchemas() {
		if schema.Domain == "vine.hub" {
			hubSchema = schema
			break
		}
	}
	if hubSchema == nil {
		t.Fatal("expected hub domain schema")
	}

	schemaRepo := new(schema.MemorySchemaRepo)
	p := &Initializer{
		Syncer:        testSyncer(redisServer),
		AppConfigRepo: &testAppConfigRepo{},
		RuleRepo:      &testPortalRuleRepo{},
		CertRepo:      &testPortalCertRepo{},
		EntryRepo:     &testPortalSiteRepo{},
		SchemaRepo:    schemaRepo,
		InprocFlag:    &appcore.InternalInprocFlag{},
		Flag:          &hubflag.Flag{APIListen: "127.0.0.1:7071"},
	}

	p.DIInit()

	got, ok := findDomainSchemaByHash(schemaRepo.ListDomainSchemaViews(), hubSchema.Hash)
	assert.True(t, ok)
	assert.Same(t, hubSchema, got)
}

func findDomainSchemaByHash(views []core.DomainSchemaView, hash string) (*coreskel.DomainSchema, bool) {
	for _, view := range views {
		if view.DomainVersion.Schema.Hash == hash {
			return view.DomainVersion.Schema, true
		}
	}
	return nil, false
}
