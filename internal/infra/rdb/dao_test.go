package rdb

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	dao.GormDB().Config.NowFunc = func() time.Time { return now }
	model := dao.Create(&daoTestModel{
		Name:  "alpha",
		Value: "one",
	})
	now = now.Add(time.Second)

	updated := dao.Update(model, Patch{
		"value": "two",
	})

	require.NotNil(t, updated)
	assert.Equal(t, model.Id, updated.Id)
	assert.Equal(t, "alpha", updated.Name)
	assert.Equal(t, "two", updated.Value)
	assert.Equal(t, now, updated.UpdatedAt)
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

type uuidDaoTestModel struct {
	UModel
	Name string
}

func TestDaoUUIDModelCRUD(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/uuid-dao.sqlite"
	db, err := openConnection(Option{ConnURL: connURL})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })
	require.NoError(t, db.AutoMigrate(new(uuidDaoTestModel)))
	dao := NewDao[*uuidDaoTestModel](db)
	row := dao.Create(new(uuidDaoTestModel{Name: "before"}))
	require.NotEmpty(t, row.Id)
	id := row.Id
	dao.Update(row, Patch{"name": "after"})
	loaded, ok := dao.First("id = ?", id)
	require.True(t, ok)
	assert.Equal(t, id, loaded.Id)
	assert.Equal(t, "after", loaded.Name)
	dao.Delete(loaded)
	_, ok = dao.First("id = ?", id)
	assert.False(t, ok)
}

type externalUUIDTestModel struct {
	UModel
	HookID string `gorm:"-"`
}

func (m *externalUUIDTestModel) BeforeCreate(_ *gorm.DB) error {
	m.HookID = m.Id.String()
	return nil
}

func TestNewDaoRegistersUUIDGenerationOnExternalConnection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/external.sqlite"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, db.AutoMigrate(new(externalUUIDTestModel)))
	dao := NewDao[*externalUUIDTestModel](db)
	row := dao.Create(new(externalUUIDTestModel))
	require.NotZero(t, row.Id)
	require.Equal(t, row.Id.String(), row.HookID)
	direct := new(externalUUIDTestModel)
	require.NoError(t, db.Create(direct).Error)
	require.NotZero(t, direct.Id)
	require.Equal(t, direct.Id.String(), direct.HookID)
}
