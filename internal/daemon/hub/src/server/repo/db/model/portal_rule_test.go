package model

import (
	"os"
	"path/filepath"
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
	require.NoError(t, db.Exec(createPortalRuleSQLiteSQL).Error)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	dao.InitSchema()
	dao.InitSchema()
}
