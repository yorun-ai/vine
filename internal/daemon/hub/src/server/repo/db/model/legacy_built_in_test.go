package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
)

// TestPortalRuleInitSchemaRemovesLegacyBuiltInEntities pins the cleanup of the
// Dashboard entities an earlier release provisioned for Portal: Hub deletes the
// stored built-in rows and the entry the access migration left at the access
// those rules served, and it keeps everything an operator stores.
func TestPortalRuleInitSchemaRemovesLegacyBuiltInEntities(t *testing.T) {
	db := newTestLegacyPortalRuleDB(t,
		`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
		 VALUES ('demo.web', 'http', '', 8080, '/', 'SITE', 'demo.Web', '')`,
	)
	entryDao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	entryDao.InitSchema()
	require.NoError(t, db.Exec(`INSERT INTO portal_entry (name, scheme, host, port, built_in, enabled, created_at, updated_at)
		VALUES ('vine.hub.dashboard', 'http', '', 7099, TRUE, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_entry (name, scheme, host, port, built_in, enabled, created_at, updated_at)
		VALUES ('http:7099', 'http', '', 7099, FALSE, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern, built_in)
		VALUES ('vine.hub.admin-api', 'http', '', 7099, '/api', 'SITE', 'vine.hub.admin.AdminActor-client-rpc', '', TRUE)`).Error)

	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	dao.InitSchema()

	rules := dao.ListOrdered()
	require.Len(t, rules, 1)
	assert.Equal(t, "demo.web", rules[0].Name)

	entries := entryDao.ListOrdered()
	require.Len(t, entries, 1)
	assert.Equal(t, "http:8080", entries[0].Name)
	assert.Equal(t, 8080, entries[0].Port)
}
