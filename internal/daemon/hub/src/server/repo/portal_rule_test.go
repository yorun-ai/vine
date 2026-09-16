package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalRuleRepoDB     *gorm.DB
	testPortalRuleRepoDBOnce sync.Once
)

func TestPortalRuleRepoSaveCreate(t *testing.T) {
	_, repo, watchServer := newTestPortalRuleRepo(t)

	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)

	got, ok := repo.GetById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, rule, got)

	key := watched.FormatPortalRuleKey("admin")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, syncer.ToWatchedPortalRule(rule, portalRuleEntry(t, repo, rule)), vcode.MustUnmarshalJsonS[*watched.PortalRule](raw))
}

func TestPortalRuleRepoSaveUpdate(t *testing.T) {
	_, repo, _ := newTestPortalRuleRepo(t)

	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)
	rule.MatchPathPrefix = "/console"
	rule.RouteSiteName = "console@demo.app"
	repo.Save(rule)

	got, ok := repo.GetById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, "/console", got.MatchPathPrefix)
	assert.Equal(t, "console@demo.app", got.RouteSiteName)
}

func TestPortalRuleRepoSaveKeepsDeprecatedAccessColumns(t *testing.T) {
	db, repo, _ := newTestPortalRuleRepo(t)

	// The rule reads its access from the entry, and the deprecated columns stay
	// filled with that access until Hub removes them.
	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)

	var row model.PortalRule
	require.NoError(t, db.First(&row, "id = ?", rule.Id).Error)
	assert.Equal(t, "https", row.MatchScheme)
	assert.Equal(t, "demo.local", row.MatchHost)
	assert.Equal(t, 443, row.MatchPort)
}

func TestPortalRuleRepoSaveRename(t *testing.T) {
	_, repo, watchServer := newTestPortalRuleRepo(t)

	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)
	rule.Name = "console"
	repo.Save(rule)

	got, ok := repo.GetById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, "console", got.Name)

	_, ok = watchServer.Get(watched.FormatPortalRuleKey("admin"))
	assert.False(t, ok)

	raw, ok := watchServer.Get(watched.FormatPortalRuleKey("console"))
	require.True(t, ok)
	assert.Equal(t, syncer.ToWatchedPortalRule(rule, portalRuleEntry(t, repo, rule)), vcode.MustUnmarshalJsonS[*watched.PortalRule](raw))
}

func TestPortalRuleRepoRemove(t *testing.T) {
	_, repo, watchServer := newTestPortalRuleRepo(t)

	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)
	assert.True(t, repo.Remove(rule.Id))

	got, ok := repo.GetById(rule.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	key := watched.FormatPortalRuleKey("admin")
	_, ok = watchServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.Remove(rule.Id))
}

// TestPortalRuleRepoPublishesEntryAccess covers the entry contract: a rule owns
// the entry it belongs to, and the access of that entry is what Portal receives.
func TestPortalRuleRepoPublishesEntryAccess(t *testing.T) {
	_, repo, watchServer := newTestPortalRuleRepo(t)

	rule := testPortalRule(t, repo, "admin")
	repo.Save(rule)

	entry, ok := repo.PortalEntryRepo.GetById(rule.EntryId)
	require.True(t, ok)
	entry.Scheme = "http"
	entry.Host = "app.example.com"
	entry.Port = 8080
	repo.PortalEntryRepo.Save(entry)

	got, ok := repo.GetById(rule.Id)
	require.True(t, ok)
	assert.Equal(t, rule.EntryId, got.EntryId)

	// Changing the entry republishes the rules it routes with the new access.
	raw, ok := watchServer.Get(watched.FormatPortalRuleKey("admin"))
	require.True(t, ok)
	assert.Equal(t, syncer.ToWatchedPortalRule(got, entry), vcode.MustUnmarshalJsonS[*watched.PortalRule](raw))
}

// portalRuleEntry returns the entry a rule belongs to.
func portalRuleEntry(t *testing.T, repo *PortalRuleRepo, rule *core.PortalRule) *core.PortalEntry {
	t.Helper()
	entry, ok := repo.PortalEntryRepo.GetById(rule.EntryId)
	require.True(t, ok)
	return entry
}

func newTestPortalRuleRepo(t *testing.T) (*gorm.DB, *PortalRuleRepo, *watchserver.Server) {
	t.Helper()

	db := sharedTestPortalRuleRepoDB(t)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	// One syncer serves both repositories, the way the injector hands Hub a
	// single one: a rule publishes the access of the entry it belongs to.
	sync := testSyncer(watchServer)
	entryRepo := newTestPortalEntryRepo(db, sync)
	entryRepo.Dao.InitSchema()
	repo := &PortalRuleRepo{
		Dao: &model.PortalRuleDao{
			Dao: rdb.NewDao[*model.PortalRule](db),
		},
		Syncer:          sync,
		Access:          new(configaccess.Access),
		PortalEntryRepo: entryRepo,
	}
	repo.Dao.InitSchema()
	require.NoError(t, db.Exec("DELETE FROM portal_rule").Error)
	require.NoError(t, db.Exec("DELETE FROM portal_entry").Error)

	return db, repo, watchServer
}

func newTestPortalEntryRepo(db *gorm.DB, sync *syncer.Syncer) *PortalEntryRepo {
	return &PortalEntryRepo{
		Dao:    &model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](db)},
		Syncer: sync,
		Access: new(configaccess.Access),
	}
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

// testPortalRule builds a rule that belongs to the entry serving the access, the
// way Core hands a complete rule to the repository.
func testPortalRule(t *testing.T, repo *PortalRuleRepo, name string) *core.PortalRule {
	t.Helper()

	entry, ok := repo.PortalEntryRepo.GetByAccess("https", "demo.local", 443)
	if !ok {
		entry = &core.PortalEntry{Scheme: "https", Host: "demo.local", Port: 443, Enabled: true}
		repo.PortalEntryRepo.Save(entry)
	}
	return &core.PortalRule{
		Name:                    name,
		EntryId:                 entry.Id,
		MatchPathPrefix:         "/admin",
		RouteType:               "SITE",
		RouteSiteName:           "admin@demo.app",
		RouteRedirectionPattern: "",
		Enabled:                 true,
	}
}

func TestPortalRuleRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &PortalRuleRepo{Access: access}
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Save(new(core.PortalRule)) })
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Remove(1) })
}
