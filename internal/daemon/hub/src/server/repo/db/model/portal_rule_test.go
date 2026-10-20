package model

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

func TestPortalRuleEnsureSchemaOnCurrentSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "current.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec(createPortalEntrySQLiteSQL).Error)
	require.NoError(t, db.Exec(createPortalRuleSQLiteSQL).Error)
	dao := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	dao.EnsureSchema()
	dao.EnsureSchema()
}
