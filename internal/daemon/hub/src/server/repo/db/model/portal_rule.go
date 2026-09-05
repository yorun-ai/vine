package model

import (
	_ "embed"
	"sync"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_portal_rule.sql
var createPortalRuleSQLiteSQL string

//go:embed sql/pgsql/create_portal_rule.sql
var createPortalRulePgSQL string

var entryRuleSchemaOnce sync.Once

type PortalRule struct {
	rdb.Model
	Name                    string `gorm:"column:name"`
	MatchScheme             string `gorm:"column:match_scheme"`
	MatchHost               string `gorm:"column:match_host"`
	MatchPort               int    `gorm:"column:match_port"`
	MatchPathPrefix         string `gorm:"column:match_path_prefix"`
	RoutePathPrefix         string `gorm:"column:route_path_prefix;not null;default:''"`
	RouteType               string `gorm:"column:route_type"`
	RouteSiteName           string `gorm:"column:route_site_name"`
	RouteRedirectionPattern string `gorm:"column:route_redirection_pattern"`
	BuiltIn                 bool   `gorm:"column:built_in;not null;default:false"`
}

func (*PortalRule) TableName() string {
	return "portal_rule"
}

type PortalRuleDao struct {
	rdb.Dao[*PortalRule]
}

func (d *PortalRuleDao) DIInit() {
	d.ensureSchema()
}

func (d *PortalRuleDao) ensureSchema() {
	// TODO: Design a unified, versioned migration mechanism tied to database
	// initialization instead of DAO initialization and a process-wide sync.Once.
	entryRuleSchemaOnce.Do(func() {
		ex.PanicIfError(d.migrateSchema())
	})
}

func (d *PortalRuleDao) ListOrdered() []*PortalRule {
	return d.Query().Order("name").List()
}

func (d *PortalRuleDao) ByName(name string) (*PortalRule, bool) {
	return d.First("name = ?", name)
}

func (d *PortalRuleDao) ById(id int) (*PortalRule, bool) {
	return d.First("id = ?", id)
}

func (d *PortalRuleDao) Save(rule *PortalRule) *PortalRule {
	if rule.Id == 0 {
		d.Create(rule)
		return rule
	}

	row, ok := d.ById(rule.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry rule %d not found", rule.Id))
	d.Update(row, rdb.Patch{
		"name":                      rule.Name,
		"match_scheme":              rule.MatchScheme,
		"match_host":                rule.MatchHost,
		"match_port":                rule.MatchPort,
		"match_path_prefix":         rule.MatchPathPrefix,
		"route_path_prefix":         rule.RoutePathPrefix,
		"route_type":                rule.RouteType,
		"route_site_name":           rule.RouteSiteName,
		"route_redirection_pattern": rule.RouteRedirectionPattern,
		"built_in":                  rule.BuiltIn,
	})
	return row
}

func (d *PortalRuleDao) DeleteById(id int) (*PortalRule, bool) {
	row, ok := d.ById(id)
	if !ok {
		return nil, false
	}
	d.Delete(row)
	return row, true
}

// migrateSchema upgrades legacy columns before creating indexes on the new names.
func (d *PortalRuleDao) migrateSchema() error {
	return d.GormDB().Transaction(func(tx *gorm.DB) error {
		migrator := tx.Migrator()
		if migrator.HasTable(&PortalRule{}) {
			for _, columns := range [][2]string{
				{"scheme", "match_scheme"}, {"host", "match_host"}, {"port", "match_port"},
				{"path_prefix", "match_path_prefix"}, {"target_type", "route_type"},
				{"site_name", "route_site_name"}, {"target_path", "route_path_prefix"},
				{"redirection_pattern", "route_redirection_pattern"},
			} {
				if migrator.HasColumn("portal_rule", columns[0]) {
					if err := migrator.RenameColumn("portal_rule", columns[0], columns[1]); err != nil {
						return err
					}
				}
			}
			if !migrator.HasColumn(&PortalRule{}, "route_path_prefix") {
				if err := migrator.AddColumn(&PortalRule{}, "RoutePathPrefix"); err != nil {
					return err
				}
			}
		}
		return tx.Exec(schemaSQL(tx, createPortalRuleSQLiteSQL, createPortalRulePgSQL)).Error
	})
}
