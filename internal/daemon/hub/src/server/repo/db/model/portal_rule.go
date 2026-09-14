package model

import (
	_ "embed"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_portal_rule.sql
var createPortalRuleSQLiteSQL string

//go:embed sql/pgsql/create_portal_rule.sql
var createPortalRulePgSQL string

type PortalRule struct {
	FieldSources string `gorm:"-"`
	rdb.Model
	Name                    string `gorm:"column:name"`
	MatchScheme             string `gorm:"column:match_scheme"`
	MatchHost               string `gorm:"column:match_host"`
	MatchPort               int    `gorm:"column:match_port"`
	MatchPathPrefix         string `gorm:"column:match_path_prefix"`
	RouteType               string `gorm:"column:route_type"`
	RouteSiteName           string `gorm:"column:route_site_name"`
	RouteRedirectionPattern string `gorm:"column:route_redirection_pattern"`
	RoutePathPrefix         string `gorm:"column:route_path_prefix;not null;default:''"`
	BuiltIn                 bool   `gorm:"column:built_in;not null;default:false"`
}

func (*PortalRule) TableName() string {
	return "portal_rule"
}

type PortalRuleDao struct {
	rdb.Dao[*PortalRule]
}

func (d *PortalRuleDao) InitSchema() {
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
		"match_scheme":              rule.MatchScheme,
		"match_host":                rule.MatchHost,
		"match_port":                rule.MatchPort,
		"match_path_prefix":         rule.MatchPathPrefix,
		"route_type":                rule.RouteType,
		"route_site_name":           rule.RouteSiteName,
		"route_redirection_pattern": rule.RouteRedirectionPattern,
		"route_path_prefix":         rule.RoutePathPrefix,
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

func (row *PortalRule) AfterFind(tx *gorm.DB) error {
	return loadFieldSource(tx, "portal_rule", row.Id, &row.FieldSources)
}

func (row *PortalRule) AfterSave(tx *gorm.DB) error {
	return saveFieldSource(tx, "portal_rule", row.Id, row.FieldSources)
}

func (row *PortalRule) AfterDelete(tx *gorm.DB) error {
	return deleteFieldSource(tx, "portal_rule", row.Id)
}
