package model

import (
	_ "embed"
	"sync"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
)

//go:embed sql/sqlite/create_portal_rule.sql
var createPortalRuleSQLiteSQL string

//go:embed sql/pgsql/create_portal_rule.sql
var createPortalRulePgSQL string

var entryRuleSchemaOnce sync.Once

type PortalRule struct {
	rdb.Model
	Name               string `gorm:"column:name"`
	Scheme             string `gorm:"column:scheme"`
	Host               string `gorm:"column:host"`
	Port               int    `gorm:"column:port"`
	PathPrefix         string `gorm:"column:path_prefix"`
	TargetType         string `gorm:"column:target_type"`
	SiteName           string `gorm:"column:site_name"`
	RedirectionPattern string `gorm:"column:redirection_pattern"`
	BuiltIn            bool   `gorm:"column:built_in;not null;default:false"`
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
	entryRuleSchemaOnce.Do(func() {
		sql := schemaSQL(d.GormDB(), createPortalRuleSQLiteSQL, createPortalRulePgSQL)
		err := d.GormDB().Exec(sql).Error
		ex.PanicIfError(err)
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
		"name":                rule.Name,
		"scheme":              rule.Scheme,
		"host":                rule.Host,
		"port":                rule.Port,
		"path_prefix":         rule.PathPrefix,
		"target_type":         rule.TargetType,
		"site_name":           rule.SiteName,
		"redirection_pattern": rule.RedirectionPattern,
		"built_in":            rule.BuiltIn,
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
