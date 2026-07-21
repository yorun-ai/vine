package rdb

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type _GormDBSetter interface {
	setGormDB(*gorm.DB)
}

type Dao[M ModelConstraint] struct {
	gormDB *gorm.DB
}

type Patch map[string]any

func NewDao[M ModelConstraint](gdb *gorm.DB) Dao[M] {
	return Dao[M]{
		gormDB: gdb,
	}
}

func (d *Dao[M]) setGormDB(gdb *gorm.DB) {
	d.gormDB = gdb
}

func (d *Dao[M]) GormDB() *gorm.DB {
	return d.gormDB
}

func (d *Dao[M]) Query(conditions ...any) *Query[M] {
	return &Query[M]{
		gormDB:     d.gormDB,
		conditions: conditions,
	}
}

func (d *Dao[M]) First(conditions ...any) (M, bool) {
	return d.Query(conditions...).First()
}

func (d *Dao[M]) List(conditions ...any) []M {
	return d.Query(conditions...).List()
}

func (d *Dao[M]) Create(model M) M {
	result := d.gormDB.Clauses(clause.Returning{}).Create(model)
	ex.PanicIfError(result.Error)
	return model
}

func (d *Dao[M]) Update(model M, patch Patch) M {
	patchMap := map[string]any(patch)
	result := d.gormDB.Model(model).Clauses(clause.Returning{}).Updates(patchMap)
	ex.PanicIfError(result.Error)
	return model
}

func (d *Dao[M]) Delete(model M) {
	result := d.gormDB.Delete(&model)
	ex.PanicIfError(result.Error)
}
