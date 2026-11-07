package rdb

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type filteredTestModel struct {
	UModel
	Group  string  `gorm:"column:group_name"`
	Count  int     `gorm:"column:count"`
	Active bool    `gorm:"column:active"`
	Note   *string `gorm:"column:note"`
}

type filteredPhysicalModel struct {
	UDeletableModel
	Name string `gorm:"column:name"`
}

func newFilteredTestDao(t *testing.T) (Dao[*filteredTestModel], *gorm.DB) {
	t.Helper()
	db, err := openConnection(Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/filtered.sqlite",
		MaxOpenConn: 1,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(new(filteredTestModel)))
	return NewDao[*filteredTestModel](db), db
}

func TestFilteredUpdatePreservesValuesAndScope(t *testing.T) {
	dao, db := newFilteredTestDao(t)
	first := dao.Create(new(filteredTestModel{
		Group:  "selected",
		Count:  3,
		Active: true,
		Note:   new("original"),
	}))
	second := dao.Create(new(filteredTestModel{Group: "selected"}))
	other := dao.Create(new(filteredTestModel{Group: "other", Active: true}))

	filtered := dao.Filter("group_name = ?", "selected")
	affected := filtered.Update(Patch{
		"active": false,
		"note":   nil,
		"count":  gorm.Expr("count + ?", 2),
	})
	require.Equal(t, 2, affected)
	got, ok := dao.First(first.Id)
	require.True(t, ok)
	assert.False(t, got.Active)
	assert.Nil(t, got.Note)
	assert.Equal(t, 5, got.Count)
	got, ok = dao.First(second.Id)
	require.True(t, ok)
	assert.Equal(t, 2, got.Count)
	got, ok = dao.First(other.Id)
	require.True(t, ok)
	assert.True(t, got.Active)
	assert.Equal(t, 0, got.Count)

	// Reusing the selection must not accumulate previous updates or predicates.
	assert.Equal(t, 2, filtered.Update(Patch{"count": 0}))
	assert.Equal(t, 0, dao.Filter("group_name = ?", "missing").Update(Patch{"count": 1}))
	require.NoError(t, db.Delete(new(filteredTestModel), "id = ?", first.Id).Error)
	assert.Equal(t, 1, filtered.Update(Patch{"count": 7}))
}

func TestFilteredNormalizesUUIDConditionsAndSoftDeletes(t *testing.T) {
	dao, db := newFilteredTestDao(t)
	first := dao.Create(new(filteredTestModel{Group: "first"}))
	second := dao.Create(new(filteredTestModel{Group: "second"}))
	conditions := map[string]any{"id": first.Id}
	filtered := dao.Filter(conditions)
	conditions["id"] = second.Id
	assert.Equal(t, 1, filtered.Update(Patch{"group_name": "changed"}))
	assert.Equal(t, 1, dao.Filter(first.Id).Delete())
	assert.Equal(t, 0, dao.Filter(first.Id).Delete())
	_, exists := dao.First(first.Id)
	assert.False(t, exists)
	got, exists := dao.First(second.Id)
	require.True(t, exists)
	assert.Equal(t, "second", got.Group)
	var deleted filteredTestModel
	require.NoError(t, db.Unscoped().First(&deleted, "id = ?", first.Id).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, "changed", deleted.Group)
	assert.Equal(t, 0, dao.Filter(uuid.NewV7()).Delete())
}

func TestFilteredPhysicallyDeletesModelsWithoutDeletedAt(t *testing.T) {
	_, db := newFilteredTestDao(t)
	require.NoError(t, db.AutoMigrate(new(filteredPhysicalModel)))
	dao := NewDao[*filteredPhysicalModel](db)
	record := dao.Create(new(filteredPhysicalModel{Name: "selected"}))
	assert.Equal(t, 1, dao.Filter(record.Id).Delete())
	var count int64
	require.NoError(t, db.Unscoped().Model(new(filteredPhysicalModel)).Count(&count).Error)
	assert.Zero(t, count)
}

