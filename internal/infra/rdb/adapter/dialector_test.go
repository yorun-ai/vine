package adapter

import (
	"context"
	"os"
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
