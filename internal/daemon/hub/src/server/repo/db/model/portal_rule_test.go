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
		Name:               "admin",
		Scheme:             "https",
		Host:               "example.com",
		Port:               443,
		PathPrefix:         "/admin",
		TargetType:         "SITE",
		SiteName:           "admin@demo.app",
		RedirectionPattern: "",
	})

	rule, ok := dao.ByName("admin")
	require.True(t, ok)
	assert.Equal(t, "https", rule.Scheme)
	assert.Equal(t, "example.com", rule.Host)
	assert.Equal(t, 443, rule.Port)
	assert.Equal(t, "/admin", rule.PathPrefix)
	assert.Equal(t, "SITE", rule.TargetType)
	assert.Equal(t, "admin@demo.app", rule.SiteName)
}

func TestPortalRuleDaoListOrdered(t *testing.T) {
	dao := newTestPortalRuleDao(t)

	dao.Create(&PortalRule{Name: "z", Scheme: "https", Host: "", PathPrefix: "", TargetType: "SITE", SiteName: "z"})
	dao.Create(&PortalRule{Name: "a", Scheme: "https", Host: "", PathPrefix: "/a", TargetType: "SITE", SiteName: "a"})

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
