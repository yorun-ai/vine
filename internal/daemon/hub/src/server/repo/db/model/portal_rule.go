package model

import (
	"database/sql"
	_ "embed"
	"maps"
	"slices"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
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
	ensureLiveNameSchema(d.GormDB(), "portal_rule", createPortalRuleSQLiteSQL, createPortalRulePgSQL)
	ensureFieldSourceTable(d.GormDB())
}

func (d *PortalRuleDao) ListOrdered() []*PortalRule {
	return listWithFieldSources(d.GormDB().Order("name"), func(row *PortalRule) (int, *string) {
		return row.Id, &row.FieldSources
	})
}

// PortalRuleListRows holds the rule list and its associations from one database snapshot.
type PortalRuleListRows struct {
	Rules   []*PortalRule
	Entries []*PortalEntry
	Certs   []*PortalCert
	Sites   []*PortalSite
}

// ListWithTargets reads just the associations used by the listed rules.
// The list does not expose field provenance or need certificate key material.
func (d *PortalRuleDao) ListWithTargets() PortalRuleListRows {
	var rows PortalRuleListRows
	ex.PanicIfError(d.GormDB().Transaction(func(tx *gorm.DB) error {
		db := tx.Session(&gorm.Session{SkipHooks: true})
		err := db.Select("id", "name", "entry_id", "enabled", "match_path_prefix", "route_type",
			"route_site_name", "route_redirection_pattern", "route_path_prefix").
			Order("name").Find(&rows.Rules).Error
		if err != nil || len(rows.Rules) == 0 {
			return err
		}
		entryIDs := make(map[int]struct{})
		siteNames := make(map[string]struct{})
		for _, rule := range rows.Rules {
			entryIDs[rule.EntryId] = struct{}{}
			if rule.RouteType == core.PortalRuleRouteTypeSite {
				siteNames[rule.RouteSiteName] = struct{}{}
			}
		}
		rows.Entries = listRuleTargets[*PortalEntry](db, "id IN ?", entryIDs)
		rows.Sites = listRuleTargets[*PortalSite](db, "name IN ?", siteNames)
		return db.Select("domains", "enabled").Where("enabled = ?", true).Find(&rows.Certs).Error
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead}))
	return rows
}

func listRuleTargets[T any, K comparable](db *gorm.DB, condition string, keys map[K]struct{}) []T {
	var rows []T
	for batch := range slices.Chunk(slices.Collect(maps.Keys(keys)), lookupBatchSize) {
		var targets []T
		ex.PanicIfError(db.Where(condition, batch).Find(&targets).Error)
		rows = append(rows, targets...)
	}
	return rows
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
