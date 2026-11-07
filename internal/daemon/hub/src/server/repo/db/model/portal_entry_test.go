package model

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

func TestPortalEntryDaoStoresAndFindsName(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Name: "web", Protocol: "http", HTTPConfig: `{"host":"demo.local","http":"DISABLED","httpPort":80,"https":"ENABLED","httpsPort":8443,"autohttps":"DISABLED"}`})
	require.NotZero(t, entry.Id)

	byName, ok := dao.ByName("web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, byName.Id)
	_, ok = dao.ByName("other")
	assert.False(t, ok)

	// A user entry name identifies one entry.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Name: "web", Protocol: "http", HTTPConfig: `{"host":"demo.local","http":"DISABLED","httpPort":80,"https":"ENABLED","httpsPort":9443,"autohttps":"DISABLED"}`})
	})
}

func TestPortalEntryDaoCreateQueryAndRemove(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"demo.local","http":"DISABLED","httpPort":80,"https":"ENABLED","httpsPort":8443,"autohttps":"DISABLED"}`})
	require.NotZero(t, entry.Id)

	byAccess, ok := dao.ById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry.Id, byAccess.Id)

	updated := dao.Save(&PortalEntry{Id: entry.Id, Protocol: "http", HTTPConfig: `{"host":"demo.local","http":"DISABLED","httpPort":80,"https":"ENABLED","httpsPort":9443,"autohttps":"DISABLED"}`})
	assert.Equal(t, entry.Id, updated.Id)
	got, ok := dao.ById(entry.Id)
	require.True(t, ok)
	assert.Contains(t, got.HTTPConfig, `"httpsPort":9443`)

	_, ok = dao.DeleteById(entry.Id)
	require.True(t, ok)
	_, ok = dao.ById(entry.Id)
	assert.False(t, ok)

	// Hub recreates an entry when rules return to an access, so the deleted row
	// must not keep the access index occupied.
	recreated := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"demo.local","http":"DISABLED","httpPort":80,"https":"ENABLED","httpsPort":9443,"autohttps":"DISABLED"}`})
	assert.NotZero(t, recreated.Id)
}

func TestPortalEntryDaoKeepsOneEntryPerAccess(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	user := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"","http":"ENABLED","httpPort":7099,"https":"DISABLED","httpsPort":443,"autohttps":"DISABLED"}`})
	require.NotZero(t, user.Id)
	byAccess, ok := dao.ById(user.Id)
	require.True(t, ok)
	assert.Equal(t, user.Id, byAccess.Id)

	// One access never has two entries.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"","http":"ENABLED","httpPort":7099,"https":"DISABLED","httpsPort":443,"autohttps":"DISABLED"}`})
	})
}

func newTestPortalEntryDao(t *testing.T) *PortalEntryDao {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "portal-entry.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
	return dao
}

// Fixtures are the unmodified portal_entry schemas from the v0.28.0 tag.
func TestPortalEntryMigrationFrom028(t *testing.T) {
	db := newModelTestDB(t, "entry-028.sqlite")
	testPortalEntryMigrationFrom028(t, db, "sqlite")
}

func TestPostgresPortalEntryMigrationFrom028(t *testing.T) {
	testPortalEntryMigrationFrom028(t, newModelPostgresTestDB(t), "pgsql")
}

