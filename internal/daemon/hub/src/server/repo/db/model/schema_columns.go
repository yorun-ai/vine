package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

// dropColumns removes retired columns without changing existing rows. Each table
// is changed atomically, and repeated schema initialization is a no-op.
func dropColumns(db *gorm.DB, table string, names ...string) {
	ex.PanicIfError(db.Transaction(func(tx *gorm.DB) error {
		columns, err := tableColumnNames(tx, table)
		if err != nil {
			return err
		}
		for _, name := range names {
			if columns[name] {
				if err := tx.Exec("ALTER TABLE " + table + " DROP COLUMN " + name).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}))
}

// tableColumnNames returns the stored columns of one table. Hub reads the
// catalog instead of the table itself: a migration adds and drops columns while
// it runs, and PostgreSQL refuses a prepared statement whose result type changed
// since its plan was cached, which is what reading a table before and after its
// own DDL would ask for.
func tableColumnNames(db *gorm.DB, table string) (map[string]bool, error) {
	query := "SELECT column_name AS name FROM information_schema.columns WHERE table_name = ? AND table_schema = ANY(current_schemas(false))"
	if db.Dialector.Name() != "postgres" {
		query = "SELECT name FROM pragma_table_info(?)"
	}
	columnNames := []string{}
	if err := db.Raw(query, table).Scan(&columnNames).Error; err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(columnNames))
	for _, name := range columnNames {
		names[name] = true
	}
	return names, nil
}
