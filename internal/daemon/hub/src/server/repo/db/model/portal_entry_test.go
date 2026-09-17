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

func TestPortalEntryDaoCreateQueryAndRemove(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Scheme: "https", Host: "demo.local", Port: 8443})
	require.NotZero(t, entry.Id)

	byAccess, ok := dao.BySchemeHostPort("https", "demo.local", 8443)
	require.True(t, ok)
	assert.Equal(t, entry.Id, byAccess.Id)

	updated := dao.Save(&PortalEntry{Id: entry.Id, Scheme: "https", Host: "demo.local", Port: 9443})
	assert.Equal(t, entry.Id, updated.Id)
	_, ok = dao.BySchemeHostPort("https", "demo.local", 8443)
	assert.False(t, ok)
	_, ok = dao.BySchemeHostPort("https", "demo.local", 9443)
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
	byAccess, ok := dao.BySchemeHostPort("http", "", 7099)
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
