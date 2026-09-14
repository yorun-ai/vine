package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testConfigDB     *gorm.DB
	testConfigDBOnce sync.Once
)

// DB config repo

func TestDBAppConfigRepoListItems(t *testing.T) {
	_, repo, _ := newTestDBAppConfigRepo(t)
	repo.SaveItem(testAppConfig("db.main", `{"connUrl":"postgres://demo","maxPoolSize":8}`, 1))
	repo.SaveItem(testAppConfig("feature.flag", `{"enabled":true}`, 2))

	items := repo.ListItems()
	require.Len(t, items, 2)
	assert.Equal(t, `{"connUrl":"postgres://demo","maxPoolSize":8}`, items[0].Value)
	assert.Equal(t, "feature.flag", items[1].Name)
	assert.Equal(t, 2, items[1].Version)
}

func TestDBAppConfigRepoSaveItemCreate(t *testing.T) {
	_, repo, watchServer := newTestDBAppConfigRepo(t)

	item := testAppConfig("db.main", `{"connUrl":"postgres://demo"}`, 1)
	repo.SaveItem(item)

	item, ok := repo.GetItemById(item.Id)
	require.True(t, ok)
	assert.NotZero(t, item.Id)
	assert.Equal(t, `{"connUrl":"postgres://demo"}`, item.Value)
	assert.Equal(t, 1, item.Version)

	key := watched.FormatConfigKey("db.main")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, &watched.ConfigValue{Name: "db.main", Value: []byte(`{"connUrl":"postgres://demo"}`)}, vcode.MustUnmarshalJsonS[*watched.ConfigValue](raw))
}

func TestDBAppConfigRepoSaveItemDuplicateNameWithoutId(t *testing.T) {
	_, repo, _ := newTestDBAppConfigRepo(t)

	repo.SaveItem(testAppConfig("feature.flag", `{"enabled":true}`, 1))

	assert.Panics(t, func() {
		repo.SaveItem(testAppConfig("feature.flag", `{"enabled":false}`, 1))
	})
}

func TestDBAppConfigRepoSaveItemUpdate(t *testing.T) {
	_, repo, watchServer := newTestDBAppConfigRepo(t)

	item := testAppConfig("feature.flag", `{"enabled":true}`, 1)
	repo.SaveItem(item)
	item.Value = `{"enabled":false}`
	item.Version = 2
	repo.SaveItem(item)

	item, ok := repo.GetItemById(item.Id)
	require.True(t, ok)
	assert.Equal(t, `{"enabled":false}`, item.Value)
	assert.Equal(t, 2, item.Version)

	key := watched.FormatConfigKey("feature.flag")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, &watched.ConfigValue{Name: "feature.flag", Value: []byte(`{"enabled":false}`)}, vcode.MustUnmarshalJsonS[*watched.ConfigValue](raw))
}

func TestDBAppConfigRepoRemoveItem(t *testing.T) {
	_, repo, watchServer := newTestDBAppConfigRepo(t)
	item := testAppConfig("feature.flag", `{"enabled":true}`, 1)
	repo.SaveItem(item)
	id := item.Id

	assert.True(t, repo.RemoveItem(id))

	item, ok := repo.GetItemById(id)
	assert.False(t, ok)
	assert.Nil(t, item)
	assert.Empty(t, repo.ListItems())

	key := watched.FormatConfigKey("feature.flag")
	_, ok = watchServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.RemoveItem(id))
}

// Helpers

func newTestDBAppConfigRepo(t *testing.T) (*gorm.DB, *DBAppConfigRepo, *watchserver.Server) {
	t.Helper()

	db := sharedTestConfigDB(t)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	repo := &DBAppConfigRepo{
		Dao: &model.AppConfigDao{
			Dao: rdb.NewDao[*model.AppConfig](db),
		},
		Syncer: testSyncer(watchServer),
		Access: new(configaccess.Access),
	}
	repo.Dao.InitSchema()
	require.NoError(t, db.Exec("DELETE FROM app_config").Error)

	return db, repo, watchServer
}

func sharedTestConfigDB(t *testing.T) *gorm.DB {
	t.Helper()

	testConfigDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-config-repo-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "config.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testConfigDB = db
	})
	return testConfigDB
}

func testAppConfig(name string, value string, version int) *core.AppConfig {
	return &core.AppConfig{
		Name:    name,
		Value:   value,
		Version: version,
	}
}

func TestDBAppConfigRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &DBAppConfigRepo{Access: access}
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.SaveItem(new(core.AppConfig)) })
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.RemoveItem(1) })
}
