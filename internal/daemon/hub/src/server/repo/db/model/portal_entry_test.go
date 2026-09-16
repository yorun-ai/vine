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

func TestPortalEntryDaoStoresAndFindsName(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Name: "web", Scheme: "https", Host: "demo.local", Port: 8443})
	require.NotZero(t, entry.Id)

	byName, ok := dao.ByName("web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, byName.Id)
	_, ok = dao.ByName("other")
	assert.False(t, ok)

	// A user entry name identifies one entry.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Name: "web", Scheme: "https", Host: "demo.local", Port: 9443})
	})
}

// _legacyPortalEntrySchema is the entry table before Hub named entries.
const _legacyPortalEntrySchema = `
CREATE TABLE portal_entry (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    scheme TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    built_in BOOLEAN NOT NULL DEFAULT FALSE
);
`

func TestPortalEntryInitSchemaNamesEntriesFromAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-entry.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec(_legacyPortalEntrySchema).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_entry (scheme, host, port, built_in) VALUES ('http', '', 80, FALSE)`).Error)

	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.InitSchema()

	// Hub keeps the label it showed for those entries before they had a name.
	web, ok := dao.ByAccess("http", "", 80)
	require.True(t, ok)
	assert.Equal(t, "http:80", web.Name)
}

func TestPortalEntryDaoCreateQueryAndRemove(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Scheme: "https", Host: "demo.local", Port: 8443})
	require.NotZero(t, entry.Id)

	byAccess, ok := dao.ByAccess("https", "demo.local", 8443)
	require.True(t, ok)
	assert.Equal(t, entry.Id, byAccess.Id)

	updated := dao.Save(&PortalEntry{Id: entry.Id, Scheme: "https", Host: "demo.local", Port: 9443})
	assert.Equal(t, entry.Id, updated.Id)
	_, ok = dao.ByAccess("https", "demo.local", 8443)
	assert.False(t, ok)
	_, ok = dao.ByAccess("https", "demo.local", 9443)
	assert.True(t, ok)

	_, ok = dao.DeleteById(entry.Id)
	require.True(t, ok)
	_, ok = dao.ById(entry.Id)
	assert.False(t, ok)

	// Hub recreates an entry when rules return to an access, so the deleted row
	// must not keep the access index occupied.
	recreated := dao.Save(&PortalEntry{Scheme: "https", Host: "demo.local", Port: 9443})
	assert.NotZero(t, recreated.Id)
}

func TestPortalEntryDaoKeepsOneEntryPerAccess(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	user := dao.Save(&PortalEntry{Scheme: "http", Host: "", Port: 7099})
	require.NotZero(t, user.Id)
	byAccess, ok := dao.ByAccess("http", "", 7099)
	require.True(t, ok)
	assert.Equal(t, user.Id, byAccess.Id)

	// One access never has two entries.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Scheme: "http", Host: "", Port: 7099})
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
	dao.InitSchema()
	dao.InitSchema()
	return dao
}
