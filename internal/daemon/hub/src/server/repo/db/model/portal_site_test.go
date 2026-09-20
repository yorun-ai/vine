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
