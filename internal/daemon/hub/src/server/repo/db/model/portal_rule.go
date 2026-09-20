package model

import (
	_ "embed"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_portal_rule.sql
var createPortalRuleSQLiteSQL string

//go:embed sql/pgsql/create_portal_rule.sql
var createPortalRulePgSQL string

type PortalRule struct {
	FieldSources string `gorm:"-"`
	rdb.Model
	Name string `gorm:"column:name"`
	// EntryId refers to the portal_entry that owns the access configuration of
	// the rule.
	EntryId                 int    `gorm:"column:entry_id"`
	MatchPathPrefix         string `gorm:"column:match_path_prefix"`
	RouteType               string `gorm:"column:route_type"`
	RouteSiteName           string `gorm:"column:route_site_name"`
	RouteRedirectionPattern string `gorm:"column:route_redirection_pattern"`
	RoutePathPrefix         string `gorm:"column:route_path_prefix;not null;default:''"`
	Enabled                 bool   `gorm:"column:enabled;not null"`
}

func (*PortalRule) TableName() string {
	return "portal_rule"
}

type PortalRuleDao struct {
	rdb.Dao[*PortalRule]
}

func (d *PortalRuleDao) EnsureSchema() {
	ex.PanicIfError(ensurePortalEntryTable(d.GormDB()))
	dropColumns(d.GormDB(), "portal_rule", "match_scheme", "match_host", "match_port", "built_in")
	sql := schemaSQL(d.GormDB(), createPortalRuleSQLiteSQL, createPortalRulePgSQL)
	ex.PanicIfError(d.GormDB().Exec(sql).Error)
	ensureFieldSourceTable(d.GormDB())
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
	row.FieldSources = rule.FieldSources
	d.Update(row, rdb.Patch{
		"name":                      rule.Name,
		"entry_id":                  rule.EntryId,
		"match_path_prefix":         rule.MatchPathPrefix,
		"route_type":                rule.RouteType,
		"route_site_name":           rule.RouteSiteName,
		"route_redirection_pattern": rule.RouteRedirectionPattern,
		"route_path_prefix":         rule.RoutePathPrefix,
		"enabled":                   rule.Enabled,
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

func (row *PortalRule) AfterFind(tx *gorm.DB) error {
	return loadFieldSource(tx, "portal_rule", row.Id, &row.FieldSources)
}

func (row *PortalRule) AfterSave(tx *gorm.DB) error {
	return saveFieldSource(tx, "portal_rule", row.Id, row.FieldSources)
}

func (row *PortalRule) AfterDelete(tx *gorm.DB) error {
	return deleteFieldSource(tx, "portal_rule", row.Id)
}
