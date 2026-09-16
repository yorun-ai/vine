package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

var (
	testPortalRuleDB     *gorm.DB
	testPortalRuleDBOnce sync.Once
)

func TestPortalRuleDaoCreateAndQuery(t *testing.T) {
	dao := newTestPortalRuleDao(t)

	dao.Create(&PortalRule{
		Name:                    "admin",
		EntryId:                 3,
		MatchPathPrefix:         "/admin",
		RouteType:               "SITE",
		RouteSiteName:           "admin@demo.app",
		RouteRedirectionPattern: "",
	})

	rule, ok := dao.ByName("admin")
	require.True(t, ok)
	assert.Equal(t, 3, rule.EntryId)
	assert.Equal(t, "/admin", rule.MatchPathPrefix)
	assert.Equal(t, "SITE", rule.RouteType)
	assert.Equal(t, "admin@demo.app", rule.RouteSiteName)
}

func TestPortalRuleDaoListOrdered(t *testing.T) {
	dao := newTestPortalRuleDao(t)

	dao.Create(&PortalRule{Name: "z", EntryId: 1, MatchPathPrefix: "", RouteType: "SITE", RouteSiteName: "z"})
	dao.Create(&PortalRule{Name: "a", EntryId: 1, MatchPathPrefix: "/a", RouteType: "SITE", RouteSiteName: "a"})

	rules := dao.ListOrdered()
	require.Len(t, rules, 2)
	assert.Equal(t, "a", rules[0].Name)
	assert.Equal(t, "z", rules[1].Name)
}

func newTestPortalRuleDao(t *testing.T) *PortalRuleDao {
	t.Helper()

	db := sharedTestPortalRuleDB(t)
	dao := &PortalRuleDao{
		Dao: rdb.NewDao[*PortalRule](db),
	}
	dao.InitSchema()
	require.NoError(t, db.Exec("DELETE FROM portal_rule").Error)
	return dao
}

func sharedTestPortalRuleDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalRuleDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-portal-rule-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal-rule.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalRuleDB = db
	})
	return testPortalRuleDB
}

func TestPortalRuleInitSchemaOnCurrentSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "current.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec(createPortalEntrySQLiteSQL).Error)
	require.NoError(t, db.Exec(createPortalRuleSQLiteSQL).Error)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	dao.InitSchema()
	dao.InitSchema()
}

// _legacyPortalRuleSchema is the rule table before entries existed, when every
// rule stored the access it was reached on.
const _legacyPortalRuleSchema = `
CREATE TABLE portal_rule (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    match_scheme TEXT NOT NULL,
    match_host TEXT NOT NULL,
    match_port INTEGER NOT NULL,
    match_path_prefix TEXT NOT NULL,
    route_type TEXT NOT NULL,
    route_site_name TEXT NOT NULL,
    route_redirection_pattern TEXT NOT NULL,
    route_path_prefix TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX uk_portal_rule_match
    ON portal_rule(match_scheme, match_host, match_port, match_path_prefix);

CREATE UNIQUE INDEX uk_portal_rule_name
    ON portal_rule(name);
`

func TestPortalRuleInitSchemaMigratesRuleAccessToEntry(t *testing.T) {
	db := newTestLegacyPortalRuleDB(t)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}

	dao.InitSchema()
	dao.InitSchema()

	// One entry per stored access, with the effective port Portal serves.
	stored := (&PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}).ListOrdered()
	require.Len(t, stored, 3)
	entries := map[string]*PortalEntry{}
	for _, entry := range stored {
		entries[fmt.Sprintf("%s:%s:%d", entry.Scheme, entry.Host, entry.Port)] = entry
		assert.False(t, entry.BuiltIn)
	}
	require.Contains(t, entries, "http::80")
	require.Contains(t, entries, "http:demo.local:8080")
	require.Contains(t, entries, "https::443")

	rules := dao.ListOrdered()
	require.Len(t, rules, 5)
	byName := map[string]*PortalRule{}
	for _, rule := range rules {
		byName[rule.Name] = rule
	}
	// Rules keep their own columns and gain the entry that owns the access.
	assert.Equal(t, entries["http::80"].Id, byName["web"].EntryId)
	assert.Equal(t, entries["http::80"].Id, byName["api"].EntryId)
	assert.Equal(t, entries["http:demo.local:8080"].Id, byName["hosted"].EntryId)
	assert.Equal(t, entries["https::443"].Id, byName["secure"].EntryId)
	assert.Equal(t, entries["https::443"].Id, byName["vine.hub.dashboard-web"].EntryId)
	assert.Equal(t, "/", byName["web"].MatchPathPrefix)
	assert.True(t, byName["vine.hub.dashboard-web"].BuiltIn)

	// The access columns and their index are gone from the rule table.
	columns, err := tableColumnNames(db, &PortalRule{})
	require.NoError(t, err)
	assert.False(t, columns["match_scheme"])
	assert.False(t, columns["match_host"])
	assert.False(t, columns["match_port"])
	assert.True(t, columns["entry_id"])
	assert.False(t, db.Migrator().HasIndex("portal_rule", "uk_portal_rule_match"))
	assert.True(t, db.Migrator().HasIndex("portal_rule", "uk_portal_rule_entry_path"))
}

func TestPortalRuleInitSchemaMovesDefaultPortRuleToMigratedPath(t *testing.T) {
	// The stored port kept these rules apart, but all three resolve to the same
	// access and share a path, so the migration must separate them instead of
	// refusing to start on data the operator cannot edit.
	db := newTestLegacyPortalRuleDB(t,
		`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
		 VALUES ('a', 'https', '', 0, '/app', 'SITE', 'web', '')`,
		`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
		 VALUES ('b', 'https', '', 443, '/app', 'SITE', 'web', '')`,
		`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
		 VALUES ('c', 'https', '', 443, '/app/migrated', 'SITE', 'web', '')`,
	)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}

	dao.InitSchema()

	byName := map[string]*PortalRule{}
	for _, rule := range dao.ListOrdered() {
		byName[rule.Name] = rule
	}
	require.Len(t, byName, 3)
	assert.Equal(t, "/app", byName["b"].MatchPathPrefix)
	assert.Equal(t, "/app/migrated", byName["c"].MatchPathPrefix)
	// Rule "a" left the port unset, so it moves aside, and the first free path
	// inside its entry wins.
	assert.Equal(t, "/app/migrated/migrated", byName["a"].MatchPathPrefix)
	assert.Equal(t, byName["b"].EntryId, byName["a"].EntryId)
	assert.Equal(t, byName["c"].EntryId, byName["a"].EntryId)
}

func newTestLegacyPortalRuleDB(t *testing.T, statements ...string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec(_legacyPortalRuleSchema).Error)
	if len(statements) == 0 {
		statements = []string{
			`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
			 VALUES ('web', 'http', '', 0, '/', 'SITE', 'web-site', '')`,
			`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
			 VALUES ('api', 'http', '', 0, '/api', 'SITE', 'rpc-site', '')`,
			`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
			 VALUES ('hosted', 'http', 'demo.local', 8080, '/', 'SITE', 'web-site', '')`,
			`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
			 VALUES ('secure', 'https', '', 443, '/', 'SITE', 'web-site', '')`,
			`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern, built_in)
			 VALUES ('vine.hub.dashboard-web', 'https', '', 443, '/hub', 'SITE', 'dashboard-web', '', TRUE)`,
		}
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}
