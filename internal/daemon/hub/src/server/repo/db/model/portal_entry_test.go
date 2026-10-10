package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/infra/rdb/adapter"
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
	dao.EnsureSchema()
	dao.EnsureSchema()
	return dao
}

func TestPortalEntryDaoMigratesListenIPs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	testPortalEntryListenIPsMigration(t, db, createPortalEntrySQLiteSQL)
}

func TestPostgresPortalEntryListenIPsMigration(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_entry_ips_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	testPortalEntryListenIPsMigration(t, tx, createPortalEntryPgSQL)
}

func testPortalEntryListenIPsMigration(t *testing.T, db *gorm.DB, schema string) {
	t.Helper()
	legacy := strings.ReplaceAll(schema, "    listen_ips TEXT NOT NULL DEFAULT '[]',  -- Explicit listener IPs; empty keeps legacy wildcard TCP\n", "")
	require.NoError(t, db.Exec(legacy).Error)
	require.NoError(t, db.Exec("INSERT INTO portal_entry (name, scheme, host, port) VALUES ('legacy', 'http', '', 8080)").Error)
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
	entry, ok := dao.ByName("legacy")
	require.True(t, ok)
	assert.Equal(t, "[]", entry.ListenIPs)
	assert.True(t, entry.Enabled)
	assert.Equal(t, 8080, entry.Port)
	entry.ListenIPs = `["127.0.0.1","::1"]`
	dao.Save(entry)
	entry, _ = dao.ByName("legacy")
	assert.Equal(t, `["127.0.0.1","::1"]`, entry.ListenIPs)
}

func TestPortalEntryProtocolMigrationPreservesSeparateLegacyEntries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "protocol.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	testPortalEntryProtocolMigration(t, db, createPortalEntrySQLiteSQL)
}
func TestPostgresPortalEntryProtocolMigration(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_entry_protocol_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	testPortalEntryProtocolMigration(t, tx, createPortalEntryPgSQL)
}
func testPortalEntryProtocolMigration(t *testing.T, db *gorm.DB, schema string) {
	t.Helper()
	legacyLines := []string{}
	for _, line := range strings.Split(schema, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "protocol TEXT") || strings.HasPrefix(strings.TrimSpace(line), "http_config TEXT") {
			continue
		}
		legacyLines = append(legacyLines, line)
	}
	legacy := strings.ReplaceAll(strings.Join(legacyLines, "\n"), " AND scheme <> ''", "")
	require.NoError(t, db.Exec(legacy).Error)
	require.NoError(t, db.Exec("INSERT INTO portal_entry (name,scheme,host,port,listen_ips,enabled) VALUES ('plain','http','demo.local',8080,'[\"127.0.0.1\"]',true),('secure','https','demo.local',8443,'[\"::1\"]',false)").Error)
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	before := dao.ListOrdered()
	require.Len(t, before, 2)
	dao.EnsureSchema()
	dao.EnsureSchema()
	after := dao.ListOrdered()
	require.Len(t, after, 2)
	for i, entry := range after {
		require.Equal(t, before[i].Id, entry.Id)
		require.Equal(t, before[i].Name, entry.Name)
		require.Equal(t, before[i].Enabled, entry.Enabled)
		require.Equal(t, before[i].ListenIPs, entry.ListenIPs)
		require.Equal(t, before[i].Scheme, entry.Scheme)
		require.Equal(t, before[i].Port, entry.Port)
		require.Equal(t, "http", entry.Protocol)
		require.Contains(t, entry.HTTPConfig, `"autoHTTPS":false`)
		if entry.Scheme == "http" {
			require.Contains(t, entry.HTTPConfig, `"httpsEnabled":false`)
		} else {
			require.Contains(t, entry.HTTPConfig, `"httpEnabled":false`)
		}
	}
	dao.Save(&PortalEntry{Name: "dual", Protocol: "http", HTTPConfig: `{"httpEnabled":true,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":true}`, Host: "other.local", Enabled: true})
	dao.Save(&PortalEntry{Name: "another-dual", Protocol: "http", HTTPConfig: `{"httpEnabled":true,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":true}`, Host: "third.local", Enabled: true})
	dao.EnsureSchema()
	require.Len(t, dao.ListOrdered(), 4)
}
