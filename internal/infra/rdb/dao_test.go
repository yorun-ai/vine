package rdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type daoTestModel struct {
	Model
	Name  string `gorm:"column:name"`
	Value string `gorm:"column:value"`
}

func (*daoTestModel) TableName() string {
	return "dao_test_models"
}

func openDaoTestDB(t *testing.T) *Dao[*daoTestModel] {
	t.Helper()

	db, err := openConnection(Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/dao.sqlite",
		MaxOpenConn: 1,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&daoTestModel{}))

	dao := NewDao[*daoTestModel](db)
	return &dao
}

func TestDaoCreateReturnsInsertedModel(t *testing.T) {
	dao := openDaoTestDB(t)

	model := dao.Create(&daoTestModel{
		Name:  "alpha",
		Value: "one",
	})

	require.NotNil(t, model)
	assert.NotZero(t, model.Id)
	assert.False(t, model.CreatedAt.IsZero())
	assert.False(t, model.UpdatedAt.IsZero())
	assert.Equal(t, "alpha", model.Name)
	assert.Equal(t, "one", model.Value)
}

func TestDaoUpdateReturnsUpdatedModel(t *testing.T) {
	dao := openDaoTestDB(t)
	model := dao.Create(&daoTestModel{
		Name:  "alpha",
		Value: "one",
	})
	time.Sleep(time.Millisecond)

	updated := dao.Update(model, Patch{
		"value": "two",
	})

	require.NotNil(t, updated)
	assert.Equal(t, model.Id, updated.Id)
	assert.Equal(t, "alpha", updated.Name)
	assert.Equal(t, "two", updated.Value)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestDaoDeleteRemovesModel(t *testing.T) {
	dao := openDaoTestDB(t)
	model := dao.Create(&daoTestModel{
		Name:  "alpha",
		Value: "one",
	})

	dao.Delete(model)

	_, ok := dao.First("id = ?", model.Id)
	assert.False(t, ok)
}
