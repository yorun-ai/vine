package model

import (
	_ "embed"
	"fmt"

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
	BuiltIn bool   `gorm:"column:built_in;not null;default:false"`
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
	return d.First("name = ? AND built_in = ?", name, false)
}

// ByAccess returns the user entry that serves the access. The built-in
// Dashboard entry is never returned, so user rules cannot join it.
func (d *PortalEntryDao) ByAccess(scheme string, host string, port int) (*PortalEntry, bool) {
	return d.First("scheme = ? AND host = ? AND port = ? AND built_in = ?", scheme, host, port, false)
}

func (d *PortalEntryDao) BuiltIn() (*PortalEntry, bool) {
	return d.First("built_in = ?", true)
}

func (d *PortalEntryDao) Save(entry *PortalEntry) *PortalEntry {
	if entry.Id == 0 {
		d.Create(entry)
		return entry
	}

	row, ok := d.ById(entry.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %d not found", entry.Id))
	d.Update(row, rdb.Patch{
		"name":     entry.Name,
		"scheme":   entry.Scheme,
		"host":     entry.Host,
		"port":     entry.Port,
		"built_in": entry.BuiltIn,
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
	if err := ensurePortalEntryNameColumn(db); err != nil {
		return err
	}
	return db.Exec(schemaSQL(db, createPortalEntrySQLiteSQL, createPortalEntryPgSQL)).Error
}

// _PortalEntryName is one entry name the migration backfills.
type _PortalEntryName struct {
	Id     int
	Scheme string
	Host   string
	Port   int
}

// ensurePortalEntryNameColumn names entries a pre-release Hub stored without a
// name. Hub derives the name from the access, which is the label that Hub
// already showed for those entries.
func ensurePortalEntryNameColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&PortalEntry{}) {
		return nil
	}
	columns, err := tableColumnNames(db, &PortalEntry{})
	if err != nil {
		return err
	}
	if columns["name"] {
		return nil
	}
	if err := db.Exec("ALTER TABLE portal_entry ADD COLUMN name TEXT NOT NULL DEFAULT ''").Error; err != nil {
		return err
	}
	entries := []_PortalEntryName{}
	if err := db.Raw("SELECT id, scheme, host, port FROM portal_entry").Scan(&entries).Error; err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Host
		if name == "" {
			name = fmt.Sprintf("%s:%d", entry.Scheme, entry.Port)
		} else {
			name = fmt.Sprintf("%s:%s:%d", entry.Scheme, name, entry.Port)
		}
		if err := db.Exec("UPDATE portal_entry SET name = ? WHERE id = ?", name, entry.Id).Error; err != nil {
			return err
		}
	}
	return nil
}
