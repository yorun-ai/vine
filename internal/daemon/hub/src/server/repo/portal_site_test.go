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
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalSiteRepoDB     *gorm.DB
	testPortalSiteRepoDBOnce sync.Once
)

type _PortalSiteSchemaRepo struct{}

func (*_PortalSiteSchemaRepo) SaveDomainSchemas(string, string, []*skel.DomainSchema) {
}

func (*_PortalSiteSchemaRepo) SaveDomainSchemasJSON(string, string, []skel.JSON) {
}

func (*_PortalSiteSchemaRepo) ReleaseDomainSchemas(string, string) {
}

func (*_PortalSiteSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	return nil
}

func (*_PortalSiteSchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	return nil
}

func (*_PortalSiteSchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	return nil
}

func (*_PortalSiteSchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	return nil
}

func (*_PortalSiteSchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	return nil
}

func (*_PortalSiteSchemaRepo) ListWebSchemas() []*skel.WebSchema {
	return nil
}

func (*_PortalSiteSchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	return nil
}

func (*_PortalSiteSchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	return nil
}

func TestDBPortalSiteRepoSaveEntryCreate(t *testing.T) {
	_, repo, redisServer := newTestDBPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.SaveEntry(entry)

	got, ok := repo.GetEntryById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry, got)

	raw, ok := redisServer.Get(redised.FormatPortalSiteKey("demo-entry"))
	require.True(t, ok)
	assertRedisedPortalSite(t, entry, vcode.MustUnmarshalJsonS[*redised.PortalSite](raw))
}

func TestDBPortalSiteRepoSaveEntryBuiltIn(t *testing.T) {
	db, repo, _ := newTestDBPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	entry.BuiltIn = true
	repo.SaveEntry(entry)

	var row model.PortalSite
	require.NoError(t, db.First(&row, "id = ?", entry.Id).Error)
	assert.True(t, row.BuiltIn)

	got, ok := repo.GetEntryById(entry.Id)
	require.True(t, ok)
	assert.True(t, got.BuiltIn)
}

func TestDBPortalSiteRepoSaveEntryRename(t *testing.T) {
	_, repo, redisServer := newTestDBPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.SaveEntry(entry)
	entry.Name = "next-entry"
	repo.SaveEntry(entry)

	_, ok := redisServer.Get(redised.FormatPortalSiteKey("demo-entry"))
	assert.False(t, ok)

	raw, ok := redisServer.Get(redised.FormatPortalSiteKey("next-entry"))
	require.True(t, ok)
	assertRedisedPortalSite(t, entry, vcode.MustUnmarshalJsonS[*redised.PortalSite](raw))
}

func TestDBPortalSiteRepoRemoveEntry(t *testing.T) {
	_, repo, redisServer := newTestDBPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.SaveEntry(entry)
	assert.True(t, repo.RemoveEntry(entry.Id))

	got, ok := repo.GetEntryById(entry.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	_, ok = redisServer.Get(redised.FormatPortalSiteKey("demo-entry"))
	assert.False(t, ok)
	assert.False(t, repo.RemoveEntry(entry.Id))
}

func newTestDBPortalSiteRepo(t *testing.T) (*gorm.DB, *DBPortalSiteRepo, *redisserver.Server) {
	t.Helper()

	db := sharedTestPortalSiteRepoDB(t)
	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	repo := &DBPortalSiteRepo{
		Dao: &model.PortalSiteDao{
			Dao: rdb.NewDao[*model.PortalSite](db),
		},
		SchemaRepo: &_PortalSiteSchemaRepo{},
		Syncer:     testSyncer(redisServer),
	}
	repo.Dao.DIInit()
	require.NoError(t, db.Exec("DELETE FROM portal_site").Error)

	return db, repo, redisServer
}

func sharedTestPortalSiteRepoDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalSiteRepoDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-portal-site-repo-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal_site.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalSiteRepoDB = db
	})
	return testPortalSiteRepoDB
}

func testPortalSite(name string) *core.PortalSite {
	return &core.PortalSite{
		Name:          name,
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "demo.Actor",
		ActorVia:      "client",
		Cors: core.PortalCors{
			Mode:           core.PortalCorsModeStrict,
			AllowedOrigins: []string{"https://console.example.com"},
		},
	}
}

func assertRedisedPortalSite(t *testing.T, expected *core.PortalSite, actual *redised.PortalSite) {
	t.Helper()

	require.NotNil(t, actual)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, string(expected.Type), actual.Type)
	assert.Equal(t, expected.ActorSkelName, actual.ActorVia.ActorSkelName)
	assert.Equal(t, expected.ActorVia, actual.ActorVia.ActorVia)
	assert.Equal(t, redised.PortalCorsMode(expected.Cors.Mode), actual.Cors.Mode)
	assert.Equal(t, expected.Cors.AllowedOrigins, actual.Cors.AllowedOrigins)
	require.NotNil(t, actual.RpcgwConfig)
	assert.Empty(t, actual.RpcgwConfig.Services)
}
