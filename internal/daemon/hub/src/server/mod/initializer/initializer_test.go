package initializer

import (
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

func (r *testPortalEntryRepo) GetById(id int) (*core.PortalEntry, bool) {
	for i := range r.entries {
		if r.entries[i].Id == id {
			return r.entries[i], true
		}
	}
	return nil, false
}

func (r *testPortalEntryRepo) GetByName(name string) (*core.PortalEntry, bool) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			return r.entries[i], true
		}
	}
	return nil, false
}

func (*testPortalEntryRepo) GetByAccess(string, string, int) (*core.PortalEntry, bool) {
	return nil, false
}

func (*testPortalEntryRepo) Save(*core.PortalEntry) {}

func (r *testPortalEntryRepo) Remove(id int) bool {
	for i := range r.entries {
		if r.entries[i].Id == id {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			return true
		}
	}
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

func (r *testPortalRuleRepo) Remove(id int) bool {
	for i := range r.rules {
		if r.rules[i].Id == id {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return true
		}
	}
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

func (r *testPortalSiteRepo) Remove(id int) bool {
	for i := range r.entries {
		if r.entries[i].Id == id {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			return true
		}
	}
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
			},
		},
		SchemaRepo:   schemaRepo,
		RegistryCore: &core.RegistryCore{SchemaRepo: schemaRepo},
		InprocFlag:   &appcore.InternalInprocFlag{},
		Flag:         &hubflag.Flag{AdminListen: "127.0.0.1:7099"},
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

	siteKey := watched.FormatPortalSiteKey("demo-entry")
	value, err = db.Get(siteKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-entry"`)

	certKey := watched.FormatPortalCertKey("demo-cert")
	value, err = db.Get(certKey)
	assert.NoError(t, err)
	assert.Contains(t, value, `"name":"demo-cert"`)
}

func TestInitializerRemovesLegacyDashboard(t *testing.T) {
	// Hub used to publish its own Dashboard through Portal. A Watch store an
	// earlier release filled still carries those keys, and Hub removes them on
	// startup.
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	ruleRepo := &testPortalRuleRepo{rules: []*core.PortalRule{
		{Id: 1, Name: "demo.web", RouteType: "SITE", RouteSiteName: "demo.Web", Enabled: true},
	}}
	siteRepo := &testPortalSiteRepo{entries: []*core.PortalSite{
		{Id: 1, Name: "demo.Web", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", ActorVia: "client", WebName: "demo.Web", Enabled: true},
	}}
	entryRepo := &testPortalEntryRepo{entries: []*core.PortalEntry{
		{Id: 1, Name: "demo.web", Scheme: "http", Port: 80, Enabled: true},
	}}
	for _, name := range legacyDashboardRuleNames {
		watchServer.Set(watched.FormatPortalRuleKey(name), "{}")
	}
	for _, name := range legacyDashboardSiteNames {
		watchServer.Set(watched.FormatPortalSiteKey(name), "{}")
	}
	for _, serviceName := range legacyDashboardRpcServiceNames() {
		watchServer.Set(watched.FormatRpcServiceRegistrationKey(serviceName, legacyDashboardAppName, legacyDashboardAppInstanceId), "{}")
	}
	watchServer.Set(watched.FormatWebRegistrationKey(legacyDashboardWebSkelName, legacyDashboardAppName, legacyDashboardAppInstanceId), "{}")

	p := &Initializer{
		Syncer:          testSyncer(watchServer),
		AppConfigRepo:   &testAppConfigRepo{},
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  ruleRepo,
		PortalCertRepo:  &testPortalCertRepo{},
		PortalSiteRepo:  siteRepo,
		SchemaRepo:      &schema.SchemaRepo{},
		RegistryCore:    &core.RegistryCore{SchemaRepo: &schema.SchemaRepo{}},
		InprocFlag:      &appcore.InternalInprocFlag{},
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7099"},
	}

	p.DIInit()

	// Hub stores the entities its own seed declares and keeps the legacy
	// Dashboard keys out of Watch. The stored built-in rows go with the model's
	// cleanup, so they never reach this point.
	for _, key := range []string{
		watched.FormatPortalRuleKey("vine.hub.admin-api"),
		watched.FormatPortalRuleKey("vine.hub.dashboard-web"),
		watched.FormatPortalSiteKey("vine.hub.admin.AdminActor-client-rpc"),
		watched.FormatWebRegistrationKey(legacyDashboardWebSkelName, legacyDashboardAppName, legacyDashboardAppInstanceId),
	} {
		_, ok := watchServer.Get(key)
		assert.False(t, ok, key)
	}
	_, ok := watchServer.Get(watched.FormatPortalRuleKey("demo.web"))
	assert.True(t, ok, "Hub publishes the rules it stores")
}

func TestLegacyDashboardRpcServiceNamesDerivedFromRegisteredSchema(t *testing.T) {
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
	}, legacyDashboardRpcServiceNames())
}

func marshalTestConfigValue(name string, value string) string {
	return vcode.MustMarshalJsonS(watched.ConfigValue{
		Name:  name,
		Value: []byte(value),
	})
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
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7099"},
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
		Flag:            &hubflag.Flag{AdminListen: "127.0.0.1:7099"},
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