func testPortalEntryMigrationFrom028(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	loadModel028Schema(t, db, dialect, "portal_entry")
	require.NoError(t, db.Exec(`INSERT INTO portal_entry (name,scheme,host,port,enabled,created_at,updated_at) VALUES
 ('plain','http','demo.local',8080,true,'2026-01-01 00:00:00','2026-01-02 00:00:00'),
 ('secure','https','DEMO.LOCAL.',8443,false,'2026-01-01 00:00:00','2026-01-02 00:00:00'),
 ('archived','http','',80,false,'2026-01-01 00:00:00','2026-01-02 00:00:00')`).Error)
	require.NoError(t, db.Exec("UPDATE portal_entry SET deleted_at = '2026-01-03 00:00:00' WHERE name = 'archived'").Error)
	// Only read fields that exist before and after the migration, including deleted rows.
	before := []*PortalEntry{}
	require.NoError(t, db.Unscoped().Select("id, name, enabled, created_at, updated_at, deleted_at").Order("id").Find(&before).Error)
	require.Len(t, before, 3)
	ruleDao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	ruleDao.EnsureSchema()
	rule := ruleDao.Save(new(PortalRule{Name: "kept", EntryId: before[0].Id, MatchPathPrefix: "/", RouteType: "SITE", RouteSiteName: "application", Enabled: true}))
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.EnsureSchema()
	after := []*PortalEntry{}
	require.NoError(t, db.Unscoped().Order("id").Find(&after).Error)
	require.Len(t, after, 3)
	for i, entry := range after {
		require.Equal(t, before[i].Id, entry.Id)
		require.Equal(t, before[i].Name, entry.Name)
		require.Equal(t, before[i].Enabled, entry.Enabled)
		require.Equal(t, before[i].CreatedAt, entry.CreatedAt)
		require.Equal(t, before[i].UpdatedAt, entry.UpdatedAt)
		require.Equal(t, before[i].DeletedAt, entry.DeletedAt)
		require.Equal(t, "[]", entry.ListenIPs)
		require.Equal(t, "http", entry.Protocol)
		cfg := vcode.MustUnmarshalJsonS[core.PortalEntryHTTP](entry.HTTPConfig)
		require.Equal(t, core.PortalSwitchModeDisabled, cfg.AutoHTTPS)
		switch entry.Name {
		case "plain":
			require.Equal(t, core.PortalEntryHTTP{Host: "demo.local", Http: core.PortalSwitchModeEnabled, HttpPort: 8080, Https: core.PortalSwitchModeDisabled, HttpsPort: 443, AutoHTTPS: core.PortalSwitchModeDisabled}, cfg)
		case "secure":
			require.Equal(t, core.PortalEntryHTTP{Host: "demo.local", Http: core.PortalSwitchModeDisabled, HttpPort: 80, Https: core.PortalSwitchModeEnabled, HttpsPort: 8443, AutoHTTPS: core.PortalSwitchModeDisabled}, cfg)
		case "archived":
			require.Equal(t, "", cfg.Host)
			require.Equal(t, 80, cfg.HttpPort)
		}
	}
	columns, err := tableColumnNames(db, "portal_entry")
	require.NoError(t, err)
	for _, field := range []string{"scheme", "host", "port"} {
		require.NotContains(t, columns, field)
	}
	for _, field := range []string{"protocol", "http_config", "listen_ips"} {
		require.Contains(t, columns, field)
	}
	loadedRule, ok := ruleDao.ById(rule.Id)
	require.True(t, ok)
	require.Equal(t, rule.EntryId, loadedRule.EntryId)
	require.Len(t, dao.ListOrdered(), 2)
	// A second startup preserves the complete result, including soft-deleted rows.
	dao.EnsureSchema()
	again := []*PortalEntry{}
	require.NoError(t, db.Unscoped().Order("id").Find(&again).Error)
	require.Equal(t, after, again)
	// The retired single-transport index must no longer block new-format entries.
	dao.Save(new(PortalEntry{Name: "dual", Protocol: "http", HTTPConfig: `{"host":"other.local","http":"AUTO","https":"AUTO","autohttps":"AUTO"}`, ListenIPs: "[]", Enabled: true}))
	dao.EnsureSchema()
	require.Len(t, dao.ListOrdered(), 3)
}

func TestPortalEntryMigrationRollsBackInvalidLegacyData(t *testing.T) {
	for _, scheme := range []string{"tcp", "https"} {
		t.Run(scheme, func(t *testing.T) {
			db := newModelTestDB(t, "rollback.sqlite")
			loadModel028Schema(t, db, "sqlite", "portal_entry")
			require.NoError(t, db.Exec("INSERT INTO portal_entry (name,scheme,host,port) VALUES ('invalid',?,'',80)", scheme).Error)
			before, err := tableColumnNames(db, "portal_entry")
			require.NoError(t, err)
			dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
			require.Panics(t, dao.EnsureSchema)
			after, err := tableColumnNames(db, "portal_entry")
			require.NoError(t, err)
			require.Equal(t, before, after)
			var stored string
			require.NoError(t, db.Raw("SELECT scheme FROM portal_entry WHERE name = 'invalid'").Scan(&stored).Error)
			require.Equal(t, scheme, stored)
		})
	}
}
