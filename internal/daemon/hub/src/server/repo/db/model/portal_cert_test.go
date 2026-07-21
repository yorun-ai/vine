package model

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/infra/rdb"
	"gorm.io/gorm"
)

var (
	testPortalCertDB     *gorm.DB
	testPortalCertDBOnce sync.Once
)

func TestPortalCertDaoCreateAndQuery(t *testing.T) {
	dao := newTestPortalCertDao(t)
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	dao.Create(&PortalCert{
		Name:             "demo-cert",
		Issuer:           "letsencrypt",
		Domains:          `["demo.local","www.demo.local"]`,
		PublicKeyBase64:  "cHVibGlj",
		PrivateKeyBase64: "cHJpdmF0ZQ==",
		ValidFrom:        validFrom,
		ValidTo:          validTo,
	})

	cert, ok := dao.ByName("demo-cert")
	require.True(t, ok)
	assert.Equal(t, "letsencrypt", cert.Issuer)
	assert.Equal(t, `["demo.local","www.demo.local"]`, cert.Domains)
	assert.Equal(t, "cHVibGlj", cert.PublicKeyBase64)
	assert.Equal(t, "cHJpdmF0ZQ==", cert.PrivateKeyBase64)
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
		Name:             "demo-cert",
		Issuer:           "manual",
		Domains:          `["old.local"]`,
		PublicKeyBase64:  "old-public",
		PrivateKeyBase64: "old-private",
	})
	dao.Save(&PortalCert{
		Model:            rdb.Model{Id: cert.Id},
		Name:             "demo-cert",
		Issuer:           "letsencrypt",
		Domains:          `["demo.local"]`,
		PublicKeyBase64:  "new-public",
		PrivateKeyBase64: "new-private",
		ValidFrom:        validFrom,
		ValidTo:          validTo,
	})

	certs := dao.ListOrdered()
	require.Len(t, certs, 1)
	assert.Equal(t, "demo-cert", certs[0].Name)
	assert.Equal(t, "letsencrypt", certs[0].Issuer)
	assert.Equal(t, `["demo.local"]`, certs[0].Domains)
	assert.Equal(t, "new-public", certs[0].PublicKeyBase64)
	assert.Equal(t, "new-private", certs[0].PrivateKeyBase64)
	assert.True(t, certs[0].ValidFrom.Equal(validFrom))
	assert.True(t, certs[0].ValidTo.Equal(validTo))
}

func newTestPortalCertDao(t *testing.T) *PortalCertDao {
	t.Helper()

	db := sharedTestPortalCertDB(t)
	dao := &PortalCertDao{
		Dao: rdb.NewDao[*PortalCert](db),
	}
	dao.DIInit()
	require.NoError(t, db.Exec("DELETE FROM portal_cert").Error)
	return dao
}

func sharedTestPortalCertDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalCertDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-portal-cert-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal-cert.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalCertDB = db
	})
	return testPortalCertDB
}
