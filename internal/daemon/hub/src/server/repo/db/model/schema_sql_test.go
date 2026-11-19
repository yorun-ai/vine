package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

func TestMigrateConfigNamesAfterSoftDelete(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		testMigrateConfigNames(t, newModelTestDB(t, "names.db"))
	})
	t.Run("postgres", func(t *testing.T) {
		testMigrateConfigNames(t, newModelPostgresTestDB(t))
	})
}

func testMigrateConfigNames(t *testing.T, db *gorm.DB) {
	configs := AppConfigDao{Dao: rdb.NewDao[*AppConfig](db)}
	rules := PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	sites := PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	certs := PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}
	for _, tc := range []struct {
		table  string
		ensure func()
		row    func() any
	}{
		{"app_config", configs.EnsureSchema, func() any { return new(AppConfig{Name: "reuse", Value: "{}", Version: 1}) }},
		{"portal_rule", rules.EnsureSchema, func() any { return new(PortalRule{Name: "reuse"}) }},
		{"portal_site", sites.EnsureSchema, func() any { return new(PortalSite{Name: "reuse"}) }},
		{"portal_cert", certs.EnsureSchema, func() any { return new(PortalCert{Name: "reuse"}) }},
	} {
		t.Run(tc.table, func(t *testing.T) {
			tc.ensure()
			// Reproduce a database created by a Hub that indexed deleted names.
			require.NoError(t, db.Exec("DROP INDEX IF EXISTS uk_"+tc.table+"_live_name").Error)
			require.NoError(t, db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_"+tc.table+"_name ON "+tc.table+"(name)").Error)
			old := tc.row()
			require.NoError(t, db.Create(old).Error)
			require.NoError(t, db.Delete(old).Error)

			tc.ensure()
			require.NoError(t, db.Create(tc.row()).Error)
			tc.ensure()
			var deleted int64
			require.NoError(t, db.Table(tc.table).Where("name = ? AND deleted_at IS NOT NULL", "reuse").Count(&deleted).Error)
			require.EqualValues(t, 1, deleted)
			err := db.Transaction(func(tx *gorm.DB) error { return tx.Create(tc.row()).Error })
			require.Error(t, err)
			require.Contains(t, strings.ToLower(err.Error()), "unique")
		})
	}
}