func TestFilteredRequiresConditions(t *testing.T) {
	dao, db := newFilteredTestDao(t)
	dao.Create(new(filteredTestModel{Group: "first"}))
	permissiveDao := NewDao[*filteredTestModel](db.Session(new(gorm.Session{AllowGlobalUpdate: true})))
	require.Panics(t, func() { permissiveDao.Filter() })
	// GORM's default missing-WHERE protection rejects empty predicates.
	for _, conditions := range [][]any{{""}, {map[string]any{}}} {
		require.Panics(t, func() { dao.Filter(conditions...).Update(Patch{"count": 1}) })
		require.Panics(t, func() { dao.Filter(conditions...).Delete() })
	}
	assert.Equal(t, 1, dao.Query().Count())
}

func TestFilteredDatabaseErrorsPanic(t *testing.T) {
	dao, db := newFilteredTestDao(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	require.Panics(t, func() { dao.Filter("group_name = ?", "selected").Update(Patch{"count": 1}) })
	require.Panics(t, func() { dao.Filter("group_name = ?", "selected").Delete() })
	require.Panics(t, func() { dao.One("group_name = ?", "selected").Update(Patch{"count": 1}) })
	require.Panics(t, func() { dao.One("group_name = ?", "selected").Delete() })
}

func TestOneUpdateAndDeleteRequireExactlyOneRow(t *testing.T) {
	dao, _ := newFilteredTestDao(t)
	first := dao.Create(new(filteredTestModel{Group: "selected", Count: 3}))
	dao.Create(new(filteredTestModel{Group: "selected", Count: 4}))
	require.Panics(t, func() { dao.One() })
	require.Panics(t, func() { dao.One(uuid.NewV7()).Update(Patch{"count": 1}) })
	require.Panics(t, func() { dao.One(uuid.NewV7()).Delete() })
	require.Panics(t, func() {
		dao.One("group_name = ?", "selected").Update(Patch{"count": 9})
	})
	got, exists := dao.First(first.Id)
	require.True(t, exists)
	assert.Equal(t, 3, got.Count)
	require.Panics(t, func() { dao.One("group_name = ?", "selected").Delete() })
	assert.Equal(t, 2, dao.Query().Count())

	one := dao.One(first.Id)
	assert.Equal(t, 1, one.Update(Patch{"count": gorm.Expr("count + ?", 1)}))
	assert.Equal(t, 1, one.Update(Patch{"active": false, "note": nil}))
	got, exists = dao.First(first.Id)
	require.True(t, exists)
	assert.Equal(t, 4, got.Count)
	assert.Equal(t, 1, one.Delete())
	require.Panics(t, func() { one.Delete() })
	assert.Equal(t, 1, dao.Query().Count())
}

func TestOnePhysicalDeleteRollsBackMultipleMatches(t *testing.T) {
	_, db := newFilteredTestDao(t)
	require.NoError(t, db.AutoMigrate(new(filteredPhysicalModel)))
	dao := NewDao[*filteredPhysicalModel](db)
	first := dao.Create(new(filteredPhysicalModel{Name: "selected"}))
	dao.Create(new(filteredPhysicalModel{Name: "selected"}))
	require.Panics(t, func() { dao.One("name = ?", "selected").Delete() })
	assert.Equal(t, 2, dao.Query().Count())
	assert.Equal(t, 1, dao.One(first.Id).Delete())
	assert.Equal(t, 1, dao.Query().Count())
}

func TestOneFailureRollsBackWithinExistingTransaction(t *testing.T) {
	dao, db := newFilteredTestDao(t)
	first := dao.Create(new(filteredTestModel{Group: "selected", Count: 3}))
	dao.Create(new(filteredTestModel{Group: "selected", Count: 4}))
	// One must keep its rollback guarantee even with nested transactions disabled
	// on the supplied connection, without changing that connection's config.
	db = db.Session(new(gorm.Session{DisableNestedTransaction: true}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		transactionDao := NewDao[*filteredTestModel](tx)
		require.Panics(t, func() {
			transactionDao.One("group_name = ?", "selected").Update(Patch{"count": 9})
		})
		got, exists := transactionDao.First(first.Id)
		require.True(t, exists)
		assert.Equal(t, 3, got.Count)
		assert.Equal(t, 1, transactionDao.One(first.Id).Update(Patch{"count": 5}))
		assert.True(t, tx.Config.DisableNestedTransaction)
		return nil
	}))
	got, exists := dao.First(first.Id)
	require.True(t, exists)
	assert.Equal(t, 5, got.Count)
}
