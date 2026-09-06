package rdb

import (
	"database/sql/driver"
	"go.yorun.ai/vine/internal/infra/rdb/adapter"
	"gorm.io/gorm"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelUUIDLifecycle(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/models.sqlite"
	db, err := openConnection(Option{ConnURL: connURL})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })
	require.NoError(t, db.AutoMigrate(new(UModel), new(UDeletableModel)))

	t.Run("soft deletion", func(t *testing.T) {
		supplied := uuid.NewV7()
		rows := []*UModel{new(UModel), new(UModel), new(UModel{Id: supplied})}
		require.NoError(t, db.Create(&rows).Error)
		seen := map[uuid.UUID]bool{}
		for _, row := range rows {
			parsed, err := uuid.Parse(row.Id.String())
			require.NoError(t, err)
			assert.Equal(t, byte(7), parsed[6]>>4)
			assert.False(t, seen[row.Id])
			seen[row.Id] = true
			var loaded UModel
			require.NoError(t, db.First(&loaded, "id = ?", row.Id).Error)
			assert.Equal(t, row.Id, loaded.Id)
		}
		assert.Equal(t, supplied, rows[2].Id)
		dao := NewDao[*UModel](db)
		dao.Delete(rows[0])
		assert.Equal(t, 2, dao.Query().Count())
		var deleted UModel
		require.NoError(t, db.Unscoped().First(&deleted, "id = ?", rows[0].Id).Error)
		assert.True(t, deleted.DeletedAt.Valid)
	})

	t.Run("physical deletion", func(t *testing.T) {
		supplied := uuid.NewV7()
		rows := []*UDeletableModel{new(UDeletableModel), new(UDeletableModel{Id: supplied})}
		require.NoError(t, db.Create(&rows).Error)
		parsed, err := uuid.Parse(rows[0].Id.String())
		require.NoError(t, err)
		assert.Equal(t, byte(7), parsed[6]>>4)
		assert.Equal(t, supplied, rows[1].Id)
		dao := NewDao[*UDeletableModel](db)
		loaded, ok := dao.First("id = ?", rows[0].Id)
		require.True(t, ok)
		assert.Equal(t, rows[0].Id, loaded.Id)
		dao.Delete(loaded)
		var count int64
		require.NoError(t, db.Unscoped().Model(new(UDeletableModel)).Where("id = ?", loaded.Id).Count(&count).Error)
		assert.Zero(t, count)
	})
}

func TestUUIDColumnTypes(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/uuid.sqlite"
	db, err := openConnection(Option{ConnURL: connURL})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })
	require.NoError(t, db.AutoMigrate(new(UModel)))
	require.NoError(t, db.AutoMigrate(new(UModel)))
	columns, err := db.Migrator().ColumnTypes(new(UModel))
	require.NoError(t, err)
	for _, column := range columns {
		if column.Name() == "id" {
			assert.Equal(t, "text", column.DatabaseTypeName())
		}
	}
	row := new(UModel)
	require.NoError(t, db.Create(row).Error)
	var stored struct {
		ID      string
		Storage string
	}
	require.NoError(t, db.Raw("SELECT id, typeof(id) AS storage FROM u_models").Scan(&stored).Error)
	assert.Equal(t, row.Id.String(), stored.ID)
	assert.Equal(t, "text", stored.Storage)

	// Verify PostgreSQL schema and insert binding without requiring a running server.
	pg, err := gorm.Open(adapter.NewDialector("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{
		DisableAutomaticPing: true, DryRun: true, SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	sqlDB, err := pg.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, model := range []any{new(UModel), new(UDeletableModel)} {
		stmt := new(gorm.Statement{DB: pg})
		require.NoError(t, stmt.Parse(model))
		assert.Equal(t, "uuid", pg.Migrator().FullDataTypeOf(stmt.Schema.LookUpField("Id")).SQL)
		result := pg.Create(model)
		require.NoError(t, result.Error)
		valuer, ok := result.Statement.Vars[0].(driver.Valuer)
		require.True(t, ok)
		value, err := valuer.Value()
		require.NoError(t, err)
		parsed, err := uuid.Parse(value.(string))
		require.NoError(t, err)
		assert.Equal(t, byte(7), parsed[6]>>4)
	}
}
