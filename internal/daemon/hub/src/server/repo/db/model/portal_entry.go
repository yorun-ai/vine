package model

import (
	_ "embed"
	"encoding/json/v2"
	"fmt"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
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
		// Old columns are read only here, then removed after every row has
		// the canonical protocol configuration, including soft-deleted rows.
		if columns["host"] {
			type _LegacyEntry struct {
				Id         int
				Protocol   string
				HTTPConfig string
				Scheme     string
				Host       string
				Port       int
			}
			rows := []_LegacyEntry{}
			if err := tx.Table("portal_entry").Find(&rows).Error; err != nil {
				return err
			}
			for _, row := range rows {
				config := core.PortalEntryHTTP{}
				if row.Protocol == "" || row.HTTPConfig == "" {
					legacy, err := legacyHTTPConfig(row.Scheme, row.Port)
					if err != nil {
						return err
					}
					config = legacy
				} else {
					if row.Protocol != "http" {
						return fmt.Errorf("cannot migrate portal entry protocol %q", row.Protocol)
					}
					if err := json.Unmarshal([]byte(row.HTTPConfig), &config); err != nil {
						return err
					}
				}
				// A nested host already written by a newer version wins,
				// including an explicitly empty host.
				fields := map[string]any{}
				if row.HTTPConfig != "" {
					if err := json.Unmarshal([]byte(row.HTTPConfig), &fields); err != nil {
						return err
					}
				}
				if _, exists := fields["host"]; !exists {
					config.Host = row.Host
				}
				raw, err := json.Marshal(config)
				if err != nil {
					return err
				}
				if err := tx.Table("portal_entry").Where("id = ?", row.Id).Updates(map[string]any{"protocol": "http", "http_config": string(raw)}).Error; err != nil {
					return err
				}
			}
			for _, index := range []string{"uk_portal_entry_access", "uk_portal_entry_protocol_access"} {
				if err := tx.Exec("DROP INDEX IF EXISTS " + index).Error; err != nil {
					return err
				}
			}
			for _, column := range []string{"scheme", "host", "port"} {
				if columns[column] {
					if err := tx.Exec("ALTER TABLE portal_entry DROP COLUMN " + column).Error; err != nil {
						return err
					}
				}
			}
		}
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_protocol_access ON portal_entry(protocol, http_config) WHERE deleted_at IS NULL AND http_config <> ''").Error

	}))
}

func (d *PortalEntryDao) ListOrdered() []*PortalEntry {
	return d.Query().Order("name").Order("id").List()
}

func (d *PortalEntryDao) ById(id int) (*PortalEntry, bool) {
	return d.First("id = ?", id)
}

// ByName returns the user entry Hub labels with the name.
func (d *PortalEntryDao) ByName(name string) (*PortalEntry, bool) {
	return d.First("name = ?", name)
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
func legacyHTTPConfig(scheme string, port int) (core.PortalEntryHTTP, error) {
	if scheme != "http" && scheme != "https" {
		return core.PortalEntryHTTP{}, fmt.Errorf("cannot migrate portal entry scheme %q", scheme)
	}
	if port < 0 || port > 65535 {
		return core.PortalEntryHTTP{}, fmt.Errorf("cannot migrate portal entry port %d", port)
	}
	httpPort, httpsPort := 80, 443
	if scheme == "http" && port != 0 {
		httpPort = port
	}
	if scheme == "https" && port != 0 {
		httpsPort = port
	}
	return core.PortalEntryHTTP{HttpEnabled: scheme == "http", HttpPort: httpPort, HttpsEnabled: scheme == "https", HttpsPort: httpsPort}, nil
}
