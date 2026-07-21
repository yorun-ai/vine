package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalRuleRepoDB     *gorm.DB
	testPortalRuleRepoDBOnce sync.Once
)

func TestDBPortalRuleRepoSaveRuleCreate(t *testing.T) {
	_, repo, redisServer := newTestDBPortalRuleRepo(t)

	rule := testPortalRule("admin")
	repo.SaveRule(rule)

	got, ok := repo.GetRuleById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, rule, got)

	key := redised.FormatPortalRuleKey("admin")
	raw, ok := redisServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, syncer.ToRedisedPortalRule(rule), vcode.MustUnmarshalJsonS[*redised.PortalRule](raw))
}

func TestDBPortalRuleRepoSaveRuleUpdate(t *testing.T) {
	_, repo, _ := newTestDBPortalRuleRepo(t)

	rule := testPortalRule("admin")
	repo.SaveRule(rule)
	rule.PathPrefix = "/console"
	rule.SiteName = "console@demo.app"
	repo.SaveRule(rule)

	got, ok := repo.GetRuleById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, "/console", got.PathPrefix)
	assert.Equal(t, "console@demo.app", got.SiteName)
}

func TestDBPortalRuleRepoSaveRuleBuiltIn(t *testing.T) {
	db, repo, _ := newTestDBPortalRuleRepo(t)

	rule := testPortalRule("admin")
	rule.BuiltIn = true
	repo.SaveRule(rule)

	var row model.PortalRule
	require.NoError(t, db.First(&row, "id = ?", rule.Id).Error)
	assert.True(t, row.BuiltIn)

	got, ok := repo.GetRuleById(rule.Id)
	require.True(t, ok)
	assert.True(t, got.BuiltIn)
}

func TestDBPortalRuleRepoSaveRuleRename(t *testing.T) {
	_, repo, redisServer := newTestDBPortalRuleRepo(t)

	rule := testPortalRule("admin")
	repo.SaveRule(rule)
	rule.Name = "console"
	repo.SaveRule(rule)

	got, ok := repo.GetRuleById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, "console", got.Name)

	_, ok = redisServer.Get(redised.FormatPortalRuleKey("admin"))
	assert.False(t, ok)

	raw, ok := redisServer.Get(redised.FormatPortalRuleKey("console"))
	require.True(t, ok)
	assert.Equal(t, syncer.ToRedisedPortalRule(rule), vcode.MustUnmarshalJsonS[*redised.PortalRule](raw))
}

func TestDBPortalRuleRepoRemoveRule(t *testing.T) {
	_, repo, redisServer := newTestDBPortalRuleRepo(t)

	rule := testPortalRule("admin")
	repo.SaveRule(rule)
	assert.True(t, repo.RemoveRule(rule.Id))

	got, ok := repo.GetRuleById(rule.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	key := redised.FormatPortalRuleKey("admin")
	_, ok = redisServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.RemoveRule(rule.Id))
}

func newTestDBPortalRuleRepo(t *testing.T) (*gorm.DB, *DBPortalRuleRepo, *redisserver.Server) {
	t.Helper()

	db := sharedTestPortalRuleRepoDB(t)
	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	repo := &DBPortalRuleRepo{
		Dao: &model.PortalRuleDao{
			Dao: rdb.NewDao[*model.PortalRule](db),
		},
		Syncer: testSyncer(redisServer),
	}
	repo.Dao.DIInit()
	require.NoError(t, db.Exec("DELETE FROM portal_rule").Error)

	return db, repo, redisServer
}

func sharedTestPortalRuleRepoDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalRuleRepoDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-portal-rule-repo-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal_rule.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalRuleRepoDB = db
	})
	return testPortalRuleRepoDB
}

func testPortalRule(name string) *core.PortalRule {
	return &core.PortalRule{
		Name:               name,
		Scheme:             "https",
		Host:               "demo.local",
		Port:               443,
		PathPrefix:         "/admin",
		TargetType:         "SITE",
		SiteName:           "admin@demo.app",
		RedirectionPattern: "",
	}
}
