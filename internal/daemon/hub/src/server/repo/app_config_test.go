package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
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

func TestAppConfigRepoListItems(t *testing.T) {
	_, repo, _ := newTestAppConfigRepo(t)
	repo.Save(testAppConfig("db.main", `{"connUrl":"postgres://demo","maxPoolSize":8}`, 1))
	repo.Save(testAppConfig("feature.flag", `{"enabled":true}`, 2))

	items := repo.List()
	require.Len(t, items, 2)
	assert.Equal(t, `{"connUrl":"postgres://demo","maxPoolSize":8}`, items[0].Value)
	assert.Equal(t, "feature.flag", items[1].Name)
	assert.Equal(t, 2, items[1].Version)
}

func TestAppConfigRepoSaveItemCreate(t *testing.T) {
	_, repo, watchServer := newTestAppConfigRepo(t)

	item := testAppConfig("db.main", `{"connUrl":"postgres://demo"}`, 1)
	repo.Save(item)

	item, ok := repo.GetById(item.Id)
	require.True(t, ok)
	assert.NotZero(t, item.Id)
	assert.Equal(t, `{"connUrl":"postgres://demo"}`, item.Value)
	assert.Equal(t, 1, item.Version)

	key := watched.FormatConfigKey("db.main")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, &watched.ConfigValue{Name: "db.main", Value: []byte(`{"connUrl":"postgres://demo"}`)}, vcode.MustUnmarshalJsonS[*watched.ConfigValue](raw))
}

func TestAppConfigRepoSaveItemDuplicateNameWithoutId(t *testing.T) {
	_, repo, _ := newTestAppConfigRepo(t)

	repo.Save(testAppConfig("feature.flag", `{"enabled":true}`, 1))

	assert.Panics(t, func() {
		repo.Save(testAppConfig("feature.flag", `{"enabled":false}`, 1))
	})
}

func TestAppConfigRepoSaveItemUpdate(t *testing.T) {
	_, repo, watchServer := newTestAppConfigRepo(t)

	item := testAppConfig("feature.flag", `{"enabled":true}`, 1)
	repo.Save(item)
	item.Value = `{"enabled":false}`
	item.Version = 2
	repo.Save(item)

	item, ok := repo.GetById(item.Id)
	require.True(t, ok)
	assert.Equal(t, `{"enabled":false}`, item.Value)
	assert.Equal(t, 2, item.Version)

	key := watched.FormatConfigKey("feature.flag")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, &watched.ConfigValue{Name: "feature.flag", Value: []byte(`{"enabled":false}`)}, vcode.MustUnmarshalJsonS[*watched.ConfigValue](raw))
}

func TestAppConfigRepoRemoveItem(t *testing.T) {
	_, repo, watchServer := newTestAppConfigRepo(t)
	item := testAppConfig("feature.flag", `{"enabled":true}`, 1)
	repo.Save(item)
	id := item.Id

	assert.True(t, repo.Remove(id))

	item, ok := repo.GetById(id)
	assert.False(t, ok)
	assert.Nil(t, item)
	assert.Empty(t, repo.List())

	key := watched.FormatConfigKey("feature.flag")
	_, ok = watchServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.Remove(id))
}

// Helpers

func newTestAppConfigRepo(t *testing.T) (*gorm.DB, *AppConfigRepo, *watchserver.Server) {
	t.Helper()

	db := sharedTestConfigDB(t)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	repo := &AppConfigRepo{
		Dao: &model.AppConfigDao{
			Dao: rdb.NewDao[*model.AppConfig](db),
		},
		SchemaRepo: new(_PortalSiteSchemaRepo),
		Syncer:     testSyncer(watchServer),
		Access:     new(configaccess.Access),
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

func TestAppConfigRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &AppConfigRepo{Access: access}
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Save(new(core.AppConfig)) })
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Remove(1) })
}

