package model

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_portal_cert.sql
var createPortalCertSQLiteSQL string

//go:embed sql/pgsql/create_portal_cert.sql
var createPortalCertPgSQL string

type PortalCert struct {
	FieldSources string `gorm:"-"`
	rdb.Model
	Certificate string `gorm:"column:certificate"`
	PrivateKey  string `gorm:"column:private_key"`
	Name        string `gorm:"column:name"`
	Issuer      string `gorm:"column:issuer"`
	Domains     string `gorm:"column:domains"`
	// TODO: After the legacy database upgrade window closes, remove both Base64
	// fields and SQL columns together with the PEM migration below.
	//
	// Deprecated: use Certificate. Retained only to migrate existing database records.
	PublicKeyBase64 string `gorm:"column:public_key_base64"`
	// Deprecated: use PrivateKey. Retained only to migrate existing database records.
	PrivateKeyBase64 string    `gorm:"column:private_key_base64"`
	ValidFrom        time.Time `gorm:"column:valid_from"`
	ValidTo          time.Time `gorm:"column:valid_to"`
	Enabled          bool      `gorm:"column:enabled;not null"`
}

func (*PortalCert) TableName() string {
	return "portal_cert"
}

type PortalCertDao struct {
	rdb.Dao[*PortalCert]
}

func (d *PortalCertDao) EnsureSchema() {
	sql := schemaSQL(d.GormDB(), createPortalCertSQLiteSQL, createPortalCertPgSQL)
	err := d.GormDB().Exec(sql).Error
	ex.PanicIfError(err)
	ensureFieldSourceTable(d.GormDB())
	d.migratePEMColumns()
}

func (d *PortalCertDao) ListOrdered() []*PortalCert {
	return d.Query().Order("name").List()
}

func (d *PortalCertDao) ByName(name string) (*PortalCert, bool) {
	return d.First("name = ?", name)
}

func (d *PortalCertDao) ById(id int) (*PortalCert, bool) {
	return d.First("id = ?", id)
}

func (d *PortalCertDao) Save(cert *PortalCert) *PortalCert {
	if cert.Id == 0 {
		d.Create(cert)
		return cert
	}

	row, ok := d.ById(cert.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", cert.Id))
	row.FieldSources = cert.FieldSources
	d.Update(row, rdb.Patch{
		"name":        cert.Name,
		"issuer":      cert.Issuer,
		"domains":     cert.Domains,
		"certificate": cert.Certificate,
		"private_key": cert.PrivateKey,
		"valid_from":  cert.ValidFrom,
		"valid_to":    cert.ValidTo,
		"enabled":     cert.Enabled,
	})
	return row
}

func (d *PortalCertDao) DeleteById(id int) (*PortalCert, bool) {
	row, ok := d.ById(id)
	if !ok {
		return nil, false
	}
	d.Delete(row)
	return row, true
}

func (row *PortalCert) AfterFind(tx *gorm.DB) error {
	return loadFieldSource(tx, "portal_cert", row.Id, &row.FieldSources)
}

func (row *PortalCert) AfterSave(tx *gorm.DB) error {
	return saveFieldSource(tx, "portal_cert", row.Id, row.FieldSources)
}

func (row *PortalCert) AfterDelete(tx *gorm.DB) error {
	return deleteFieldSource(tx, "portal_cert", row.Id)
}

// TODO: After the legacy database upgrade window closes, remove this migration,
// its EnsureSchema call, legacy decoding and provenance helpers, and the migration
// tests and portal_cert_0220 SQL fixtures. Drop the old columns in the same change.
//
// migratePEMColumns upgrades stored certificates before Hub publishes them.
// The transaction includes schema, data and provenance changes; subsequent starts
// leave populated PEM fields intact, even when the legacy columns are stale.
func (d *PortalCertDao) migratePEMColumns() {
	ex.PanicIfError(d.GormDB().Transaction(func(tx *gorm.DB) error {
		columns, err := tableColumnNames(tx, "portal_cert")
		if err != nil {
			return err
		}
		for _, name := range []string{"certificate", "private_key"} {
			if !columns[name] {
				if err := tx.Exec("ALTER TABLE portal_cert ADD COLUMN " + name + " TEXT NOT NULL DEFAULT ''").Error; err != nil {
					return err
				}
			}
		}
		// Read explicit columns, including soft-deleted rows, without entity hooks.
		rows := []PortalCert{}
		if err := tx.Table("portal_cert").Select("id, certificate, private_key, public_key_base64, private_key_base64").
			Where("(certificate = '' AND public_key_base64 <> '') OR (private_key = '' AND private_key_base64 <> '')").Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if row.Certificate == "" {
				row.Certificate, err = legacyCertificatePEM(row.PublicKeyBase64)
				if err != nil {
					return fmt.Errorf("portal certificate %d: cannot migrate certificate to PEM", row.Id)
				}
			}
			if row.PrivateKey == "" {
				row.PrivateKey, err = legacyPrivateKeyPEM(row.PrivateKeyBase64)
				if err != nil {
					return fmt.Errorf("portal certificate %d: cannot migrate private key to PEM", row.Id)
				}
			}
			if _, err := tls.X509KeyPair([]byte(row.Certificate), []byte(row.PrivateKey)); err != nil {
				return fmt.Errorf("portal certificate %d: migrated certificate and private key do not form a valid pair", row.Id)
			}
			if err := tx.Table("portal_cert").Where("id = ?", row.Id).Updates(map[string]any{"certificate": row.Certificate, "private_key": row.PrivateKey}).Error; err != nil {
				return err
			}
		}
		return migratePortalCertSources(tx)
	}))
}

func legacyPEMBytes(value string) ([]byte, error) {
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return []byte(value), nil
	}
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return base64.RawStdEncoding.DecodeString(compact)
	}
	return decoded, nil
}

func legacyCertificatePEM(value string) (string, error) {
	data, err := legacyPEMBytes(value)
	if err != nil {
		return "", err
	}
	if block, _ := pem.Decode(data); block != nil {
		return string(data), nil
	}
	if _, err := x509.ParseCertificate(data); err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})), nil
}

func legacyPrivateKeyPEM(value string) (string, error) {
	data, err := legacyPEMBytes(value)
	if err != nil {
		return "", err
	}
	if block, _ := pem.Decode(data); block != nil {
		return string(data), nil
	}
	kind := "PRIVATE KEY"
	if _, err := x509.ParsePKCS8PrivateKey(data); err != nil {
		if _, err := x509.ParsePKCS1PrivateKey(data); err == nil {
			kind = "RSA PRIVATE KEY"
		} else if _, err := x509.ParseECPrivateKey(data); err == nil {
			kind = "EC PRIVATE KEY"
		} else {
			return "", fmt.Errorf("unsupported private key")
		}
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: data})), nil
}

func migratePortalCertSources(tx *gorm.DB) error {
	rows := []_FieldSource{}
	if err := tx.Where("kind = ?", "portal_cert").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		fields := map[string]jsontext.Value{}
		if err := json.Unmarshal([]byte(row.Fields), &fields); err != nil {
			return err
		}
		changed := false
		for old, next := range map[string]string{"/publicKeyBase64": "/certificate", "/privateKeyBase64": "/privateKey"} {
			if value, ok := fields[old]; ok {
				if _, exists := fields[next]; !exists {
					fields[next] = value
				}
				delete(fields, old)
				changed = true
			}
		}
		if changed {
			encoded, err := json.Marshal(fields)
			if err != nil {
				return err
			}
			if err := saveFieldSource(tx, row.Kind, row.EntityId, string(encoded)); err != nil {
				return err
			}
		}
	}
	return nil
}
