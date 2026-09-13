package rdb

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSharesGormDBByConnURLAndKeepsFirstPoolSize(t *testing.T) {
	connURL := "sqlite://" + filepath.Join(t.TempDir(), "shared.sqlite")

	db1, err := openConnection(Option{
		ConnURL:     connURL,
		MaxOpenConn: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })

	db2, err := openConnection(Option{
		ConnURL:     connURL,
		MaxOpenConn: 9,
	})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })

	assert.Same(t, db1, db2)

	sqlDB, err := db1.DB()
	require.NoError(t, err)
	assert.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)
}

func TestMemoryDatabaseLastsUntilConnectionLifecycleEnds(t *testing.T) {
	url := "sqlite://file:vine-memory-lifecycle?mode=memory&cache=shared"
	db, err := openConnection(Option{ConnURL: url, MaxOpenConn: 1})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE memory_probe (value TEXT)").Error)
	require.NoError(t, db.Exec("INSERT INTO memory_probe VALUES ('seed')").Error)
	var value string
	require.NoError(t, db.Raw("SELECT value FROM memory_probe").Scan(&value).Error)
	require.Equal(t, "seed", value)
	closeConnection(url)
	reopened, err := openConnection(Option{ConnURL: url, MaxOpenConn: 1})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(url) })
	require.False(t, reopened.Migrator().HasTable("memory_probe"))
}
