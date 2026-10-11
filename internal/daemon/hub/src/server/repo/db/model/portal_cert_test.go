package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

func TestPortalCertDaoCreateAndQuery(t *testing.T) {
	dao := newTestPortalCertDao(t)
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	dao.Create(&PortalCert{
		Name:        "demo-cert",
		Issuer:      "letsencrypt",
		Domains:     `["demo.local","www.demo.local"]`,
		Certificate: "cHVibGlj",
		PrivateKey:  "cHJpdmF0ZQ==",
		ValidFrom:   validFrom,
		ValidTo:     validTo,
	})

	cert, ok := dao.ByName("demo-cert")
	require.True(t, ok)
	assert.Equal(t, "letsencrypt", cert.Issuer)
	assert.Equal(t, `["demo.local","www.demo.local"]`, cert.Domains)
	assert.Equal(t, "cHVibGlj", cert.Certificate)
	assert.Equal(t, "cHJpdmF0ZQ==", cert.PrivateKey)
	assert.True(t, cert.ValidFrom.Equal(validFrom))
	assert.True(t, cert.ValidTo.Equal(validTo))
}

func TestPortalCertDaoListOrdered(t *testing.T) {
	dao := newTestPortalCertDao(t)

	dao.Create(&PortalCert{Name: "z", Domains: `["z.local"]`})
	dao.Create(&PortalCert{Name: "a", Domains: `["a.local"]`})

	certs := dao.ListOrdered()
	require.Len(t, certs, 2)
	assert.Equal(t, "a", certs[0].Name)
	assert.Equal(t, "z", certs[1].Name)
}

func TestPortalCertDaoSaveUpdatesExistingRow(t *testing.T) {
	dao := newTestPortalCertDao(t)
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	cert := dao.Save(&PortalCert{
		Name:        "demo-cert",
		Issuer:      "manual",
		Domains:     `["old.local"]`,
		Certificate: "old-public",
		PrivateKey:  "old-private",
	})
	dao.Save(&PortalCert{
		Id:          cert.Id,
		Name:        "demo-cert",
		Issuer:      "letsencrypt",
		Domains:     `["demo.local"]`,
		Certificate: "new-public",
		PrivateKey:  "new-private",
		ValidFrom:   validFrom,
		ValidTo:     validTo,
	})

	certs := dao.ListOrdered()
	require.Len(t, certs, 1)
	assert.Equal(t, "demo-cert", certs[0].Name)
	assert.Equal(t, "letsencrypt", certs[0].Issuer)
	assert.Equal(t, `["demo.local"]`, certs[0].Domains)
	assert.Equal(t, "new-public", certs[0].Certificate)
	assert.Equal(t, "new-private", certs[0].PrivateKey)
	assert.True(t, certs[0].ValidFrom.Equal(validFrom))
	assert.True(t, certs[0].ValidTo.Equal(validTo))
}

func newTestPortalCertDao(t *testing.T) *PortalCertDao {
	t.Helper()

	db := newModelTestDB(t, "portal-cert.sqlite")
	dao := &PortalCertDao{
		Dao: rdb.NewDao[*PortalCert](db),
	}
	dao.EnsureSchema()
	return dao
}

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
