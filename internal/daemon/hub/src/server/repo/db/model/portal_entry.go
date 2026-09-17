package model

import (
	_ "embed"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_portal_entry.sql
var createPortalEntrySQLiteSQL string

//go:embed sql/pgsql/create_portal_entry.sql
var createPortalEntryPgSQL string

// PortalEntry is one Portal access entry. Rules reference the entry instead of
// storing the access themselves, so Hub changes an access once per entry.
type PortalEntry struct {
	rdb.Model
	Name    string `gorm:"column:name"`
	Scheme  string `gorm:"column:scheme"`
	Host    string `gorm:"column:host"`
	Port    int    `gorm:"column:port"`
	Enabled bool   `gorm:"column:enabled;not null"`
}

func (*PortalEntry) TableName() string {
	return "portal_entry"
}

type PortalEntryDao struct {
	rdb.Dao[*PortalEntry]
}

func (d *PortalEntryDao) InitSchema() {
	ex.PanicIfError(ensurePortalEntryTable(d.GormDB()))
}

func (d *PortalEntryDao) ListOrdered() []*PortalEntry {
	return d.Query().Order("port").Order("scheme").Order("host").List()
}

func (d *PortalEntryDao) ById(id int) (*PortalEntry, bool) {
	return d.First("id = ?", id)
}

// ByName returns the user entry Hub labels with the name.
func (d *PortalEntryDao) ByName(name string) (*PortalEntry, bool) {
	return d.First("name = ?", name)
}

// BySchemeHostPort returns the entry that serves the access.
func (d *PortalEntryDao) BySchemeHostPort(scheme string, host string, port int) (*PortalEntry, bool) {
	return d.First("scheme = ? AND host = ? AND port = ?", scheme, host, port)
}

func (d *PortalEntryDao) Save(entry *PortalEntry) *PortalEntry {
	if entry.Id == 0 {
		d.Create(entry)
		return entry
	}

	row, ok := d.ById(entry.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", entry.Id))
	d.Update(row, rdb.Patch{
		"name":    entry.Name,
		"scheme":  entry.Scheme,
		"host":    entry.Host,
		"port":    entry.Port,
		"enabled": entry.Enabled,
	})
	return row
}

func (d *PortalEntryDao) DeleteById(id int) (*PortalEntry, bool) {
	row, ok := d.ById(id)
	if !ok {
		return nil, false
	}
	d.Delete(row)
	return row, true
}

// ensurePortalEntryTable creates the entry table and its indexes. The rule DAO
// calls it before migrating rule access columns into entries.
func ensurePortalEntryTable(db *gorm.DB) error {
	return db.Exec(schemaSQL(db, createPortalEntrySQLiteSQL, createPortalEntryPgSQL)).Error
}
