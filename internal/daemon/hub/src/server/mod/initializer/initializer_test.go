package initializer

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appcore "go.yorun.ai/vine/internal/app"
	coreskel "go.yorun.ai/vine/internal/core/skel"
	_ "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	_ "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
	"go.yorun.ai/vine/util/vcode"
)

type _WatchTestStore struct {
	server *watchserver.Server
}

func formatTestWatchListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}

func (s _WatchTestStore) Get(key string) (string, error) {
	value, _ := s.server.Get(key)
	return value, nil
}

func (s _WatchTestStore) ExecuteCommand(command string, args ...string) ([]byte, error) {
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
	rules []*core.PortalRule
}

type testPortalCertRepo struct {
	certs []*core.PortalCert
}

type testPortalSiteRepo struct {
	entries []*core.PortalSite
}

// testPortalEntryRepo is an empty entry repository: the initializer only lists
// entries to record which of them Hub publishes rules for.
type testPortalEntryRepo struct {
	entries []*core.PortalEntry
}

func (r *testPortalEntryRepo) List() []*core.PortalEntry {
	return r.entries
}

func (*testPortalEntryRepo) GetById(int) (*core.PortalEntry, bool) {
	return nil, false
}

func (*testPortalEntryRepo) GetByName(string) (*core.PortalEntry, bool) {
	return nil, false
}

func (*testPortalEntryRepo) GetByAccess(string, string, int) (*core.PortalEntry, bool) {
	return nil, false
}

func (*testPortalEntryRepo) GetBuiltIn() (*core.PortalEntry, bool) {
	return nil, false
}

func (*testPortalEntryRepo) Save(*core.PortalEntry) {}

func (*testPortalEntryRepo) Remove(int) bool {
	return false
}

func (r *testAppConfigRepo) List() []*core.AppConfig {
	return r.items
}

func (r *testAppConfigRepo) ListSlots() []*core.AppConfig {
	return r.List()
}

func (r *testAppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	return r.GetByName(name)
}

func (*testAppConfigRepo) GetById(int) (*core.AppConfig, bool) {
	return nil, false
}

func (*testAppConfigRepo) GetByName(string) (*core.AppConfig, bool) {
	return nil, false
}

func (*testAppConfigRepo) Save(*core.AppConfig) {
}

func (*testAppConfigRepo) Remove(int) bool {
	return false
}

func (r *testPortalRuleRepo) List() []*core.PortalRule {
	return r.rules
}

func (*testPortalRuleRepo) GetById(int) (*core.PortalRule, bool) {
	return nil, false
}

func (r *testPortalRuleRepo) GetByName(name string) (*core.PortalRule, bool) {
	for i := range r.rules {
		if r.rules[i].Name == name {
			return r.rules[i], true
		}
	}
	return nil, false
}

func (r *testPortalRuleRepo) Save(rule *core.PortalRule) {
	for i := range r.rules {
		if r.rules[i].Name == rule.Name {
			r.rules[i] = rule
			return
		}
	}
	r.rules = append(r.rules, rule)
}

func (*testPortalRuleRepo) Remove(int) bool {
	return false
}

func (r *testPortalCertRepo) List() []*core.PortalCert {
	return r.certs
}

func (*testPortalCertRepo) GetById(int) (*core.PortalCert, bool) {
	return nil, false
}

func (*testPortalCertRepo) GetByName(string) (*core.PortalCert, bool) {
	return nil, false
}

func (*testPortalCertRepo) Save(*core.PortalCert) {
}

func (*testPortalCertRepo) Remove(int) bool {
	return false
}

func (r *testPortalSiteRepo) List() []*core.PortalSite {
	return r.entries
}

func (*testPortalSiteRepo) GetById(int) (*core.PortalSite, bool) {
	return nil, false
}

func (r *testPortalSiteRepo) GetByName(name string) (*core.PortalSite, bool) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			return r.entries[i], true
		}
	}
	return nil, false
}

func (r *testPortalSiteRepo) Save(entry *core.PortalSite) {
	for i := range r.entries {
		if r.entries[i].Name == entry.Name {
			r.entries[i] = entry
			return
		}
	}
	r.entries = append(r.entries, entry)
}

