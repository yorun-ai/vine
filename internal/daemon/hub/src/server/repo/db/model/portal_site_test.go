package model

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

func TestPortalSiteDaoStoresEnabledSwitch(t *testing.T) {
	dao := newTestPortalSiteDao(t)

	enabled := dao.Save(&PortalSite{Name: "enabled", Type: "WEBGW", WebName: "demo.Web", Enabled: true})
	assert.True(t, enabled.Enabled)

	// Hub stores a disabled entity with the switch off.
	disabled := dao.Save(&PortalSite{Name: "disabled", Type: "WEBGW", WebName: "demo.Web", Enabled: false})
	site, ok := dao.ByName("disabled")
	require.True(t, ok)
	assert.False(t, site.Enabled)
	assert.Equal(t, disabled.Id, site.Id)
}

func TestPortalSiteDaoMigratesFrom028(t *testing.T) {
	testPortalSiteMigrationFrom028(t, newModelTestDB(t, "site-028.sqlite"), "sqlite")
}

func TestPostgresPortalSiteMigrationFrom028(t *testing.T) {
	testPortalSiteMigrationFrom028(t, newModelPostgresTestDB(t), "pgsql")
}

func testPortalSiteMigrationFrom028(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	loadModel028Schema(t, db, dialect, "portal_site")
	require.NoError(t, db.Exec(`INSERT INTO portal_site (name,type,actor_skel_name,actor_via,cors_mode,cors_origins,web_name,enabled)
 VALUES ('kept','RPCGW','demo.Actor','agent','STRICT','["https://example.com"]','',false)`).Error)
	dao := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	dao.EnsureSchema()
	columns, err := tableColumnNames(db, "portal_site")
	require.NoError(t, err)
	require.NotContains(t, columns, "actor_via")
	stored, ok := dao.ByName("kept")
	require.True(t, ok)
	require.NotZero(t, stored.Id)
	require.Equal(t, "RPCGW", stored.Type)
	require.Equal(t, "demo.Actor", stored.ActorSkelName)
	require.Equal(t, "STRICT", stored.CorsMode)
	require.Equal(t, `["https://example.com"]`, stored.CorsOrigins)
	require.False(t, stored.Enabled)
	require.False(t, stored.InspectEnabled)
	require.False(t, stored.McpEnabled)
	require.Equal(t, "null", stored.WebTimeout)
	dao.EnsureSchema()
	again, ok := dao.ById(stored.Id)
	require.True(t, ok)
	require.Equal(t, stored, again)
	for _, switches := range [][2]bool{{true, false}, {false, true}, {false, false}} {
		stored.InspectEnabled, stored.McpEnabled = switches[0], switches[1]
		dao.Save(stored)
		loaded, ok := dao.ById(stored.Id)
		require.True(t, ok)
		require.Equal(t, switches[0], loaded.InspectEnabled)
		require.Equal(t, switches[1], loaded.McpEnabled)
	}
	require.NotZero(t, dao.Save(new(PortalSite{Name: "new", Type: "RPCGW", ActorSkelName: "demo.Actor", Enabled: true})).Id)
}

func newTestPortalSiteDao(t *testing.T) *PortalSiteDao {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "portal-site.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	dao := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
	return dao
}
