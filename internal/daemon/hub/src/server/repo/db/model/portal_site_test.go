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

// _legacyPortalSiteSchema is the site table before Hub had an enable switch.
const _legacyPortalSiteSchema = `
CREATE TABLE portal_site (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    actor_skel_name TEXT NOT NULL,
    actor_via TEXT NOT NULL,
    cors_mode TEXT NOT NULL DEFAULT 'SAME_DOMAIN',
    cors_origins TEXT NOT NULL DEFAULT '[]',
    web_name TEXT NOT NULL,
    built_in BOOLEAN NOT NULL DEFAULT FALSE
);
`

func TestPortalSiteEnsureSchemaAddsEnabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-site.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec(_legacyPortalSiteSchema).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_site (name, type, actor_skel_name, actor_via, web_name) VALUES ('web', 'WEBGW', 'demo.Actor', 'client', 'demo.Web')`).Error)

	dao := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	dao.EnsureSchema()

	// A site Hub already stores stays enabled.
	site, ok := dao.ByName("web")
	require.True(t, ok)
	assert.True(t, site.Enabled)
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
