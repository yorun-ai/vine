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
	Name       string `gorm:"column:name"`
	Protocol   string `gorm:"column:protocol;not null;default:''"`
	HTTPConfig string `gorm:"column:http_config;not null;default:''"`
	Scheme     string `gorm:"column:scheme"`
	Host       string `gorm:"column:host"`
	Port       int    `gorm:"column:port"`
	ListenIPs  string `gorm:"column:listen_ips;not null;default:'[]'"`
	Enabled    bool   `gorm:"column:enabled;not null"`
}

func (*PortalEntry) TableName() string {
	return "portal_entry"
}

type PortalEntryDao struct {
	rdb.Dao[*PortalEntry]
}

func (d *PortalEntryDao) EnsureSchema() {
	ex.PanicIfError(ensurePortalEntryTable(d.GormDB()))
	ex.PanicIfError(d.GormDB().Transaction(func(tx *gorm.DB) error {
		columns, err := tableColumnNames(tx, "portal_entry")
		if err != nil {
			return err
		}
		if !columns["listen_ips"] {
			if err := tx.Exec("ALTER TABLE portal_entry ADD COLUMN listen_ips TEXT NOT NULL DEFAULT '[]'").Error; err != nil {
				return err
			}
		}
		for _, name := range []string{"protocol", "http_config"} {
			if !columns[name] {
				if err := tx.Exec("ALTER TABLE portal_entry ADD COLUMN " + name + " TEXT NOT NULL DEFAULT ''").Error; err != nil {
					return err
				}
			}
		}
		rows := []*PortalEntry{}
		if err := tx.Unscoped().Where("protocol = '' OR http_config = ''").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			config, err := legacyHTTPConfig(row.Scheme, row.Port)
			if err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&PortalEntry{}).Where("id = ?", row.Id).Updates(map[string]any{"protocol": "http", "http_config": config}).Error; err != nil {
				return err
			}
		}
		if !columns["protocol"] || !columns["http_config"] {
			if err := tx.Exec("DROP INDEX IF EXISTS uk_portal_entry_access").Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_access ON portal_entry(scheme, host, port) WHERE deleted_at IS NULL AND scheme <> ''").Error; err != nil {
			return err
		}
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_protocol_access ON portal_entry(protocol, host, http_config) WHERE deleted_at IS NULL AND http_config <> ''").Error

	}))
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
		"name":        entry.Name,
		"protocol":    entry.Protocol,
		"http_config": entry.HTTPConfig,
		"scheme":      entry.Scheme,
		"host":        entry.Host,
		"port":        entry.Port,
		"listen_ips":  entry.ListenIPs,
		"enabled":     entry.Enabled,
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

// ensurePortalEntryTable creates the entry table and its indexes.
func ensurePortalEntryTable(db *gorm.DB) error {
	return db.Exec(schemaSQL(db, createPortalEntrySQLiteSQL, createPortalEntryPgSQL)).Error
}

// legacyHTTPConfig preserves exactly one enabled transport while upgrading rows.
func legacyHTTPConfig(scheme string, port int) (string, error) {
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("cannot migrate portal entry scheme %q", scheme)
	}
	if port < 0 || port > 65535 {
		return "", fmt.Errorf("cannot migrate portal entry port %d", port)
	}
	httpPort, httpsPort := 80, 443
	if scheme == "http" && port != 0 {
		httpPort = port
	}
	if scheme == "https" && port != 0 {
		httpsPort = port
	}
	return fmt.Sprintf(`{"httpEnabled":%t,"httpPort":%d,"httpsEnabled":%t,"httpsPort":%d,"autoHTTPS":false}`, scheme == "http", httpPort, scheme == "https", httpsPort), nil
}