func TestAppConfigRepoSlotsAssembleDeclaredAndStoredConfigs(t *testing.T) {
	_, repo, _ := newTestAppConfigRepo(t)
	schemas := repo.SchemaRepo.(*_PortalSiteSchemaRepo)
	schemas.enumSchemas = []*skel.EnumSchema{{
		SkelName: "demo.Mode",
		Items:    []*skel.EnumItemSchema{{Name: "FAST", Description: "Fast"}, {Name: "SAFE"}},
	}}
	schemas.configSchemas = []*skel.ConfigSchema{
		{
			Name:      "FeatureConfig",
			SkelName:  "demo.FeatureConfig",
			Lifecycle: "ETERNAL",
			Members: []*skel.MemberSchema{
				{Name: "enabled", Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool}},
				{Name: "mode", Description: "Run mode", Type: &skel.TypeSchema{Kind: skel.TypeKindEnum, SkelName: "demo.Mode"}},
			},
		},
		{Name: "OtherConfig", SkelName: "demo.OtherConfig", Lifecycle: "INSTANT"},
		{
			Name:     "MismatchedConfig",
			SkelName: "demo.MismatchedConfig",
			Members: []*skel.MemberSchema{
				{Name: "enabled", Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool}},
			},
		},
	}

	repo.Save(testAppConfig("demo.FeatureConfig", `{"enabled":true,"mode":"FAST"}`, 2))
	repo.Save(testAppConfig("demo.MismatchedConfig", `{"enabled":"yes"}`, 1))
	repo.Save(testAppConfig("demo.LegacyConfig", `{}`, 1))

	slots := map[string]*core.AppConfig{}
	for _, slot := range repo.ListSlots() {
		slots[slot.Name] = slot
	}
	require.Len(t, slots, 4)

	configured := slots["demo.FeatureConfig"]
	require.NotNil(t, configured)
	assert.True(t, configured.Configured)
	assert.Equal(t, core.AppConfigStatusNormal, configured.Status)
	assert.Equal(t, "ETERNAL", configured.Lifecycle)
	require.NotNil(t, configured.Definition)
	require.Len(t, configured.Definition.Fields, 2)
	assert.Equal(t, "Run mode", configured.Definition.Fields[1].Description)
	require.Len(t, configured.Definition.Fields[1].EnumItems, 2)
	assert.Equal(t, "FAST", configured.Definition.Fields[1].EnumItems[0].Name)

	// A configuration an application declares without a stored value has no
	// storage identity and no provenance.
	declared := slots["demo.OtherConfig"]
	require.NotNil(t, declared)
	assert.False(t, declared.Configured)
	assert.Zero(t, declared.Id)
	assert.Equal(t, core.AppConfigStatusUnconfigured, declared.Status)
	assert.Equal(t, "INSTANT", declared.Lifecycle)
	assert.Empty(t, declared.FieldSources)

	// A stored value that does not match its declaration is reported as such.
	mismatched := slots["demo.MismatchedConfig"]
	require.NotNil(t, mismatched)
	assert.True(t, mismatched.Configured)
	assert.Equal(t, core.AppConfigStatusMismatch, mismatched.Status)
	require.NotNil(t, mismatched.Definition)

	// A stored value without a declaration stays usable but unused.
	unused := slots["demo.LegacyConfig"]
	require.NotNil(t, unused)
	assert.True(t, unused.Configured)
	assert.Equal(t, core.AppConfigStatusUnused, unused.Status)
	assert.Nil(t, unused.Definition)

	// The stored list keeps serving downstream sync and excludes declared slots.
	items := repo.List()
	require.Len(t, items, 3)
	for _, item := range items {
		assert.NotEqual(t, "demo.OtherConfig", item.Name)
	}

	slot, ok := repo.FindByName("demo.OtherConfig")
	require.True(t, ok)
	assert.False(t, slot.Configured)
	_, ok = repo.FindByName("demo.Missing")
	assert.False(t, ok)
}
