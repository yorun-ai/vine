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

// TestPortalSchemaAdmitsAnOlderHub pins the schema a rolled back Hub needs: it
// predates both the enable switch and the entry a rule belongs to, so it inserts
// rows that omit them. The enable switch defaults to published, and the entry is
// nullable, which also keeps two rules of a rolled back Hub out of the unique
// index on the entry path.
func TestPortalSchemaAdmitsAnOlderHub(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "schema.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	(&PortalEntryDao{Dao: rdb.NewDao[*PortalEntry](db)}).InitSchema()
	(&PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}).InitSchema()
	(&PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}).InitSchema()
	(&PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}).InitSchema()

	for _, table := range []string{"portal_entry", "portal_site", "portal_rule", "portal_cert"} {
		assert.NotNil(t, columnDefault(t, db, table, "enabled"), table+" publishes what an older Hub inserts")
	}
	assert.Nil(t, columnDefault(t, db, "portal_rule", "entry_id"))

	// The built-in marker differs by design: its gorm tag drops the zero value
	// on create, so its schema default is the value Hub relies on.
	assert.NotNil(t, columnDefault(t, db, "portal_entry", "built_in"))
}

// columnDefault returns the database default of one column, nil when the schema
// declares none.
func columnDefault(t *testing.T, db *gorm.DB, table string, column string) *string {
	t.Helper()

	var columns []struct {
		Name       string
		DefaultVal *string `gorm:"column:dflt_value"`
	}
	require.NoError(t, db.Raw("PRAGMA table_info("+table+")").Scan(&columns).Error)
	for _, candidate := range columns {
		if candidate.Name == column {
			return candidate.DefaultVal
		}
	}
	require.FailNow(t, "column not found", table+"."+column)
	return nil
}
