package repo

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"gorm.io/gorm"
)

func TestPortalEntryRepoStoresAccessAndName(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)

	entry := &core.PortalEntry{Name: "web", Scheme: "https", Host: "demo.local", Port: 8443}
	repo.Save(entry)
	require.NotZero(t, entry.Id)

	got, ok := repo.GetBySchemeHostPort("https", "demo.local", 8443)
	require.True(t, ok)
	assert.Equal(t, entry.Id, got.Id)
	// An entry keeps the name it was stored with.
	assert.Equal(t, "web", got.Name)
	byName, ok := repo.GetByName("web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, byName.Id)
	_, ok = repo.GetByName("other")
	assert.False(t, ok)

	_, ok = repo.GetById(entry.Id)
	assert.True(t, ok)
}

func TestPortalEntryRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &PortalEntryRepo{Access: access}

	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() {
		repo.Save(new(core.PortalEntry))
	})
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() {
		repo.Remove(1)
	})
}

func newTestPortalEntryRepoDB(t *testing.T) *PortalEntryRepo {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "portal-entry.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)
	repo := &PortalEntryRepo{
		Dao:    &model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](db)},
		Syncer: testSyncer(watchServer),
		Access: new(configaccess.Access),
	}
	repo.Dao.InitSchema()
	return repo
}
