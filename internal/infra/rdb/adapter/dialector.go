package adapter

import (
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// NewDialector selects PostgreSQL or SQLite with UUID column and parameter support.
// SQLite URLs begin with sqlite://; other URLs are passed to PostgreSQL.
func NewDialector(connURL string) gorm.Dialector {
	switch {
	case strings.HasPrefix(connURL, "sqlite://"):
		return &_SQLiteDialector{Dialector: &sqlite.Dialector{DriverName: "vine-infra-rdb-sqlite", DSN: strings.TrimPrefix(connURL, "sqlite://")}}
	default:
		return &_PostgresDialector{Dialector: postgres.New(postgres.Config{DriverName: "vine-infra-rdb-pgx", DSN: connURL}).(*postgres.Dialector)}
	}
}

// _SQLiteDialector stores UUID columns as text because SQLite has no UUID type.
type _SQLiteDialector struct{ *sqlite.Dialector }

func (d *_SQLiteDialector) DataTypeOf(field *schema.Field) string {
	if field.DataType == "uuid" {
		return "text"
	}
	return d.Dialector.DataTypeOf(field)
}

func (d *_SQLiteDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := d.Dialector.Migrator(db).(sqlite.Migrator)
	m.Dialector = d
	return m
}

func (d *_SQLiteDialector) Initialize(db *gorm.DB) error {
	if err := d.Dialector.Initialize(db); err != nil {
		return err
	}
	return RegisterCreateCallbacks(db)
}

type _PostgresDialector struct{ *postgres.Dialector }

func (d *_PostgresDialector) Initialize(db *gorm.DB) error {
	if err := d.Dialector.Initialize(db); err != nil {
		return err
	}
	return RegisterCreateCallbacks(db)
}
