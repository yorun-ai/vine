package adapter

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Opt in with a timezone-free PostgreSQL DSN. This test only runs SELECTs.
func TestPostgresTimestampCompatibility(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	for _, zone := range []string{"", "UTC", "Asia/Shanghai"} {
		name := zone
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			config := dsn
			if zone != "" {
				config += " TimeZone=" + zone
			}
			var timestamps [2]time.Time
			for i, dialect := range []gorm.Dialector{postgres.Open(config), NewDialector(config)} {
				db, err := gorm.Open(dialect, &gorm.Config{})
				require.NoError(t, err)
				pool, err := db.DB()
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, pool.Close()) })
				require.NoError(t, db.Raw("SELECT TIMESTAMP '2026-01-02 12:00:00'").Row().Scan(&timestamps[i]))
				if i == 1 {
					id := uuid.NewV7()
					var got uuid.UUID
					require.NoError(t, db.Raw("SELECT ?::uuid", id).Row().Scan(&got))
					require.Equal(t, id, got)
					stmt, err := pool.PrepareContext(context.Background(), "SELECT $1::uuid, TIMESTAMP '2026-01-02 12:00:00'")
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, stmt.Close()) })
					tx, err := pool.BeginTx(context.Background(), nil)
					require.NoError(t, err)
					defer tx.Rollback()
					var timestamp time.Time
					require.NoError(t, tx.Stmt(stmt).QueryRow(id).Scan(&got, &timestamp))
					require.Equal(t, id, got)
					require.Equal(t, timestamps[i].Format(time.RFC3339), timestamp.Format(time.RFC3339))
					require.NoError(t, tx.Commit())
				}
			}
			require.Equal(t, timestamps[0].Format(time.RFC3339), timestamps[1].Format(time.RFC3339))
			t.Logf("original and adapted timestamp: %s", timestamps[1].Format(time.RFC3339))
		})
	}
}

// These fields intentionally omit UUID type and serializer tags.
type automaticUUIDModel struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	Ref        uuid.UUID
	Optional   *uuid.UUID
	Text       uuid.UUID            `gorm:"type:text"`
	Serialized uuid.UUID            `gorm:"serializer:json"`
	Children   []automaticUUIDChild `gorm:"foreignKey:ParentID"`
}

type automaticUUIDChild struct {
	ID       uuid.UUID `gorm:"primaryKey"`
	ParentID uuid.UUID
}

func TestAutomaticUUIDFields(t *testing.T) {
	for _, backend := range []string{"sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			url := "sqlite://" + t.TempDir() + "/automatic.sqlite"
			if backend == "postgres" {
				url = os.Getenv("VINE_TEST_POSTGRES_DSN")
				if url == "" {
					t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
				}
			}
			db, err := openDriverTestDB(t, url)
			require.NoError(t, err)
			if backend == "postgres" {
				// Roll back both DDL and data in a private schema, including on test failure.
				db = db.Begin()
				require.NoError(t, db.Error)
				t.Cleanup(func() { require.NoError(t, db.Rollback().Error) })
				name := "vine_auto_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
				require.NoError(t, db.Exec(`CREATE SCHEMA "`+name+`"`).Error)
				require.NoError(t, db.Exec(`SET LOCAL search_path TO "`+name+`"`).Error)
			}
			require.NoError(t, db.AutoMigrate(new(automaticUUIDModel), new(automaticUUIDChild)))
			require.NoError(t, db.AutoMigrate(new(automaticUUIDModel), new(automaticUUIDChild)))
			columns, err := db.Migrator().ColumnTypes(new(automaticUUIDModel))
			require.NoError(t, err)
			types := map[string]string{}
			for _, column := range columns {
				types[column.Name()] = column.DatabaseTypeName()
			}
			expected := "text"
			if backend == "postgres" {
				expected = "uuid"
			}
			for _, name := range []string{"id", "ref", "optional"} {
				require.Equal(t, expected, types[name])
			}
			require.Equal(t, "text", types["text"])
			require.Equal(t, "text", types["serialized"])
			id := uuid.NewV7()
			row := new(automaticUUIDModel{Ref: id, Optional: &id, Text: id, Serialized: id, Children: []automaticUUIDChild{{}, {}}})
			require.NoError(t, db.Create(row).Error)
			require.NotEqual(t, uuid.Nil(), row.ID)
			for _, child := range row.Children {
				require.Equal(t, row.ID, child.ParentID)
				require.NotEqual(t, uuid.Nil(), child.ID)
			}
			var loaded automaticUUIDModel
			require.NoError(t, db.Preload("Children", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Where("id = ?", row.ID).First(&loaded).Error)
			require.Equal(t, id, loaded.Ref)
			require.Equal(t, &id, loaded.Optional)
			require.Equal(t, id, loaded.Text)
			require.Equal(t, id, loaded.Serialized)
			require.Len(t, loaded.Children, 2)
			for _, child := range loaded.Children {
				require.Equal(t, row.ID, child.ParentID)
			}
			row.Optional = nil
			require.NoError(t, db.Omit("Children").Save(row).Error)
			require.NoError(t, db.Where("id = ?", row.ID).First(&loaded).Error)
			require.Nil(t, loaded.Optional)
			zero := uuid.Nil()
			row.Optional = &zero
			require.NoError(t, db.Omit("Children").Save(row).Error)
			require.NoError(t, db.Where("id = ?", row.ID).First(&loaded).Error)
			require.Equal(t, &zero, loaded.Optional)
			var ids []uuid.UUID
			require.NoError(t, db.Model(new(automaticUUIDModel)).Pluck("id", &ids).Error)
			require.Equal(t, []uuid.UUID{row.ID}, ids)
		})
	}
}
