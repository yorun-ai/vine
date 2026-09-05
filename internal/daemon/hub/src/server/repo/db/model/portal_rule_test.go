package model

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/infra/rdb"
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
		MatchScheme:             "https",
		MatchHost:               "example.com",
		MatchPort:               443,
		MatchPathPrefix:         "/admin",
		RouteType:               "SITE",
		RouteSiteName:           "admin@demo.app",
		RouteRedirectionPattern: "",
	})

	rule, ok := dao.ByName("admin")
	require.True(t, ok)
	assert.Equal(t, "https", rule.MatchScheme)
	assert.Equal(t, "example.com", rule.MatchHost)
	assert.Equal(t, 443, rule.MatchPort)
	assert.Equal(t, "/admin", rule.MatchPathPrefix)
	assert.Equal(t, "SITE", rule.RouteType)
	assert.Equal(t, "admin@demo.app", rule.RouteSiteName)
}

func TestPortalRuleDaoListOrdered(t *testing.T) {
	dao := newTestPortalRuleDao(t)

	dao.Create(&PortalRule{Name: "z", MatchScheme: "https", MatchHost: "", MatchPathPrefix: "", RouteType: "SITE", RouteSiteName: "z"})
	dao.Create(&PortalRule{Name: "a", MatchScheme: "https", MatchHost: "", MatchPathPrefix: "/a", RouteType: "SITE", RouteSiteName: "a"})

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
	dao.DIInit()
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

func TestPortalRuleTargetPathMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	legacySQL := strings.NewReplacer(
		"match_scheme", "scheme", "match_host", "host", "match_port", "port",
		"match_path_prefix", "path_prefix", "route_type", "target_type",
		"route_site_name", "site_name", "route_path_prefix", "target_path",
		"route_redirection_pattern", "redirection_pattern",
	).Replace(createPortalRuleSQLiteSQL)
	legacySQL = strings.ReplaceAll(legacySQL, "    target_path TEXT NOT NULL DEFAULT '',\n", "")
	require.NoError(t, db.Exec(legacySQL).Error)
	require.NoError(t, db.Exec("INSERT INTO portal_rule(name, scheme, host, port, path_prefix, target_type, site_name, redirection_pattern, built_in) VALUES ('legacy', 'http', '', 80, '/api', 'SITE', 'site', '', false)").Error)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	require.NoError(t, dao.migrateSchema())
	require.NoError(t, dao.migrateSchema())
	row, ok := dao.ByName("legacy")
	require.True(t, ok)
	assert.Empty(t, row.RoutePathPrefix)
	row.RoutePathPrefix = "/internal"
	dao.Save(row)
	row, _ = dao.ByName("legacy")
	assert.Equal(t, "/internal", row.RoutePathPrefix)
	row.RoutePathPrefix = ""
	dao.Save(row)
	row, _ = dao.ByName("legacy")
	assert.Empty(t, row.RoutePathPrefix)
}

func TestPortalRuleColumnRenamePreservesDataAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-with-target.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	legacySQL := strings.NewReplacer(
		"match_scheme", "scheme", "match_host", "host", "match_port", "port",
		"match_path_prefix", "path_prefix", "route_type", "target_type",
		"route_site_name", "site_name", "route_path_prefix", "target_path",
		"route_redirection_pattern", "redirection_pattern",
	).Replace(createPortalRuleSQLiteSQL)
	require.NoError(t, db.Exec(legacySQL).Error)
	require.NoError(t, db.Exec("INSERT INTO portal_rule(id, name, scheme, host, port, path_prefix, target_type, site_name, target_path, redirection_pattern, built_in) VALUES (17, 'legacy', 'https', 'example.com', 443, '/api', 'SITE', 'web', '/internal', '', true), (18, 'redirect', 'http', 'old.example.com', 80, '/', 'PERMANENT_REDIRECT', '', '', 'https://example.com', false)").Error)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	require.NoError(t, dao.migrateSchema())
	require.NoError(t, dao.migrateSchema())
	row, ok := dao.ByName("legacy")
	require.True(t, ok)
	assert.Equal(t, 17, row.Id)
	assert.Equal(t, "https", row.MatchScheme)
	assert.Equal(t, "example.com", row.MatchHost)
	assert.Equal(t, 443, row.MatchPort)
	assert.Equal(t, "/api", row.MatchPathPrefix)
	assert.Equal(t, "SITE", row.RouteType)
	assert.Equal(t, "web", row.RouteSiteName)
	assert.Equal(t, "/internal", row.RoutePathPrefix)
	assert.True(t, row.BuiltIn)
	redirect, ok := dao.ByName("redirect")
	require.True(t, ok)
	assert.Equal(t, "PERMANENT_REDIRECT", redirect.RouteType)
	assert.Equal(t, "https://example.com", redirect.RouteRedirectionPattern)
	for _, old := range []string{"scheme", "host", "port", "path_prefix", "target_type", "site_name", "target_path", "redirection_pattern"} {
		assert.False(t, db.Migrator().HasColumn("portal_rule", old), old)
	}
	duplicate := *row
	duplicate.Id = 0
	duplicate.Name = "different-name"
	require.Error(t, db.Create(&duplicate).Error, "match uniqueness must survive migration")
	duplicate.MatchHost = "different.example.com"
	duplicate.Name = row.Name
	require.Error(t, db.Create(&duplicate).Error, "name uniqueness must survive migration")
}
