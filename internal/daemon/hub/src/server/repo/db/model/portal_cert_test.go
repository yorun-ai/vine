package model

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/infra/rdb/adapter"
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

	db := sharedTestPortalCertDB(t)
	dao := &PortalCertDao{
		Dao: rdb.NewDao[*PortalCert](db),
	}
	dao.EnsureSchema()
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

func TestSQLitePortalCertPEMMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cert.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	testPortalCertPEMMigration(t, db, "sqlite")
}

func TestPostgresPortalCertPEMMigration(t *testing.T) {
	dsn := os.Getenv("VINE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set VINE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(adapter.NewDialector(dsn), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	schema := "vine_cert_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	testPortalCertPEMMigration(t, tx, "pgsql")
}

func testPortalCertPEMMigration(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	sql, err := os.ReadFile("testdata/" + dialect + "/portal_cert_0220.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(sql)).Error)
	ensureFieldSourceTable(db)
	certPEM, keyPEM, certDER, keyDER := migrationTestPair(t)
	chain := certPEM + certPEM
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, input := range [][2]string{
		{base64.StdEncoding.EncodeToString([]byte(chain)), base64.StdEncoding.EncodeToString([]byte(keyPEM))},
		{certPEM, keyPEM},
		{base64.RawStdEncoding.EncodeToString(certDER), base64.StdEncoding.EncodeToString(keyDER)},
	} {
		id := i + 1
		require.NoError(t, db.Exec(`INSERT INTO portal_cert (id, name, issuer, domains, public_key_base64, private_key_base64, enabled, valid_from, valid_to) VALUES (?, ?, 'issuer', '["demo.local"]', ?, ?, FALSE, ?, ?)`, id, "cert"+string(rune('a'+i)), input[0], input[1], from, from.AddDate(1, 0, 0)).Error)
	}
	// Include soft-deleted rows, and ensure provenance survives under new names.
	require.NoError(t, db.Exec("UPDATE portal_cert SET deleted_at = ? WHERE id = 3", from).Error)
	require.NoError(t, saveFieldSource(db, "portal_cert", 1, `{"/publicKeyBase64":{"source":"seed"},"/privateKeyBase64":{"source":"secret"}}`))
	dao := &PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}
	dao.EnsureSchema()
	row, ok := dao.ById(1)
	require.True(t, ok)
	require.Equal(t, chain, row.Certificate)
	require.Equal(t, keyPEM, row.PrivateKey)
	require.False(t, row.Enabled)
	require.True(t, from.Equal(row.ValidFrom))
	require.Equal(t, "issuer", row.Issuer)
	require.Equal(t, `["demo.local"]`, row.Domains)
	require.Contains(t, row.FieldSources, `"/certificate"`)
	require.Contains(t, row.FieldSources, `"/privateKey"`)
	require.NotContains(t, row.FieldSources, "Base64")
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(chain)), row.PublicKeyBase64)
	for id := 1; id <= 3; id++ {
		var migrated struct {
			Certificate string
			PrivateKey  string
		}
		require.NoError(t, db.Table("portal_cert").Select("certificate, private_key").Where("id = ?", id).Scan(&migrated).Error)
		_, err := tls.X509KeyPair([]byte(migrated.Certificate), []byte(migrated.PrivateKey))
		require.NoError(t, err)
	}
	// Changed PEM remains authoritative over stale legacy data on restart.
	row.Certificate = certPEM
	dao.Save(row)
	require.NoError(t, db.Exec("UPDATE portal_cert SET public_key_base64 = 'stale', private_key_base64 = 'stale' WHERE id = 1").Error)
	dao.EnsureSchema()
	row, ok = dao.ById(1)
	require.True(t, ok)
	require.Equal(t, certPEM, row.Certificate)
	require.Equal(t, keyPEM, row.PrivateKey)
}

func TestPortalCertPEMMigrationRollsBackInvalidData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "invalid.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	sql, err := os.ReadFile("testdata/sqlite/portal_cert_0220.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(sql)).Error)
	cert, key, _, _ := migrationTestPair(t)
	for id, value := range map[int]string{1: base64.StdEncoding.EncodeToString([]byte(cert)), 2: "invalid"} {
		require.NoError(t, db.Exec(`INSERT INTO portal_cert (id,name,issuer,domains,public_key_base64,private_key_base64) VALUES (?,?,'','[]',?,?)`, id, "cert"+string(rune('a'+id)), value, base64.StdEncoding.EncodeToString([]byte(key))).Error)
	}
	dao := &PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}
	require.Panics(t, dao.EnsureSchema)
	columns, err := tableColumnNames(db, "portal_cert")
	require.NoError(t, err)
	require.NotContains(t, columns, "certificate")
	require.NotContains(t, columns, "private_key")
}

func migrationTestPair(t *testing.T) (string, string, []byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"demo.local"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})), der, keyDER
}
