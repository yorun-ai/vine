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

	entry := &core.PortalEntry{Name: "web", Protocol: "http", Http: &core.PortalEntryHTTP{Host: "demo.local", HttpEnabled: false, HttpsEnabled: true, HttpsPort: 8443}}
	repo.Save(entry)
	require.NotZero(t, entry.Id)

	got, ok := repo.GetById(entry.Id)
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
	repo.Dao.EnsureSchema()
	return repo
}

func TestPortalEntryRepoListenIPsRoundTrip(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	entry := &core.PortalEntry{Name: "local", ListenIPs: []string{"127.0.0.1", "::1"}, Protocol: "http", Http: &core.PortalEntryHTTP{HttpEnabled: true, HttpsEnabled: false, HttpPort: 8080}}
	repo.Save(entry)
	got, ok := repo.GetById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry.ListenIPs, got.ListenIPs)
	got.ListenIPs[0] = "127.0.0.2"
	again, _ := repo.GetById(entry.Id)
	assert.Equal(t, "127.0.0.1", again.ListenIPs[0])
	entry.ListenIPs = []string{}
	repo.Save(entry)
	got, _ = repo.GetById(entry.Id)
	assert.Empty(t, got.ListenIPs)
}

func TestPortalEntryRepoStoresHTTPHost(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	row := &model.PortalEntry{
		Name: "legacy", Protocol: "http", ListenIPs: "[]", Enabled: true,
		HTTPConfig: `{"host":"demo.local","httpEnabled":true,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":true}`,
	}
	repo.Dao.Save(row)
	entry, ok := repo.GetById(row.Id)
	require.True(t, ok)
	require.Equal(t, "demo.local", entry.Http.Host)
	entry.Http.Host = "new.local"
	repo.Save(entry)
	got, ok := repo.GetById(row.Id)
	require.True(t, ok)
	require.Equal(t, "new.local", got.Http.Host)
}
