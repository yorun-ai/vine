package rdb

import (
	"context"
	"reflect"

	"go.yorun.ai/vine/app"
	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/util/vpre"
	"gorm.io/gorm"
)

// Option configures a relational database component.
type Option struct {
	ConnURL     string
	MaxOpenConn int
}

func defaultOption() *Option {
	return &Option{
		MaxOpenConn: defaultMaxOpenConns,
	}
}

// TypeAdder registers a DAO pointer type with a database specification.
type TypeAdder func(daoType reflect.Type)

type _SchemaDao interface {
	_GormDBSetter
	EnsureSchema()
}

// DatabaseSpec declares database options and DAOs.
type DatabaseSpec interface {
	// InitOption configures the connection URL and pool options before opening it.
	InitOption(option *Option)
	// InitDao registers DAO pointer types to initialize and bind.
	InitDao(add TypeAdder)

	mustBeDatabase()
}

// Database provides database component lifecycle and DAO registration hooks.
type Database struct {
	app.BaseManagedComponent[*DatabaseManager]
}

// InitOption is the default configuration hook and leaves the defaults unchanged.
// Override it on the embedding component to set ConnURL and pool options.
func (*Database) InitOption(option *Option) {}

// InitDao is the default DAO registration hook and registers no DAOs.
// Override it on the embedding component to register DAO pointer types with addDao.
func (*Database) InitDao(addDao TypeAdder) {}

func (*Database) mustBeDatabase() {}

// DatabaseManager owns database connections and DAO dependency bindings.
type DatabaseManager struct {
	app.BaseComponentManager

	database app.ManagedComponent
	option   *Option
	daoTypes []reflect.Type
	gormDB   *gorm.DB
}

// InitComponent reads the component specification, opens or reuses its connection,
// and initializes each registered DAO schema before binding. Initialization failures panic.
func (m *DatabaseManager) InitComponent(component app.ManagedComponent) {
	m.database = component
	m.option = defaultOption()

	spec := component.(DatabaseSpec)
	spec.InitOption(m.option)

	m.daoTypes = []reflect.Type{}
	spec.InitDao(func(daoType reflect.Type) {
		m.daoTypes = append(m.daoTypes, daoType)
	})

	gormDB, err := openConnection(*m.option)
	vpre.CheckNilError(err, "gorm open failed")
	initialized := false
	defer func() {
		if !initialized {
			closeConnection(m.option.ConnURL)
		}
	}()
	m.gormDB = gormDB
	for _, daoType := range m.daoTypes {
		m.ensureDaoSchema(daoType)
	}
	initialized = true
}

func (m *DatabaseManager) ensureDaoSchema(daoType reflect.Type) {
	daoValue := reflect.New(daoType.Elem())
	dao, ok := reflect.TypeAssert[_SchemaDao](daoValue)
	vpre.Check(ok, "dao type %s must embed rdb.Dao[...] to receive gorm db", daoType)
	dao.setGormDB(m.gormDB)
	dao.EnsureSchema()
}

// Component returns the managed database component.
func (m *DatabaseManager) Component() app.ManagedComponent {
	return m.database
}

// Bind registers DAO factories with the dependency container.
// Each resolved DAO uses the injected context and logger on the shared connection.
func (m *DatabaseManager) Bind(b *di.Binder) {
	for _, daoType := range m.daoTypes {
		b.Bind(daoType).ToFactory(func(ctx context.Context, logger *logger.Logger) any {
			return m.instantiateDao(daoType, ctx, logger)
		})
	}
}

func (m *DatabaseManager) instantiateDao(daoType reflect.Type, ctx context.Context, logger *logger.Logger) any {
	daoValue := reflect.New(daoType.Elem())
	dao, ok := reflect.TypeAssert[_GormDBSetter](daoValue)
	vpre.Check(ok, "dao type %s must embed rdb.Dao[...] to receive gorm db", daoType)
	dao.setGormDB(m.gormDB.WithContext(contextWithLogger(ctx, logger)))
	return daoValue.Interface()
}

// AfterAppStop releases this component's connection reference.
// The connection closes when the last component sharing its URL releases it.
func (m *DatabaseManager) AfterAppStop() {
	closeConnection(m.option.ConnURL)
}
