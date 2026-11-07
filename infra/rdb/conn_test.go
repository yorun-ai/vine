package rdb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolRetainsConnectionsAfterPeakUsage(t *testing.T) {
	for _, maxOpenConn := range []int{0, 4} {
		t.Run(fmt.Sprintf("maxOpenConn=%d", maxOpenConn), func(t *testing.T) {
			connURL := "sqlite://" + filepath.Join(t.TempDir(), "pool.sqlite")
			db, err := openConnection(Option{ConnURL: connURL, MaxOpenConn: maxOpenConn})
			require.NoError(t, err)
			t.Cleanup(func() { closeConnection(connURL) })
			sqlDB, err := db.DB()
			require.NoError(t, err)
			poolSize := maxOpenConn
			if poolSize <= 0 {
				poolSize = defaultMaxOpenConns
			}
			connections := make([]*sql.Conn, 0, poolSize)
			for range poolSize {
				conn, err := sqlDB.Conn(context.Background())
				require.NoError(t, err)
				t.Cleanup(func() { _ = conn.Close() })
				connections = append(connections, conn)
			}
			require.Equal(t, poolSize, sqlDB.Stats().InUse)
			for _, conn := range connections {
				require.NoError(t, conn.Close())
			}
			stats := sqlDB.Stats()
			assert.Equal(t, poolSize, stats.Idle)
			assert.Equal(t, poolSize, stats.OpenConnections)
			assert.Zero(t, stats.MaxIdleClosed)
		})
	}
}

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
