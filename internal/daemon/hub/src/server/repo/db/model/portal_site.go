package model

import (
	_ "embed"
	"sync"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
)

//go:embed sql/sqlite/create_portal_site.sql
var createPortalSiteSQLiteSQL string

//go:embed sql/pgsql/create_portal_site.sql
var createPortalSitePgSQL string

var portalSiteSchemaOnce sync.Once

type PortalSite struct {
	rdb.Model
	Name          string `gorm:"column:name"`
	Type          string `gorm:"column:type"`
	ActorSkelName string `gorm:"column:actor_skel_name"`
	ActorVia      string `gorm:"column:actor_via"`
	CorsMode      string `gorm:"column:cors_mode"`
	CorsOrigins   string `gorm:"column:cors_origins"`
	WebName       string `gorm:"column:web_name"`
	BuiltIn       bool   `gorm:"column:built_in;not null;default:false"`
}

func (*PortalSite) TableName() string {
	return "portal_site"
}

type PortalSiteDao struct {
	rdb.Dao[*PortalSite]
}

func (d *PortalSiteDao) DIInit() {
	d.ensureSchema()
}

func (d *PortalSiteDao) ensureSchema() {
	portalSiteSchemaOnce.Do(func() {
		sql := schemaSQL(d.GormDB(), createPortalSiteSQLiteSQL, createPortalSitePgSQL)
		err := d.GormDB().Exec(sql).Error
		ex.PanicIfError(err)
		d.ensureColumn("cors_mode", "CorsMode")
		d.ensureColumn("cors_origins", "CorsOrigins")
	})
}

func (d *PortalSiteDao) ensureColumn(columnName string, fieldName string) {
	if d.GormDB().Migrator().HasColumn(&PortalSite{}, columnName) {
		return
	}
	ex.PanicIfError(d.GormDB().Migrator().AddColumn(&PortalSite{}, fieldName))
}

func (d *PortalSiteDao) ListOrdered() []*PortalSite {
	return d.Query().Order("name").List()
}

func (d *PortalSiteDao) ByName(name string) (*PortalSite, bool) {
	return d.First("name = ?", name)
}

func (d *PortalSiteDao) ById(id int) (*PortalSite, bool) {
	return d.First("id = ?", id)
}

func (d *PortalSiteDao) Save(entry *PortalSite) *PortalSite {
	if entry.Id == 0 {
		d.Create(entry)
		return entry
	}

	row, ok := d.ById(entry.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", entry.Id))
	d.Update(row, rdb.Patch{
		"name":            entry.Name,
		"type":            entry.Type,
		"actor_skel_name": entry.ActorSkelName,
		"actor_via":       entry.ActorVia,
		"cors_mode":       entry.CorsMode,
		"cors_origins":    entry.CorsOrigins,
		"web_name":        entry.WebName,
		"built_in":        entry.BuiltIn,
	})
	return row
}

func (d *PortalSiteDao) DeleteById(id int) (*PortalSite, bool) {
	row, ok := d.ById(id)
	if !ok {
		return nil, false
	}
	d.Delete(row)
	return row, true
}
