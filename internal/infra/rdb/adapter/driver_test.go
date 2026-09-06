package adapter

import (
	"context"
	"database/sql"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUUIDQueryParameters(t *testing.T) {
	url := "sqlite://" + t.TempDir() + "/params.sqlite"
	db, err := openDriverTestDB(t, url)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(new(driverTestModel)))
	for _, prepared := range []bool{false, true} {
		name := "direct"
		if prepared {
			name = "prepared"
		}
		t.Run(name, func(t *testing.T) {
			db := db.Session(&gorm.Session{PrepareStmt: prepared})
			if prepared {
				t.Cleanup(func() { db.ConnPool.(*gorm.PreparedStmtDB).Close() })
			}
			rows := []*driverTestModel{new(driverTestModel{Name: "a"}), new(driverTestModel{Name: "b"})}
			require.NoError(t, db.Create(&rows).Error)
			ids := []uuid.UUID{rows[0].Id, rows[1].Id}
			var found []driverTestModel
			require.NoError(t, db.Where("id IN ?", ids).Find(&found).Error)
			require.Len(t, found, 2)
			found = nil
			require.NoError(t, db.Raw("SELECT * FROM driver_test_models WHERE id IN (?)", ids).Scan(&found).Error)
			require.Len(t, found, 2)
			var row driverTestModel
			require.NoError(t, db.Where("id = ?", &ids[0]).First(&row).Error)
			assert.Equal(t, ids[0], row.Id)
			require.NoError(t, db.Raw("SELECT * FROM driver_test_models WHERE id = @id", sql.Named("id", ids[1])).Scan(&row).Error)
			assert.Equal(t, ids[1], row.Id)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return tx.Exec("UPDATE driver_test_models SET name = ? WHERE id = ?", "updated", ids[0]).Error
			}))
			row = driverTestModel{}
			require.NoError(t, db.Where("id = ?", ids[0]).First(&row).Error)
			assert.Equal(t, "updated", row.Name)
			found = nil
			require.NoError(t, db.Where("id IN ?", []uuid.UUID{}).Find(&found).Error)
			assert.Empty(t, found)
		})
	}
}

func TestUUIDSQLPreparedParameters(t *testing.T) {
	url := "sqlite://" + t.TempDir() + "/sql.sqlite"
	db, err := openDriverTestDB(t, url)
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	ctx := context.Background()
	stmt, err := pool.PrepareContext(ctx, "SELECT ?, ?, ?, ?")
	require.NoError(t, err)
	t.Cleanup(func() { _ = stmt.Close() })
	id := uuid.NewV7()
	var text string
	var number int
	var bytes []byte
	var null any
	require.NoError(t, stmt.QueryRowContext(ctx, id, 42, []byte{1, 2, 3}, (*uuid.UUID)(nil)).Scan(&text, &number, &bytes, &null))
	assert.Equal(t, id.String(), text)
	assert.Equal(t, 42, number)
	assert.Equal(t, []byte{1, 2, 3}, bytes)
	assert.Nil(t, null)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, tx.StmtContext(ctx, stmt).QueryRowContext(ctx, &id, 7, []byte{4}, nil).Scan(&text, &number, &bytes, &null))
	assert.Equal(t, id.String(), text)
	require.NoError(t, tx.Commit())
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	assert.ErrorIs(t, stmt.QueryRowContext(canceled, id, 1, nil, nil).Err(), context.Canceled)
}

// Fixtures deliberately use plain GORM models without depending on the parent rdb package.
type driverTestModel struct {
	Id   uuid.UUID `gorm:"primaryKey;type:uuid;serializer:uuid"`
	Name string
}

func openDriverTestDB(t *testing.T, url string) (*gorm.DB, error) {
	t.Helper()
	db, err := gorm.Open(NewDialector(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = pool.Close() })
	return db, nil
}
