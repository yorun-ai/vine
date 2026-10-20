package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

func TestPortalCertMigrationFrom028(t *testing.T) {
	testPortalCertMigrationFrom028(t, newModelTestDB(t, "cert-028.sqlite"), "sqlite")
}

func TestPostgresPortalCertMigrationFrom028(t *testing.T) {
	testPortalCertMigrationFrom028(t, newModelPostgresTestDB(t), "pgsql")
}

func testPortalCertMigrationFrom028(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	loadModel028Schema(t, db, dialect, "portal_cert")
	require.NoError(t, db.Exec(`INSERT INTO portal_cert (name,issuer,domains,certificate,private_key,public_key_base64,private_key_base64,enabled)
 VALUES ('kept','issuer','["example.com"]','current PEM','current key','stale','stale',false)`).Error)
	dao := &PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}
	dao.EnsureSchema()
	columns, err := tableColumnNames(db, "portal_cert")
	require.NoError(t, err)
	require.NotContains(t, columns, "public_key_base64")
	require.NotContains(t, columns, "private_key_base64")
	stored, ok := dao.ByName("kept")
	require.True(t, ok)
	require.NotZero(t, stored.Id)
	require.Equal(t, "current PEM", stored.Certificate)
	require.Equal(t, "current key", stored.PrivateKey)
	require.Equal(t, "issuer", stored.Issuer)
	require.Equal(t, `["example.com"]`, stored.Domains)
	require.False(t, stored.Enabled)
	dao.EnsureSchema()
	again, ok := dao.ById(stored.Id)
	require.True(t, ok)
	require.Equal(t, stored, again)
}
