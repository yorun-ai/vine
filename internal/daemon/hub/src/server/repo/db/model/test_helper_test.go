package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb/adapter"
	"gorm.io/gorm"
)

func newModelTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func newModelPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_model_028_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	return tx
}

func loadModel028Schema(t *testing.T, db *gorm.DB, dialect string, table string) {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join("testdata", dialect, table+"_0280.sql"))
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(schema)).Error)
}
