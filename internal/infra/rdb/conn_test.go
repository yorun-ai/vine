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
