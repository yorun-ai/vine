package rdb

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"

	"go.yorun.ai/vine/util/vpre"
)

type Query[M ModelConstraint] struct {
	gormDB *gorm.DB

	conditions []any
	limit      *int
	offset     *int
	order      *string
}

func (q *Query[M]) Limit(limit int) *Query[M] {
	vpre.Check(limit > 0, "query limit must be greater than 0")
	q.limit = &limit
	return q
}

func (q *Query[M]) Offset(offset int) *Query[M] {
	vpre.Check(offset >= 0, "query offset must not be negative")
	q.offset = &offset
	return q
}

func (q *Query[M]) Order(order string) *Query[M] {
	q.order = &order
	return q
}

func (q *Query[M]) queryDB() *gorm.DB {
	queryDB := q.gormDB
	if q.limit != nil {
		queryDB = queryDB.Limit(*q.limit)
	}
	if q.offset != nil {
		queryDB = queryDB.Offset(*q.offset)
	}
	if q.order != nil {
		queryDB = queryDB.Order(*q.order)
	}
	return queryDB
}

func (q *Query[M]) First() (M, bool) {
	var models []M
	result := q.queryDB().Limit(1).Find(&models, q.conditions...)
	ex.PanicIfError(result.Error)
	if len(models) == 0 {
		var zero M
		return zero, false
	}
	return models[0], true
}

func (q *Query[M]) List() []M {
	var models []M
	result := q.queryDB().Find(&models, q.conditions...)
	ex.PanicIfError(result.Error)
	return models
}

func (q *Query[M]) Count() int {
	var count int64
	queryDB := q.queryDB().Model(new(M))
	if len(q.conditions) > 0 {
		queryDB = queryDB.Where(q.conditions[0], q.conditions[1:]...)
	}
	result := queryDB.Count(&count)
	ex.PanicIfError(result.Error)
	return int(count)
}
