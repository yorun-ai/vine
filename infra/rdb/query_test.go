package rdb

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type queryTestModel struct {
	Model
	Name string `gorm:"column:name"`
}

func (*queryTestModel) TableName() string {
	return "query_test_models"
}

func TestQueryCount(t *testing.T) {
	config := Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/query.sqlite",
		MaxOpenConn: 1,
	}
	db, err := openConnection(config)
	require.NoError(t, err)

	err = db.AutoMigrate(&queryTestModel{})
	require.NoError(t, err)

	err = db.Create([]*queryTestModel{
		{Name: "alpha"},
		{Name: "alpha"},
		{Name: "beta"},
	}).Error
	require.NoError(t, err)

	dao := NewDao[*queryTestModel](db)
	query := dao.Query("name = ?", "alpha")
	assert.Equal(t, 2, query.Count())
}

func TestQueryFirstAndList(t *testing.T) {
	db, err := openConnection(Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/query.sqlite",
		MaxOpenConn: 1,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&queryTestModel{})
	require.NoError(t, err)

	err = db.Create([]*queryTestModel{
		{Name: "alpha"},
		{Name: "beta"},
	}).Error
	require.NoError(t, err)

	dao := NewDao[*queryTestModel](db)

	first, ok := dao.Query("name = ?", "alpha").First()
	require.True(t, ok)
	require.NotNil(t, first)
	assert.Equal(t, "alpha", first.Name)

	list := dao.Query("name IN ?", []string{"alpha", "beta"}).List()
	assert.Len(t, list, 2)
}

func TestQueryFirstLimitsRows(t *testing.T) {
	db, err := openConnection(Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/query.sqlite",
		MaxOpenConn: 1,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&queryTestModel{})
	require.NoError(t, err)

	err = db.Create([]*queryTestModel{
		{Name: "alpha"},
		{Name: "alpha"},
	}).Error
	require.NoError(t, err)

	sqlLogger := &_QuerySQLLogger{}
	dao := NewDao[*queryTestModel](db.Session(&gorm.Session{
		Logger: sqlLogger,
	}))

	first, ok := dao.Query("name = ?", "alpha").First()
	require.True(t, ok)
	require.NotNil(t, first)
	assert.Contains(t, strings.ToLower(sqlLogger.LastSQL()), "limit 1")
}

func TestQueryLimitOffsetAndOrder(t *testing.T) {
	db, err := openConnection(Option{
		ConnURL:     "sqlite://" + t.TempDir() + "/query.sqlite",
		MaxOpenConn: 1,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&queryTestModel{})
	require.NoError(t, err)

	err = db.Create([]*queryTestModel{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}).Error
	require.NoError(t, err)

	dao := NewDao[*queryTestModel](db)

	list := dao.Query().
		Order("name desc").
		Offset(1).
		Limit(1).
		List()

	require.Len(t, list, 1)
	assert.Equal(t, "beta", list[0].Name)
}

type queryExistsTestModel struct {
	UModel
	Name string
}

func (*queryExistsTestModel) AfterFind(_ *gorm.DB) error {
	return errors.New("Exists must not invoke AfterFind")
}

func TestQueryExists(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/exists.sqlite"
	db, err := openConnection(Option{ConnURL: connURL})
	require.NoError(t, err)
	t.Cleanup(func() { closeConnection(connURL) })
	require.NoError(t, db.AutoMigrate(new(queryExistsTestModel)))
	sqlLogger := new(_QuerySQLLogger)
	dao := NewDao[*queryExistsTestModel](db.Session(new(gorm.Session{Logger: sqlLogger})))
	assert.False(t, dao.Query().Exists())
	first := dao.Create(new(queryExistsTestModel{Name: "alpha"}))
	dao.Create(new(queryExistsTestModel{Name: "beta"}))

	query := dao.Query("name = ?", "alpha").Limit(5).Order("name").Offset(0)
	assert.True(t, query.Exists())
	sql := strings.ToLower(sqlLogger.LastSQL())
	assert.Contains(t, sql, "select 1 from")
	assert.Contains(t, sql, "limit 1")
	assert.NotContains(t, sql, "count(")
	assert.NotContains(t, sql, "select *")
	assert.Equal(t, 5, *query.limit)
	assert.False(t, query.Offset(1).Exists())
	assert.True(t, query.Offset(0).Exists())
	assert.True(t, dao.Query().Offset(1).Exists())
	assert.False(t, dao.Query().Offset(2).Exists())
	assert.False(t, dao.Query("name = ?", "missing").Exists())

	dao.Delete(first)
	assert.False(t, dao.Query(first.Id).Exists())
	unscoped := NewDao[*queryExistsTestModel](db.Unscoped())
	assert.True(t, unscoped.Query(first.Id).Exists())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := NewDao[*queryExistsTestModel](db.WithContext(ctx))
	require.Panics(t, func() { canceled.Query().Exists() })
}

type _QuerySQLLogger struct {
	gormLogger.Interface
	mu      sync.Mutex
	lastSQL string
}

func (l *_QuerySQLLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return l
}

func (l *_QuerySQLLogger) Trace(ctx context.Context, begin time.Time, sqlBuilder func() (string, int64), err error) {
	sql, _ := sqlBuilder()
	l.mu.Lock()
	l.lastSQL = sql
	l.mu.Unlock()
}

func (l *_QuerySQLLogger) LastSQL() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastSQL
}
