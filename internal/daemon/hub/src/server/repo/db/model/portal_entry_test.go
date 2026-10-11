package model

import (
	"fmt"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
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

	entry := dao.Save(&PortalEntry{Name: "web", Protocol: "http", HTTPConfig: `{"host":"demo.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":8443,"autoHTTPS":false}`})
	require.NotZero(t, entry.Id)

	byName, ok := dao.ByName("web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, byName.Id)
	_, ok = dao.ByName("other")
	assert.False(t, ok)

	// A user entry name identifies one entry.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Name: "web", Protocol: "http", HTTPConfig: `{"host":"demo.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":9443,"autoHTTPS":false}`})
	})
}

func TestPortalEntryDaoCreateQueryAndRemove(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	entry := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"demo.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":8443,"autoHTTPS":false}`})
	require.NotZero(t, entry.Id)

	byAccess, ok := dao.ById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry.Id, byAccess.Id)

	updated := dao.Save(&PortalEntry{Id: entry.Id, Protocol: "http", HTTPConfig: `{"host":"demo.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":9443,"autoHTTPS":false}`})
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
	recreated := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"demo.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":9443,"autoHTTPS":false}`})
	assert.NotZero(t, recreated.Id)
}

func TestPortalEntryDaoKeepsOneEntryPerAccess(t *testing.T) {
	dao := newTestPortalEntryDao(t)

	user := dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"","httpEnabled":true,"httpPort":7099,"httpsEnabled":false,"httpsPort":443,"autoHTTPS":false}`})
	require.NotZero(t, user.Id)
	byAccess, ok := dao.ById(user.Id)
	require.True(t, ok)
	assert.Equal(t, user.Id, byAccess.Id)

	// One access never has two entries.
	require.Panics(t, func() {
		dao.Save(&PortalEntry{Protocol: "http", HTTPConfig: `{"host":"","httpEnabled":true,"httpPort":7099,"httpsEnabled":false,"httpsPort":443,"autoHTTPS":false}`})
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
	legacy := strings.ReplaceAll(legacyEntrySchema(schema), "    listen_ips TEXT NOT NULL DEFAULT '[]',  -- Explicit listener IPs; empty keeps legacy wildcard TCP\n", "")
	require.NoError(t, db.Exec(legacy).Error)
	require.NoError(t, db.Exec("INSERT INTO portal_entry (name, scheme, host, port) VALUES ('legacy', 'http', '', 8080)").Error)
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
	entry, ok := dao.ByName("legacy")
	require.True(t, ok)
	assert.Equal(t, "[]", entry.ListenIPs)
	assert.True(t, entry.Enabled)
	assert.Equal(t, 8080, vcode.MustUnmarshalJsonS[core.PortalEntryHTTP](entry.HTTPConfig).HttpPort)
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
	for _, line := range strings.Split(legacyEntrySchema(schema), "\n") {
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
		require.Contains(t, entry.HTTPConfig, `"host":"demo.local"`)
		require.Equal(t, "http", entry.Protocol)
		require.Contains(t, entry.HTTPConfig, `"autoHTTPS":false`)
		if entry.Name == "plain" {
			require.Contains(t, entry.HTTPConfig, `"httpsEnabled":false`)
		} else {
			require.Contains(t, entry.HTTPConfig, `"httpEnabled":false`)
		}
	}
	dao.Save(&PortalEntry{Name: "dual", Protocol: "http", HTTPConfig: `{"host":"other.local","httpEnabled":true,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":true}`, Enabled: true})
	dao.Save(&PortalEntry{Name: "another-dual", Protocol: "http", HTTPConfig: `{"host":"third.local","httpEnabled":true,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":true}`, Enabled: true})
	dao.EnsureSchema()
	require.Len(t, dao.ListOrdered(), 4)
}

// Build both historical schemas explicitly; production table definitions have no old columns.
func legacyEntrySchema(schema string) string {
	schema = strings.Replace(schema, "    enabled BOOLEAN", "    scheme TEXT NOT NULL DEFAULT '',\n    host TEXT NOT NULL DEFAULT '',\n    port INTEGER NOT NULL DEFAULT 0,\n    enabled BOOLEAN", 1)
	return schema + "CREATE UNIQUE INDEX uk_portal_entry_access ON portal_entry(scheme,host,port) WHERE deleted_at IS NULL AND scheme <> '';"
}

func TestPortalEntryMigrationMovesHostAndDropsLegacyColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "host.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Exec(legacyEntrySchema(createPortalEntrySQLiteSQL)).Error)
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX uk_portal_entry_protocol_access ON portal_entry(protocol,host,http_config) WHERE deleted_at IS NULL AND http_config <> ''").Error)
	configs := []string{
		`{"httpEnabled":true,"httpPort":8080,"httpsEnabled":true,"httpsPort":8443,"autoHTTPS":true}`,
		`{"host":"","httpEnabled":true,"httpPort":80,"httpsEnabled":false,"httpsPort":443,"autoHTTPS":false}`,
		`{"host":"nested.local","httpEnabled":false,"httpPort":80,"httpsEnabled":true,"httpsPort":443,"autoHTTPS":false}`,
	}
	for i, config := range configs {
		require.NoError(t, db.Exec("INSERT INTO portal_entry (id,name,protocol,http_config,host,listen_ips,enabled) VALUES (?,?,?,?,?,?,?)", i+1, fmt.Sprint(i), "http", config, "old.local", `["127.0.0.1"]`, false).Error)
	}
	require.NoError(t, db.Exec("UPDATE portal_entry SET deleted_at = CURRENT_TIMESTAMP WHERE id = 3").Error)
	dao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
	columns, err := tableColumnNames(db, "portal_entry")
	require.NoError(t, err)
	for _, name := range []string{"scheme", "host", "port"} {
		require.False(t, columns[name])
	}
	rows := []*PortalEntry{}
	require.NoError(t, db.Unscoped().Order("id").Find(&rows).Error)
	require.Len(t, rows, 3)
	for i, host := range []string{"old.local", "", "nested.local"} {
		require.Equal(t, i+1, rows[i].Id)
		require.Equal(t, host, vcode.MustUnmarshalJsonS[core.PortalEntryHTTP](rows[i].HTTPConfig).Host)
		require.False(t, rows[i].Enabled)
		require.Equal(t, `["127.0.0.1"]`, rows[i].ListenIPs)
	}
	require.Len(t, dao.ListOrdered(), 2)
}
