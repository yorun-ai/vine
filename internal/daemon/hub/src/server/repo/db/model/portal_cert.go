package model

import (
	_ "embed"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
)

//go:embed sql/sqlite/create_portal_cert.sql
var createPortalCertSQLiteSQL string

//go:embed sql/pgsql/create_portal_cert.sql
var createPortalCertPgSQL string

var entryCertSchemaOnce sync.Once

type PortalCert struct {
	rdb.Model
	Name             string    `gorm:"column:name"`
	Issuer           string    `gorm:"column:issuer"`
	Domains          string    `gorm:"column:domains"`
	PublicKeyBase64  string    `gorm:"column:public_key_base64"`
	PrivateKeyBase64 string    `gorm:"column:private_key_base64"`
	ValidFrom        time.Time `gorm:"column:valid_from"`
	ValidTo          time.Time `gorm:"column:valid_to"`
}

func (*PortalCert) TableName() string {
	return "portal_cert"
}

type PortalCertDao struct {
	rdb.Dao[*PortalCert]
}

func (d *PortalCertDao) DIInit() {
	d.ensureSchema()
}

func (d *PortalCertDao) ensureSchema() {
	entryCertSchemaOnce.Do(func() {
		sql := schemaSQL(d.GormDB(), createPortalCertSQLiteSQL, createPortalCertPgSQL)
		err := d.GormDB().Exec(sql).Error
		ex.PanicIfError(err)
	})
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
	d.Update(row, rdb.Patch{
		"name":               cert.Name,
		"issuer":             cert.Issuer,
		"domains":            cert.Domains,
		"public_key_base64":  cert.PublicKeyBase64,
		"private_key_base64": cert.PrivateKeyBase64,
		"valid_from":         cert.ValidFrom,
		"valid_to":           cert.ValidTo,
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
