package rdb

import (
	"go.yorun.ai/vine/core/ex"
	"gorm.io/gorm"

	"go.yorun.ai/vine/util/vpre"
)

// Query builds and executes typed queries for M.
type Query[M ModelConstraint] struct {
	gormDB *gorm.DB

	conditions []any
	limit      *int
	offset     *int
	order      *string
}

// Limit sets the maximum number of records and returns this query.
// It panics when limit is not greater than zero.
func (q *Query[M]) Limit(limit int) *Query[M] {
	vpre.Check(limit > 0, "query limit must be greater than 0")
	q.limit = &limit
	return q
}

// Offset sets the number of records to skip and returns this query.
// It panics when offset is negative.
func (q *Query[M]) Offset(offset int) *Query[M] {
	vpre.Check(offset >= 0, "query offset must not be negative")
	q.offset = &offset
	return q
}

// Order sets the GORM ordering expression and returns this query.
// The expression is SQL syntax and must not contain untrusted input.
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

// First returns the first matching model and true, or the zero value and false
// when no record matches. It applies offset and order with a limit of one.
// Database failures panic.
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

// Exists reports whether a matching record exists without loading a model or
// invoking model AfterFind hooks. It applies offset and order with a limit of one
// and follows the model's soft deletion scope. Database failures panic.
func (q *Query[M]) Exists() bool {
	var matches []int
	result := q.queryDB().Model(new(M)).Select("1").Limit(1).Find(&matches, q.conditions...)
	ex.PanicIfError(result.Error)
	return len(matches) > 0
}

// List returns matching models using the configured limit, offset, and order.
// An empty result may be nil. Database failures panic.
func (q *Query[M]) List() []M {
	var models []M
	result := q.queryDB().Find(&models, q.conditions...)
	ex.PanicIfError(result.Error)
	return models
}

// Count returns the matching record count as int.
// It passes the configured limit, offset, and order to GORM rather than clearing them.
// Database failures panic.
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
