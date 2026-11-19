package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

func schemaSQL(gormDB *gorm.DB, sqliteSQL string, pgsqlSQL string) string {
	switch gormDB.Dialector.Name() {
	case "postgres":
		return pgsqlSQL
	default:
		return sqliteSQL
	}
}

// Install the live-row index before dropping the legacy index that also covers
// deleted rows. Restarts leave the existing live-row index in place.
func ensureLiveNameSchema(db *gorm.DB, table string, sqliteSQL string, pgsqlSQL string) {
	ex.PanicIfError(db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(schemaSQL(tx, sqliteSQL, pgsqlSQL)).Error; err != nil {
			return err
		}
		index := "uk_" + table + "_name"
		if tx.Dialector.Name() == "postgres" {
			// An unqualified DROP could find another schema's index on a later startup.
			var schema string
			if err := tx.Raw("SELECT quote_ident(current_schema())").Scan(&schema).Error; err != nil {
				return err
			}
			index = schema + "." + index
		}
		return tx.Exec("DROP INDEX IF EXISTS " + index).Error
	}))
}
