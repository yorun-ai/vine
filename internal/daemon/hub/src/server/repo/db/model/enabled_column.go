package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

// ensureEnabledColumn adds the enable switch to a table that predates it. The
// ALTER carries DEFAULT TRUE because SQLite and PostgreSQL reject a new NOT NULL
// column without one, and that default is the backfill: a row Hub already stores
// stays published. The schema a fresh database creates declares the same default
// for the same reason: a Hub that predates the switch omits the column, and the
// row it writes stays published.
func ensureEnabledColumn(db *gorm.DB, table string) {
	if !db.Migrator().HasTable(table) {
		return
	}
	columns, err := tableColumnNames(db, table)
	ex.PanicIfError(err)
	if columns["enabled"] {
		return
	}
	ex.PanicIfError(db.Exec("ALTER TABLE " + table + " ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE").Error)
}