func (*testPortalSiteRepo) Remove(int) bool {
	return false
}

func testSyncer(watchServer *watchserver.Server) *syncer.Syncer {
	target := &syncer.Syncer{WatchServer: watchServer}
	target.DIInit()
	return target
}

func testPortalRulePtrWithId(id int, rule core.PortalRule) *core.PortalRule {
	value := testPortalRuleWithId(id, rule)
	return &value
}

func testPortalSitePtrWithId(id int, site core.PortalSite) *core.PortalSite {
	value := testPortalSiteWithId(id, site)
	return &value
}

func testPortalRuleWithId(id int, rule core.PortalRule) core.PortalRule {
	rule.Id = id
	return rule
}

func testDashboardApiRule() core.PortalRule {
	return core.PortalRule{
		Name:            core.DashboardAdminApiRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/api",
		RouteType:       "SITE",
		RouteSiteName:   seeder.DashboardRpcCoreEntry.Name,
		BuiltIn:         true,
		Enabled:         true,
	}
}

func testDashboardWebRule() core.PortalRule {
	return core.PortalRule{
		Name:            core.DashboardWebRuleName,
		MatchScheme:     "http",
		MatchPort:       7099,
		MatchPathPrefix: "/",
		RouteType:       "SITE",
		RouteSiteName:   seeder.DashboardWebCoreEntry.Name,
		BuiltIn:         true,
		Enabled:         true,
	}
}

func testPortalSiteWithId(id int, site core.PortalSite) core.PortalSite {
	site.Id = id
	return site
}

