package rdb

import (
	"go.yorun.ai/vine/core/ex"
	"go.yorun.ai/vine/util/vpre"
	"gorm.io/gorm"
)

// Filtered selects a model's records for updates and deletion.
// It stores conditions without fetching records and does not support ordering or paging.
type Filtered[M ModelConstraint] struct {
	gormDB     *gorm.DB
	conditions []any
	one        bool
}

// Update applies patch to matching records and returns the affected row count.
// Patch preserves nil and zero values and supports GORM expressions.
// Database failures panic; no matching records returns zero for Filter.
// For One, the write is transactional and rolls back and panics unless exactly
// one row is affected; success returns 1.
func (f *Filtered[M]) Update(patch Patch) int {
	return f.write(func(db *gorm.DB) *gorm.DB {
		return db.Model(new(M)).Updates(map[string]any(patch))
	})
}

// Delete deletes matching records and returns the affected row count.
// Models with DeletedAt use soft deletion; models without it are physically deleted.
// Database failures panic; no matching records returns zero for Filter.
// For One, the write is transactional and rolls back and panics unless exactly
// one row is affected; success returns 1.
func (f *Filtered[M]) Delete() int {
	return f.write(func(db *gorm.DB) *gorm.DB {
		return db.Delete(new(M))
	})
}

func (f *Filtered[M]) write(operation func(db *gorm.DB) *gorm.DB) int {
	var affected int64
	execute := func(db *gorm.DB) error {
		filteredDB := db.Where(f.conditions[0], f.conditions[1:]...)
		result := operation(filteredDB)
		ex.PanicIfError(result.Error)
		vpre.Check(!f.one || result.RowsAffected == 1,
			"one write must affect exactly one row, affected %d", result.RowsAffected)
		affected = result.RowsAffected
		return nil
	}
	if f.one {
		// Savepoints preserve rollback even if the caller catches the panic
		// and continues an existing transaction.
		db := f.gormDB.Session(new(gorm.Session))
		db.Config.DisableNestedTransaction = false
		ex.PanicIfError(db.Transaction(execute))
	} else {
		ex.PanicIfError(execute(f.gormDB))
	}
	return int(affected)
}
