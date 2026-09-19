package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
)

// TestPortalRuleEnsureSchemaRemovesLegacyBuiltInEntities pins the cleanup of the
// Dashboard entities an earlier release provisioned for Portal: Hub deletes the
// stored built-in rows and keeps everything an operator stores.
func TestPortalRuleEnsureSchemaRemovesLegacyBuiltInEntities(t *testing.T) {
	db := newTestLegacyPortalRuleDB(t,
		`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern)
		 VALUES ('demo.web', 'http', '', 8080, '/', 'SITE', 'demo.Web', '')`,
	)
	entryDao := &PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}
	entryDao.EnsureSchema()
	require.NoError(t, db.Exec(`INSERT INTO portal_rule (name, match_scheme, match_host, match_port, match_path_prefix, route_type, route_site_name, route_redirection_pattern, built_in)
		VALUES ('vine.hub.admin-api', 'http', '', 7099, '/api', 'SITE', 'vine.hub.admin.AdminActor-client-rpc', '', TRUE)`).Error)
	require.NoError(t, db.Exec(_legacyPortalSiteSchema).Error)
	require.NoError(t, db.Exec(`INSERT INTO portal_site (name, type, actor_skel_name, actor_via, web_name, built_in)
		VALUES ('vine.hub.admin.DashboardWeb-web', 'WEBGW', 'vine.hub.admin.DashboardActor', 'client', 'vine.hub.admin.DashboardWeb', TRUE)`).Error)

	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	dao.EnsureSchema()
	siteDao := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	siteDao.EnsureSchema()

	rules := dao.ListOrdered()
	require.Len(t, rules, 1)
	assert.Equal(t, "demo.web", rules[0].Name)

	entries := entryDao.ListOrdered()
	require.Len(t, entries, 1)
	assert.Equal(t, "http:8080", entries[0].Name)
	assert.Equal(t, 8080, entries[0].Port)

	assert.Empty(t, siteDao.ListOrdered())
}