func TestInitializerDIInitWritesRepoItems(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	db := _WatchTestStore{watchServer}
	schemaRepo := new(schema.SchemaRepo)

	p := &Initializer{
		Syncer: testSyncer(watchServer),
		AppConfigRepo: &testAppConfigRepo{
			items: []*core.AppConfig{
				{Id: 1, Name: "demo.DatabaseConfig", Value: `{"dsn":"postgres://demo"}`},
				{Id: 2, Name: "demo.FeatureConfig", Value: `{"enabled":true}`},
			},
		},
		PortalEntryRepo: &testPortalEntryRepo{},
		PortalRuleRepo: &testPortalRuleRepo{
			rules: []*core.PortalRule{
				{Id: 1, Name: "demo-entry", MatchScheme: "https", MatchHost: "demo.local", MatchPathPrefix: "/admin", RouteType: "SITE", RouteSiteName: "admin@demo.app", Enabled: true},
				testPortalRulePtrWithId(2, testDashboardApiRule()),
				testPortalRulePtrWithId(3, testDashboardWebRule()),
			},
		},
		PortalCertRepo: &testPortalCertRepo{
			certs: []*core.PortalCert{
				{Id: 1, Name: "demo-cert", Issuer: "letsencrypt", Domains: []string{"demo.local"}, PublicKeyBase64: "pub", PrivateKeyBase64: "pri", Enabled: true},
			},
		},
		PortalSiteRepo: &testPortalSiteRepo{
			entries: []*core.PortalSite{
				{Id: 1, Name: "demo-entry", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", ActorVia: "client", WebName: "demo.Web", Enabled: true},
				testPortalSitePtrWithId(2, seeder.DashboardRpcCoreEntry),
				testPortalSitePtrWithId(3, seeder.DashboardWebCoreEntry),
			},
		},
		SchemaRepo:   schemaRepo,
		RegistryCore: &core.RegistryCore{SchemaRepo: schemaRepo},
		InprocFlag:   &appcore.InternalInprocFlag{},
		Flag:         &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	p.DIInit()

	value, err := db.Get(watched.FormatConfigKey("demo.DatabaseConfig"))
	assert.NoError(t, err)
	assert.Equal(t, marshalTestConfigValue("demo.DatabaseConfig", `{"dsn":"postgres://demo"}`), value)

	value, err = db.Get(watched.FormatConfigKey("demo.FeatureConfig"))
	assert.NoError(t, err)
	assert.Equal(t, marshalTestConfigValue("demo.FeatureConfig", `{"enabled":true}`), value)

	ruleKey := watched.FormatPortalRuleKey("demo-entry")
	value, err = db.Get(ruleKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-entry"`)

	defaultRuleValue, err := db.Get(watched.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.NoError(t, err)
	assert.Contains(t, defaultRuleValue, `"name":"vine.hub.admin-api"`)

	defaultWebRuleValue, err := db.Get(watched.FormatPortalRuleKey(core.DashboardWebRuleName))
	assert.NoError(t, err)
	assert.Contains(t, defaultWebRuleValue, `"name":"vine.hub.dashboard-web"`)

	defaultSiteValue, err := db.Get(watched.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, defaultSiteValue, `"name":"vine.hub.admin.AdminActor-client-rpc"`)

	defaultWebSiteValue, err := db.Get(watched.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, defaultWebSiteValue, `"name":"vine.hub.admin.DashboardWeb-web"`)

	adminActorValue, err := db.Get(watched.FormatSchemaActorKey("vine.hub.admin.AdminActor"))
	assert.NoError(t, err)
	assert.Contains(t, adminActorValue, `"skelName":"vine.hub.admin.AdminActor"`)

	skeletonServiceValue, err := db.Get(watched.FormatSchemaServiceKey("vine.hub.admin.SkeletonApiService"))
	assert.NoError(t, err)
	assert.Contains(t, skeletonServiceValue, `"authMode":"noauth"`)

	siteKey := watched.FormatPortalSiteKey("demo-entry")
	value, err = db.Get(siteKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-entry"`)

	defaultWebRegistrationKey := watched.FormatWebRegistrationKey(seeder.DashboardWebCoreEntry.WebName, dashboardAppName, dashboardAppInstanceId)
	defaultWebRegistrationValue, err := db.Get(defaultWebRegistrationKey)
	assert.NoError(t, err)
	expectedWebReg := dashboardWebRegistration
	expectedWebReg.Endpoint = p.dashboardWebEndpoint()
	assert.Equal(t, vcode.MustMarshalJsonS(expectedWebReg), defaultWebRegistrationValue)

	entryRuleScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "portal:rule:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(entryRuleScan), watched.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.Contains(t, string(entryRuleScan), watched.FormatPortalRuleKey(core.DashboardWebRuleName))
	siteScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "portal:site:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(siteScan), watched.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.Contains(t, string(siteScan), watched.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))

	webRegistrationScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", "web:"+seeder.DashboardWebCoreEntry.WebName+":endpoint:*", "COUNT", strconv.Itoa(1000))
	assert.NoError(t, err)
	assert.Contains(t, string(webRegistrationScan), defaultWebRegistrationKey)

	for _, serviceName := range seeder.DashboardRpcServices {
		registrationKey := watched.FormatRpcServiceRegistrationKey(serviceName, dashboardAppName, dashboardAppInstanceId)
		registrationValue, err := db.Get(registrationKey)
		assert.NoError(t, err)
		expectedReg := dashboardRpcRegistrations[serviceName]
		expectedReg.Endpoint = p.dashboardRpcEndpoint()
		assert.Equal(t, vcode.MustMarshalJsonS(expectedReg), registrationValue)

		registrationScan, err := db.ExecuteCommand("SCAN", "0", "MATCH", formatTestWatchListPattern(watched.FormatRpcServiceRegistrationPrefix(serviceName)), "COUNT", strconv.Itoa(1000))
		assert.NoError(t, err)
		assert.Contains(t, string(registrationScan), registrationKey)
	}

	certKey := watched.FormatPortalCertKey("demo-cert")
	value, err = db.Get(certKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-cert"`)
}

func TestInitializerDIInitWritesDashboardEntriesAndRulesFromRepo(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	db := _WatchTestStore{watchServer}

	existingApiRule := core.PortalRule{
		Id:              1,
		Name:            core.DashboardAdminApiRuleName,
		MatchScheme:     "http",
		MatchPort:       8088,
		MatchPathPrefix: "/custom-api",
		RouteType:       "SITE",
		RouteSiteName:   "custom-admin-entry",
		BuiltIn:         true,
		Enabled:         true,
	}
	existingWebRule := core.PortalRule{
		Id:              2,
		Name:            core.DashboardWebRuleName,
		MatchScheme:     "http",
		MatchPort:       8088,
		MatchPathPrefix: "/custom-web",
		RouteType:       "SITE",
		RouteSiteName:   "custom-web-entry",
		BuiltIn:         true,
		Enabled:         true,
	}
	existingRpcSite := core.PortalSite{
		Id:            1,
		Name:          seeder.DashboardRpcCoreEntry.Name,
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "custom.Actor",
		ActorVia:      "client",
		BuiltIn:       true,
		Enabled:       true,
	}
	existingWebSite := core.PortalSite{
		Id:            2,
		Name:          seeder.DashboardWebCoreEntry.Name,
		Type:          core.PortalSiteTypeWEBGW,
		ActorSkelName: "custom.Actor",
		ActorVia:      "client",
		WebName:       "custom.Web",
		BuiltIn:       true,
		Enabled:       true,
	}
	ruleRepo := &testPortalRuleRepo{rules: []*core.PortalRule{&existingApiRule, &existingWebRule}}
	entryRepo := &testPortalSiteRepo{entries: []*core.PortalSite{&existingRpcSite, &existingWebSite}}
	p := &Initializer{
		Syncer:          testSyncer(watchServer),
		AppConfigRepo:   &testAppConfigRepo{},
		PortalEntryRepo: &testPortalEntryRepo{},
		PortalRuleRepo:  ruleRepo,
		PortalCertRepo:  &testPortalCertRepo{},
		PortalSiteRepo:  entryRepo,
		SchemaRepo:      &schema.SchemaRepo{},
		RegistryCore:    &core.RegistryCore{SchemaRepo: &schema.SchemaRepo{}},
		InprocFlag:      &appcore.InternalInprocFlag{},
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	p.DIInit()

	assert.Len(t, ruleRepo.rules, 2)
	assert.Len(t, entryRepo.entries, 2)

	value, err := db.Get(watched.FormatPortalRuleKey(core.DashboardAdminApiRuleName))
	assert.NoError(t, err)
	assert.Contains(t, value, `"matchPort":8088`)
	assert.Contains(t, value, "/custom-api")

	value, err = db.Get(watched.FormatPortalSiteKey(seeder.DashboardRpcCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, value, "custom.Actor")

	value, err = db.Get(watched.FormatPortalRuleKey(core.DashboardWebRuleName))
	assert.NoError(t, err)
	assert.Contains(t, value, `"matchPort":8088`)
	assert.Contains(t, value, "/custom-web")

	value, err = db.Get(watched.FormatPortalSiteKey(seeder.DashboardWebCoreEntry.Name))
	assert.NoError(t, err)
	assert.Contains(t, value, "custom.Web")
}

func TestDashboardRpcServicesDerivedFromRegisteredSchema(t *testing.T) {
	assert.Equal(t, []string{
		"vine.hub.admin.AppConfigApiService",
		"vine.hub.admin.AppStatusApiService",
		"vine.hub.admin.EventDebugApiService",
		"vine.hub.admin.MaintenanceApiService",
		"vine.hub.admin.PortalCertApiService",
		"vine.hub.admin.PortalEntryApiService",
		"vine.hub.admin.PortalRuleApiService",
		"vine.hub.admin.PortalSiteApiService",
		"vine.hub.admin.PortalStatusApiService",
		"vine.hub.admin.ServiceDebugApiService",
		"vine.hub.admin.SkeletonApiService",
		"vine.hub.admin.TaskDebugApiService",
	}, seeder.DashboardRpcServices)
}

func marshalTestConfigValue(name string, value string) string {
	return vcode.MustMarshalJsonS(watched.ConfigValue{
		Name:  name,
		Value: []byte(value),
	})
}

func TestDashboardRpcEndpointUsesInprocWhenEnabled(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	assert.Equal(t, "rpc+inproc://vine/hub/admin/rpc/invoke", initializer.dashboardRpcEndpoint())
}

func TestDashboardWebEndpointUsesInprocWhenEnabled(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{Enabled: true},
		Flag:       &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	assert.Equal(t, "web+inproc://vine/hub/admin/web/access/vine.hub.admin.DashboardWeb", initializer.dashboardWebEndpoint())
}

func TestDashboardRpcEndpointUsesHTTPListenOutsideInproc(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{},
		Flag:       &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	assert.Equal(t, "http://127.0.0.1:7075/rpc/invoke", initializer.dashboardRpcEndpoint())
}

func TestDashboardWebEndpointUsesHTTPListenOutsideInproc(t *testing.T) {
	initializer := &Initializer{
		InprocFlag: &appcore.InternalInprocFlag{},
		Flag:       &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	assert.Equal(t, "http://127.0.0.1:7075/web/access/vine.hub.admin.DashboardWeb", initializer.dashboardWebEndpoint())
}

func TestInitializerDIInitLoadsRegisteredSchemasIntoMemoryRepoInInprocMode(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()

	domainSchema := &coreskel.DomainSchema{
		Domain:    "test.initializer.schema",
		Hash:      "test-initializer-schema-hash",
		Generated: &coreskel.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	coreskel.RegisterDomainSchema(domainSchema)

	schemaRepo := new(schema.SchemaRepo)
	p := &Initializer{
		Syncer:          testSyncer(watchServer),
		AppConfigRepo:   &testAppConfigRepo{},
		PortalEntryRepo: &testPortalEntryRepo{},
		PortalRuleRepo:  &testPortalRuleRepo{},
		PortalCertRepo:  &testPortalCertRepo{},
		PortalSiteRepo:  &testPortalSiteRepo{},
		SchemaRepo:      schemaRepo,
		RegistryCore:    &core.RegistryCore{SchemaRepo: schemaRepo},
		InprocFlag:      &appcore.InternalInprocFlag{Enabled: true},
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	p.DIInit()

	got, ok := findDomainSchemaByHash(schemaRepo.ListDomainSchemaViews(), domainSchema.Hash)
	assert.True(t, ok)
	assert.Same(t, domainSchema, got)
}

func TestInitializerDIInitLoadsHubSchemasIntoMemoryRepoInNormalMode(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()

	hubSchemas := make(map[string]*coreskel.DomainSchema)
	for _, schema := range coreskel.RegisteredDomainSchemas() {
		switch schema.Domain {
		case "vine.hub.control", "vine.hub.admin":
			hubSchemas[schema.Domain] = schema
		}
	}
	if len(hubSchemas) != 2 {
		t.Fatalf("expected both Hub domain schemas, got %v", hubSchemas)
	}

	schemaRepo := new(schema.SchemaRepo)
	p := &Initializer{
		Syncer:          testSyncer(watchServer),
		AppConfigRepo:   &testAppConfigRepo{},
		PortalEntryRepo: &testPortalEntryRepo{},
		PortalRuleRepo:  &testPortalRuleRepo{},
		PortalCertRepo:  &testPortalCertRepo{},
		PortalSiteRepo:  &testPortalSiteRepo{},
		SchemaRepo:      schemaRepo,
		RegistryCore:    &core.RegistryCore{SchemaRepo: schemaRepo},
		InprocFlag:      &appcore.InternalInprocFlag{},
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7075"},
	}

	p.DIInit()

	views := schemaRepo.ListDomainSchemaViews()
	for domain, hubSchema := range hubSchemas {
		got, ok := findDomainSchemaByHash(views, hubSchema.Hash)
		assert.True(t, ok, domain)
		assert.Same(t, hubSchema, got, domain)
	}
}

func findDomainSchemaByHash(views []core.DomainSchemaView, hash string) (*coreskel.DomainSchema, bool) {
	for _, view := range views {
		if view.DomainVersion.Schema.Hash == hash {
			return view.DomainVersion.Schema, true
		}
	}
	return nil, false
}
