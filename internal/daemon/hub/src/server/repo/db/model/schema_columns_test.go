package model

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDropColumnsRollsBackWhenColumnIsReferenced(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rollback.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	require.NoError(t, db.Exec("CREATE TABLE example (id INTEGER PRIMARY KEY, retired TEXT, indexed TEXT)").Error)
	require.NoError(t, db.Exec("CREATE INDEX keep_index ON example(indexed)").Error)
	require.Panics(t, func() { dropColumns(db, "example", "retired", "indexed") })
	columns, err := tableColumnNames(db, "example")
	require.NoError(t, err)
	require.Contains(t, columns, "retired")
	require.Contains(t, columns, "indexed")
	require.True(t, db.Migrator().HasIndex("example", "keep_index"))
}
