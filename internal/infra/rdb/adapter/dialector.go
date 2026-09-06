package adapter

import (
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"

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
		return &_SQLiteDialector{Dialector: &sqlite.Dialector{DriverName: "vine-rdb-sqlite", DSN: strings.TrimPrefix(connURL, "sqlite://")}}
	default:
		return &_PostgresDialector{Dialector: postgres.New(postgres.Config{DriverName: "vine-rdb-pgx", DSN: connURL}).(*postgres.Dialector)}
	}
}

// _SQLiteDialector stores UUID columns as text because SQLite has no UUID type.
type _SQLiteDialector struct{ *sqlite.Dialector }

func (d *_SQLiteDialector) DataTypeOf(field *schema.Field) string {
	if field.DataType == "uuid" || automaticUUIDField(field) {
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

// Match GORM's PostgreSQL DSN handling, including timestamp scan locations.
var postgresTimeZoneMatcher = regexp.MustCompile(`(time_zone|TimeZone|timezone)=(.*?)($|&| )`)

func (d *_PostgresDialector) Initialize(db *gorm.DB) error {
	config := *d.Config
	var pool *sql.DB
	if config.Conn == nil {
		pgConfig, err := pgx.ParseConfig(config.DSN)
		if err != nil {
			return err
		}
		if config.PreferSimpleProtocol {
			pgConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		}
		var options []stdlib.OptionOpenDB
		if match := postgresTimeZoneMatcher.FindStringSubmatch(config.DSN); len(match) > 2 {
			zone := match[2]
			pgConfig.RuntimeParams["timezone"] = zone
			options = append(options, stdlib.OptionAfterConnect(func(_ context.Context, conn *pgx.Conn) error {
				location, err := time.LoadLocation(zone)
				if err != nil {
					return err
				}
				conn.TypeMap().RegisterType(&pgtype.Type{
					Name: "timestamp", OID: pgtype.TimestampOID,
					Codec: &pgtype.TimestampCodec{ScanLocation: location},
				})
				return nil
			}))
		}
		pool = sql.OpenDB(&_Connector{
			Connector: stdlib.GetConnector(*pgConfig, options...),
			driver:    &_Driver{Driver: stdlib.GetDefaultDriver()},
		})
		config.Conn = pool
	}
	// Passing Conn retains GORM's callback and dialect initialization while using
	// the fully configured pgx connector behind our UUID parameter wrapper.
	dialect := postgres.New(config)
	if err := dialect.Initialize(db); err != nil {
		if pool != nil {
			_ = pool.Close()
		}
		return err
	}
	if err := RegisterCreateCallbacks(db); err != nil {
		if pool != nil {
			_ = pool.Close()
		}
		return err
	}
	return nil
}

// Native UUID reads are supported by the Go SQL conversion path; writes are
// handled by our driver. Infer only the column type, without mutating cached
// schemas or replacing explicit field serializers.
func automaticUUIDField(field *schema.Field) bool {
	return field.IndirectFieldType == reflect.TypeFor[uuid.UUID]() &&
		field.TagSettings["TYPE"] == "" && field.Serializer == nil
}

func (d *_PostgresDialector) DataTypeOf(field *schema.Field) string {
	if automaticUUIDField(field) {
		return "uuid"
	}
	return d.Dialector.DataTypeOf(field)
}

func (d *_PostgresDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := d.Dialector.Migrator(db).(postgres.Migrator)
	m.Dialector = d
	return m
}
