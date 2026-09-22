package rdb

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/logger"
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

func initTestDatabase(component app.ManagedComponent) *DatabaseManager {
	manager := new(DatabaseManager)
	manager.InitComponent(component)
	return manager
}

type databaseTestConsumer struct {
	DAO *databaseTestDAO `inject:""`
}

func TestDatabaseInitComponentInitializesOptionAndDaoTypes(t *testing.T) {
	connURL := "sqlite://" + t.TempDir() + "/database.sqlite"
	component := &databaseTestComponent{
		connURL: connURL,
	}

	manager := initTestDatabase(component)
	t.Cleanup(manager.AfterAppStop)

	require.NotNil(t, manager.option)
	assert.Equal(t, connURL, manager.option.ConnURL)
	assert.Equal(t, 3, manager.option.MaxOpenConn)
	assert.Equal(t, []reflect.Type{T[*databaseTestDAO]()}, manager.daoTypes)
	assert.NotNil(t, manager.gormDB)

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
	manager := initTestDatabase(component)
	t.Cleanup(manager.AfterAppStop)

	require.NoError(t, manager.gormDB.AutoMigrate(&databaseTestModel{}))

	injector := di.NewInjector(
		func(b *di.Binder) {
			b.Bind(reflect.TypeFor[context.Context]()).ToInstance(context.Background())
			b.BindInstance(logger.New("vine:test"))
			manager.Bind(b)
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

	manager := initTestDatabase(component)
	manager.AfterAppStop()

	sharedGormDBsMu.Lock()
	_, ok := sharedGormDBs[connURL]
	sharedGormDBsMu.Unlock()
	assert.False(t, ok)
}

type schemaDatabaseTestDAO struct {
	Dao[*databaseTestModel]
}

func (d *schemaDatabaseTestDAO) EnsureSchema() {
	if err := d.GormDB().AutoMigrate(&databaseTestModel{}); err != nil {
		panic(err)
	}
}

type schemaDatabaseTestComponent struct {
	databaseTestComponent
}

func (*schemaDatabaseTestComponent) InitDao(addDao TypeAdder) {
	addDao(T[*schemaDatabaseTestDAO]())
}

func TestDatabaseInitializesSchemaBeforeBindingDAOs(t *testing.T) {
	component := &schemaDatabaseTestComponent{connURL: "sqlite://" + t.TempDir() + "/schema.sqlite"}
	manager := initTestDatabase(component)
	t.Cleanup(manager.AfterAppStop)
	require.True(t, manager.gormDB.Migrator().HasTable(&databaseTestModel{}))
	injector := di.NewInjector(func(b *di.Binder) {
		b.Bind(di.T[context.Context]()).ToInstance(context.Background())
		b.BindInstance(logger.New("test:schema"))
		manager.Bind(b)
	})
	require.NotNil(t, injector.Get(T[*schemaDatabaseTestDAO]()))
}

type failingSchemaTestDAO struct {
	Dao[*databaseTestModel]
}

func (*failingSchemaTestDAO) EnsureSchema() {
	panic("schema failed")
}

type failingSchemaTestComponent struct {
	databaseTestComponent
}

func (*failingSchemaTestComponent) InitDao(addDao TypeAdder) {
	addDao(T[*failingSchemaTestDAO]())
}

func TestDatabaseSchemaFailureReleasesConnection(t *testing.T) {
	url := "sqlite://" + t.TempDir() + "/schema.sqlite"
	component := &failingSchemaTestComponent{connURL: url}
	require.PanicsWithValue(t, "schema failed", func() { initTestDatabase(component) })
	sharedGormDBsMu.Lock()
	_, exists := sharedGormDBs[url]
	sharedGormDBsMu.Unlock()
	require.False(t, exists)
}
