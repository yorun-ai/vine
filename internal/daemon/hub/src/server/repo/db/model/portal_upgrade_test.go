package model

import (
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/infra/rdb/adapter"
	"gorm.io/gorm"
)

// _legacyPostgresSchema is the 0.19.0 PostgreSQL schema of the tables entries
// rewrite: rules store the access, no entry table exists, and no table carries
// the enable switch.
const _legacyPostgresSchema = `
CREATE TABLE IF NOT EXISTS portal_site (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    actor_skel_name TEXT NOT NULL,
    actor_via TEXT NOT NULL,
    cors_mode TEXT NOT NULL DEFAULT 'SAME_DOMAIN',
    cors_origins TEXT NOT NULL DEFAULT '[]',
    web_name TEXT NOT NULL,
    built_in BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_site_name
    ON portal_site(name);

CREATE TABLE IF NOT EXISTS portal_cert (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    name TEXT NOT NULL,
    issuer TEXT NOT NULL,
    domains TEXT NOT NULL,
    public_key_base64 TEXT NOT NULL,
    private_key_base64 TEXT NOT NULL,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_cert_name
    ON portal_cert(name);

CREATE TABLE IF NOT EXISTS portal_rule (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    name TEXT NOT NULL,
    match_scheme TEXT NOT NULL,
    match_host TEXT NOT NULL,
    match_port INTEGER NOT NULL,
    match_path_prefix TEXT NOT NULL,
    route_type TEXT NOT NULL,
    route_site_name TEXT NOT NULL,
    route_redirection_pattern TEXT NOT NULL,
    route_path_prefix TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_match
    ON portal_rule(match_scheme, match_host, match_port, match_path_prefix);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_name
    ON portal_rule(name);
`

// Opt in with a PostgreSQL DSN. The upgrade runs in a private schema inside a
// rolled back transaction, so the test leaves the database as it found it.
func TestPostgresUpgradeFrom019(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_upgrade_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)

	require.NoError(t, tx.Exec(_legacyPostgresSchema).Error)
	require.NoError(t, tx.Exec(`INSERT INTO portal_site (name, type, actor_skel_name, actor_via, web_name) VALUES ('web', 'WEBGW', 'demo.Actor', 'client', 'demo.Web')`).Error)
	require.NoError(t, tx.Exec(`INSERT INTO portal_cert (name, issuer, domains, public_key_base64, private_key_base64) VALUES ('leaf', 'demo', '["demo.local"]', 'cHVibGlj', 'cHJpdmF0ZQ==')`).Error)
	rule := `INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern, built_in) VALUES (?, ?, ?, ?, ?, 'SITE', 'demo.Web', '', FALSE)`
	require.NoError(t, tx.Exec(rule, "demo.web", "https", "", 443, "/").Error)
	require.NoError(t, tx.Exec(rule, "demo.default", "https", "", 0, "/").Error)
	require.NoError(t, tx.Exec(rule, "demo.app", "http", "demo.local", 8080, "/app").Error)

	(&PortalSiteDao{Dao: rdb.NewDao[*PortalSite](tx)}).InitSchema()
	(&PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](tx)}).InitSchema()
	(&PortalRuleDao{Dao: rdb.NewDao[*PortalRule](tx)}).InitSchema()
	(&PortalCertDao{Dao: rdb.NewDao[*PortalCert](tx)}).InitSchema()

	// Rules with the same access share one entry, and the rule that named the
	// default port keeps its path while the one that left it unset moves.
	type entryRow struct {
		Id      int
		Name    string
		Enabled bool
	}
	entries := []entryRow{}
	require.NoError(t, tx.Raw("SELECT id, name, enabled FROM portal_entry ORDER BY name").Scan(&entries).Error)
	require.Len(t, entries, 2)
	assert.Equal(t, "http:demo.local:8080", entries[0].Name)
	assert.Equal(t, "https:443", entries[1].Name)
	assert.True(t, entries[0].Enabled)
	assert.True(t, entries[1].Enabled)
	entryId := map[string]int{entries[0].Name: entries[0].Id, entries[1].Name: entries[1].Id}

	type ruleRow struct {
		Name    string
		Path    string `gorm:"column:match_path_prefix"`
		EntryId int    `gorm:"column:entry_id"`
		Enabled bool
	}
	rules := []ruleRow{}
	require.NoError(t, tx.Raw("SELECT name, match_path_prefix, entry_id, enabled FROM portal_rule ORDER BY name").Scan(&rules).Error)
	assert.Equal(t, []ruleRow{
		{Name: "demo.app", Path: "/app", EntryId: entryId["http:demo.local:8080"], Enabled: true},
		{Name: "demo.default", Path: "/migrated", EntryId: entryId["https:443"], Enabled: true},
		{Name: "demo.web", Path: "/", EntryId: entryId["https:443"], Enabled: true},
	}, rules)

	// The access columns stay for one release, and they keep the access the rule
	// matched before the entry took over.
	type accessRow struct {
		Name   string
		Scheme string `gorm:"column:match_scheme"`
		Host   string `gorm:"column:match_host"`
		Port   int    `gorm:"column:match_port"`
	}
	access := []accessRow{}
	require.NoError(t, tx.Raw("SELECT name, match_scheme, match_host, match_port FROM portal_rule WHERE name = 'demo.default'").Scan(&access).Error)
	assert.Equal(t, []accessRow{{Name: "demo.default", Scheme: "https", Host: "", Port: 0}}, access)
	assert.False(t, tx.Migrator().HasIndex("portal_rule", "uk_portal_rule_match"))

	// The configuration Hub already stored stays published.
	type switchRow struct {
		Name    string
		Enabled bool
	}
	sites := []switchRow{}
	require.NoError(t, tx.Raw("SELECT name, enabled FROM portal_site").Scan(&sites).Error)
	assert.Equal(t, []switchRow{{Name: "web", Enabled: true}}, sites)
	certs := []switchRow{}
	require.NoError(t, tx.Raw("SELECT name, enabled FROM portal_cert").Scan(&certs).Error)
	assert.Equal(t, []switchRow{{Name: "leaf", Enabled: true}}, certs)
}
