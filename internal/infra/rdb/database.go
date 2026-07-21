package rdb

import (
	"context"
	"reflect"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/util/vpre"
	"gorm.io/gorm"
)

type Option struct {
	ConnURL     string
	MaxOpenConn int
}

func defaultOption() *Option {
	return &Option{
		MaxOpenConn: defaultMaxOpenConns,
	}
}

type TypeAdder func(daoType reflect.Type)

type DatabaseSpec interface {
	InitOption(option *Option)
	InitDao(add TypeAdder)

	mustBeDatabase()
}

type Database struct {
	app.BaseFrameworkComponent[*DatabaseMinder]
}

func (*Database) InitOption(option *Option) {}

func (*Database) InitDao(addDao TypeAdder) {}

func (*Database) mustBeDatabase() {}

type DatabaseMinder struct {
	app.BaseFrameworkComponentMinder

	database app.FrameworkComponent
	option   *Option
	daoTypes []reflect.Type
	gormDB   *gorm.DB
}

func (m *DatabaseMinder) InitComponent(component app.FrameworkComponent) {
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
	m.gormDB = gormDB
}

func (m *DatabaseMinder) Component() app.FrameworkComponent {
	return m.database
}

func (m *DatabaseMinder) Bind(b *di.Binder) {
	for _, daoType := range m.daoTypes {
		b.Bind(daoType).ToFactory(func(ctx context.Context, logger *logger.Logger) any {
			return m.instantiateDao(daoType, ctx, logger)
		})
	}
}

func (m *DatabaseMinder) instantiateDao(daoType reflect.Type, ctx context.Context, logger *logger.Logger) any {
	daoValue := reflect.New(daoType.Elem())
	dao, ok := daoValue.Interface().(_GormDBSetter)
	vpre.Check(ok, "dao type %s must embed rdb.Dao[...] to receive gorm db", daoType)
	dao.setGormDB(m.gormDB.WithContext(contextWithLogger(ctx, logger)))
	return daoValue.Interface()
}

func (m *DatabaseMinder) AfterAppStop() {
	closeConnection(m.option.ConnURL)
}
