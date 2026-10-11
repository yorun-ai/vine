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

func TestPortalSiteDaoMigratesInspectDisabledAndPersistsToggles(t *testing.T) {
	dao := newTestPortalSiteDao(t)
	entry := dao.Save(new(PortalSite{Name: "existing-api", Type: "RPCGW", ActorSkelName: "demo.Actor", Enabled: true}))
	require.NoError(t, dao.GormDB().Exec("ALTER TABLE portal_site DROP COLUMN inspect_enabled").Error)
	dao.EnsureSchema()
	dao.EnsureSchema()
	stored, ok := dao.ById(entry.Id)
	require.True(t, ok)
	require.False(t, stored.InspectEnabled)
	for _, enabled := range []bool{true, false} {
		stored.InspectEnabled = enabled
		dao.Save(stored)
		stored, ok = dao.ById(entry.Id)
		require.True(t, ok)
		require.Equal(t, enabled, stored.InspectEnabled)
	}
}

func TestPortalSiteDaoMigratesMcpDisabledAndPersistsToggles(t *testing.T) {
	dao := newTestPortalSiteDao(t)
	entry := dao.Save(new(PortalSite{Name: "existing-api", Type: "RPCGW", ActorSkelName: "demo.Actor", Enabled: true}))
	require.NoError(t, dao.GormDB().Exec("ALTER TABLE portal_site DROP COLUMN mcp_enabled").Error)
	dao.EnsureSchema()
	dao.EnsureSchema()
	stored, ok := dao.ById(entry.Id)
	require.True(t, ok)
	require.False(t, stored.McpEnabled)
	for _, enabled := range []bool{true, false} {
		stored.McpEnabled = enabled
		dao.Save(stored)
		stored, ok = dao.ById(entry.Id)
		require.True(t, ok)
		require.Equal(t, enabled, stored.McpEnabled)
	}
}

func TestPortalSiteDaoRemovesViaWithoutChangingSites(t *testing.T) {
	dao := newTestPortalSiteDao(t)
	site := dao.Save(new(PortalSite{Name: "kept", Type: "RPCGW", ActorSkelName: "demo.Actor", Enabled: true}))
	require.NoError(t, dao.GormDB().Exec("ALTER TABLE portal_site ADD COLUMN actor_via TEXT NOT NULL DEFAULT 'agent'").Error)
	dao.EnsureSchema()
	dao.EnsureSchema()
	columns, err := tableColumnNames(dao.GormDB(), "portal_site")
	require.NoError(t, err)
	require.NotContains(t, columns, "actor_via")
	stored, ok := dao.ById(site.Id)
	require.True(t, ok)
	require.Equal(t, site, stored)
	stored.Name = "updated"
	dao.Save(stored)
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
