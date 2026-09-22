package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/infra/rdb/adapter"
	"gorm.io/gorm"
)

func TestSQLiteRetiredPortalColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "columns.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	testRetiredPortalColumns(t, db, "sqlite")
}

func TestPostgresRetiredPortalColumns(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_columns_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	testRetiredPortalColumns(t, tx, "pgsql")
}

func testRetiredPortalColumns(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	for _, table := range []string{"portal_rule", "portal_site"} {
		sql, err := os.ReadFile("testdata/" + dialect + "/portal_columns_0215_" + table + ".sql")
		require.NoError(t, err)
		require.NoError(t, db.Exec(string(sql)).Error)
	}
	entryDao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	entryDao.EnsureSchema()
	entry := &PortalEntry{Name: "current", Scheme: "https", Host: "current.local", Port: 8443, Enabled: true}
	entryDao.Create(entry)
	// The retired values deliberately disagree with the current entry. Cleanup
	// must preserve the current relationship and path instead of migrating again.
	require.NoError(t, db.Exec(`INSERT INTO portal_rule (name, entry_id, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern, enabled) VALUES ('rule', ?, 'http', 'stale.local', 80, '/kept', 'SITE', 'site', '', FALSE)`, entry.Id).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_site (name, type, actor_skel_name, actor_via, web_name, enabled) VALUES ('site', 'WEBGW', 'demo.Actor', 'client', 'demo.Web', FALSE)`).Error)
	ruleDao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	siteDao := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	for range 2 {
		ruleDao.EnsureSchema()
		siteDao.EnsureSchema()
	}
	columns, err := tableColumnNames(db, "portal_rule")
	require.NoError(t, err)
	for _, name := range []string{"match_scheme", "match_host", "match_port", "built_in"} {
		require.NotContains(t, columns, name)
	}
	columns, err = tableColumnNames(db, "portal_site")
	require.NoError(t, err)
	require.NotContains(t, columns, "built_in")
	rule, ok := ruleDao.ByName("rule")
	require.True(t, ok)
	require.Equal(t, entry.Id, rule.EntryId)
	require.Equal(t, "/kept", rule.MatchPathPrefix)
	require.False(t, rule.Enabled)
	site, ok := siteDao.ByName("site")
	require.True(t, ok)
	require.Equal(t, "demo.Web", site.WebName)
	require.False(t, site.Enabled)
	require.Len(t, entryDao.ListOrdered(), 1)
	rule.MatchPathPrefix = "/updated"
	ruleDao.Save(rule)
	created := ruleDao.Save(&PortalRule{Name: "new", EntryId: entry.Id, MatchPathPrefix: "/new", RouteType: "SITE", RouteSiteName: "site", Enabled: true})
	require.NotZero(t, created.Id)
	stored, ok := ruleDao.ByName("rule")
	require.True(t, ok)
	require.Equal(t, "/updated", stored.MatchPathPrefix)
}

func TestDropColumnsRollsBackWhenColumnIsReferenced(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rollback.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec("CREATE TABLE example (id INTEGER PRIMARY KEY, retired TEXT, indexed TEXT)").Error)
	require.NoError(t, db.Exec("CREATE INDEX keep_index ON example(indexed)").Error)
	require.Panics(t, func() { dropColumns(db, "example", "retired", "indexed") })
	columns, err := tableColumnNames(db, "example")
	require.NoError(t, err)
	require.Contains(t, columns, "retired")
	require.Contains(t, columns, "indexed")
	require.True(t, db.Migrator().HasIndex("example", "keep_index"))
}
