package initializer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appcore "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
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

// withTestAudit fills the fields the conflict audit reads, so a test states only
// the repositories it exercises.
func withTestAudit(p *Initializer) *Initializer {
	if p.RuleCore == nil {
		p.RuleCore = &core.PortalRuleCore{
			PortalRuleRepo: p.PortalRuleRepo,
			PortalSiteRepo: p.PortalSiteRepo,
			PortalEntryCore: &core.PortalEntryCore{
				PortalEntryRepo: p.PortalEntryRepo,
				PortalRuleRepo:  p.PortalRuleRepo,
				PortalSiteRepo:  p.PortalSiteRepo,
			},
		}
	}
	if p.Logger == nil {
		p.Logger = logger.New("vine:test:initializer")
	}
	return p
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

func (*testPortalEntryRepo) GetBySchemeHostPort(string, string, int) (*core.PortalEntry, bool) {
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
	entryRepo := &testPortalEntryRepo{}
	ruleRepo := &testPortalRuleRepo{
		rules: []*core.PortalRule{
			{Id: 1, Name: "demo-entry", MatchPathPrefix: "/admin", RouteType: "SITE", RouteSiteName: "admin@demo.app", Enabled: true},
		},
	}
	siteRepo := &testPortalSiteRepo{
		entries: []*core.PortalSite{
			{Id: 1, Name: "demo-entry", Type: core.PortalSiteTypeWEBGW, ActorSkelName: "demo.Actor", ActorVia: "client", WebName: "demo.Web", Enabled: true},
		},
	}

	p := &Initializer{
		Syncer: testSyncer(watchServer),
		AppConfigRepo: &testAppConfigRepo{
			items: []*core.AppConfig{
				{Id: 1, Name: "demo.DatabaseConfig", Value: `{"dsn":"postgres://demo"}`},
				{Id: 2, Name: "demo.FeatureConfig", Value: `{"enabled":true}`},
			},
		},
		PortalEntryRepo: entryRepo,
		PortalRuleRepo:  ruleRepo,
		PortalCertRepo: &testPortalCertRepo{
			certs: []*core.PortalCert{
				{Id: 1, Name: "demo-cert", Issuer: "letsencrypt", Domains: []string{"demo.local"}, PublicKeyBase64: "pub", PrivateKeyBase64: "pri", Enabled: true},
			},
		},
		PortalSiteRepo: siteRepo,
		RuleCore: &core.PortalRuleCore{
			PortalRuleRepo: ruleRepo,
			PortalSiteRepo: siteRepo,
			PortalEntryCore: &core.PortalEntryCore{
				PortalEntryRepo: entryRepo,
				PortalRuleRepo:  ruleRepo,
				PortalSiteRepo:  siteRepo,
			},
		},
		SchemaRepo:   schemaRepo,
		RegistryCore: &core.RegistryCore{SchemaRepo: schemaRepo},
		InprocFlag:   &appcore.InternalInprocFlag{},
		Flag:         &hubflag.Flag{AdminListen: "127.0.0.1:7099"},
		Logger:       logger.New("vine:test:initializer"),
	}

	p = withTestAudit(p)
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

	p = withTestAudit(p)
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

	p = withTestAudit(p)
	p.DIInit()

	views := schemaRepo.ListDomainSchemaViews()
	for domain, hubSchema := range hubSchemas {
		got, ok := findDomainSchemaByHash(views, hubSchema.Hash)
		assert.True(t, ok, domain)
		assert.Same(t, hubSchema, got, domain)
	}
}

// Hub applies a seed before the applications register their schemas, so whether
// two rules match one request depends on the Web mount paths Hub reads only
// after those schemas arrive. A read-only Hub has no Dashboard to resolve the
// conflict from, so it refuses to serve the configuration; a stored one keeps
// running and reports what the operator has to fix.
func TestInitializerReportsRulesThatShareOneRequest(t *testing.T) {
	for _, noDB := range []bool{false, true} {
		t.Run(fmt.Sprint(noDB), func(t *testing.T) {
			watchServer := watchserver.NewServerForTest()
			defer watchServer.AfterAppStop()

			ruleRepo := &testPortalRuleRepo{rules: []*core.PortalRule{
				{Id: 1, Name: "demo.app", EntryId: 1, RouteType: "SITE", RouteSiteName: "demo.app-site", Enabled: true},
				{Id: 2, Name: "demo.app-explicit", EntryId: 1, RouteType: "SITE", RouteSiteName: "demo.fixed-site", Enabled: true},
			}}
			siteRepo := &testPortalSiteRepo{entries: []*core.PortalSite{
				{Id: 1, Name: "demo.app-site", Type: core.PortalSiteTypeWEBGW, Enabled: true},
				{Id: 2, Name: "demo.fixed-site", Type: core.PortalSiteTypeWEBGW, Enabled: true},
			}}
			entryRepo := &testPortalEntryRepo{entries: []*core.PortalEntry{
				{Id: 1, Name: "http:80", Scheme: "http", Port: 80, Enabled: true},
			}}
			p := withTestAudit(&Initializer{
				Syncer:          testSyncer(watchServer),
				AppConfigRepo:   &testAppConfigRepo{},
				PortalEntryRepo: entryRepo,
				PortalRuleRepo:  ruleRepo,
				PortalCertRepo:  &testPortalCertRepo{},
				PortalSiteRepo:  siteRepo,
				SchemaRepo:      new(schema.SchemaRepo),
				RegistryCore:    &core.RegistryCore{SchemaRepo: new(schema.SchemaRepo)},
				InprocFlag:      &appcore.InternalInprocFlag{},
				Flag:            &hubflag.Flag{NoDB: noDB, AdminListen: "127.0.0.1:7099"},
			})

			if !noDB {
				require.NotPanics(t, p.DIInit)
				return
			}
			require.PanicsWithError(t,
				`portal rule "demo.app-explicit" and portal rule "demo.app" both match http://*:80: give each request one rule in the seed Hub loads type=APPLICATION code=OPERATION_FAILED`,
				p.DIInit)
		})
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
