package rdb

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
)

type databaseTestModel struct {
	Model
	Name string `gorm:"column:name"`
}

func (*databaseTestModel) TableName() string {
	return "database_test_models"
}

type databaseTestDAO struct {
	Dao[*databaseTestModel]
}

type databaseTestComponent struct {
	Database
	connURL string
}

func (d *databaseTestComponent) InitOption(option *Option) {
	option.ConnURL = d.connURL
	option.MaxOpenConn = 3
}

func (*databaseTestComponent) InitDao(addDao TypeAdder) {
	addDao(T[*databaseTestDAO]())
}

func initTestDatabase(component app.FrameworkComponent) *DatabaseMinder {
	minder := new(DatabaseMinder)
	minder.InitComponent(component)
	return minder
}

type databaseTestConsumer struct {
	DAO *databaseTestDAO `inject:""`
}

func TestDatabaseInitComponentInitializesOptionAndDaoTypes(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/database.sqlite"
	component := &databaseTestComponent{
		connURL: connURL,
	}

	minder := initTestDatabase(component)
	t.Cleanup(minder.AfterAppStop)

	require.NotNil(t, minder.option)
	assert.Equal(t, connURL, minder.option.ConnURL)
	assert.Equal(t, 3, minder.option.MaxOpenConn)
	assert.Equal(t, []reflect.Type{T[*databaseTestDAO]()}, minder.daoTypes)
	assert.NotNil(t, minder.gormDB)

	sharedGormDBsMu.Lock()
	shared := sharedGormDBs[connURL]
	sharedGormDBsMu.Unlock()
	require.NotNil(t, shared)
	assert.Equal(t, 1, shared.refCount)
}

func TestDatabaseBindProvidesExecutionScopedDao(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/database.sqlite"
	component := &databaseTestComponent{
		connURL: connURL,
	}
	minder := initTestDatabase(component)
	t.Cleanup(minder.AfterAppStop)

	require.NoError(t, minder.gormDB.AutoMigrate(&databaseTestModel{}))

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(reflect.TypeFor[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.NewLogger(logger.GlobalOption()))
			minder.Bind(b)
			b.Bind(T[*databaseTestConsumer]()).In(di.TransientScope)
		},
	)

	execution := injector.StartExecution()
	defer execution.CompleteExecution()

	consumer := execution.Get(T[*databaseTestConsumer]()).Interface().(*databaseTestConsumer)
	require.NotNil(t, consumer.DAO)
	require.NotNil(t, consumer.DAO.GormDB())

	model := consumer.DAO.Create(&databaseTestModel{Name: "alpha"})
	assert.NotZero(t, model.Id)
}

func TestDatabaseAfterAppStopReleasesSharedConnection(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/database.sqlite"
	component := &databaseTestComponent{
		connURL: connURL,
	}

	minder := initTestDatabase(component)
	minder.AfterAppStop()

	sharedGormDBsMu.Lock()
	_, ok := sharedGormDBs[connURL]
	sharedGormDBsMu.Unlock()
	assert.False(t, ok)
}
