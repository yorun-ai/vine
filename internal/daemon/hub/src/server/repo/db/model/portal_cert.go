package model

import (
	_ "embed"
	"time"

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
	Name             string    `gorm:"column:name"`
	Issuer           string    `gorm:"column:issuer"`
	Domains          string    `gorm:"column:domains"`
	PublicKeyBase64  string    `gorm:"column:public_key_base64"`
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
	ensureEnabledColumn(d.GormDB(), "portal_cert")
	sql := schemaSQL(d.GormDB(), createPortalCertSQLiteSQL, createPortalCertPgSQL)
	err := d.GormDB().Exec(sql).Error
	ex.PanicIfError(err)
	ensureFieldSourceTable(d.GormDB())
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
		"name":               cert.Name,
		"issuer":             cert.Issuer,
		"domains":            cert.Domains,
		"public_key_base64":  cert.PublicKeyBase64,
		"private_key_base64": cert.PrivateKeyBase64,
		"valid_from":         cert.ValidFrom,
		"valid_to":           cert.ValidTo,
		"enabled":            cert.Enabled,
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
