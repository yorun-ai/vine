package model

import "gorm.io/gorm"

func schemaSQL(gormDB *gorm.DB, sqliteSQL string, pgsqlSQL string) string {
	switch gormDB.Dialector.Name() {
	case "postgres":
		return pgsqlSQL
	default:
		return sqliteSQL
	}
}
