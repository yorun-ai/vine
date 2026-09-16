package model

import (
	_ "embed"
	"sort"
	"strings"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"gorm.io/gorm"
)

const (
	portalEntryDefaultHTTPPort  = 80
	portalEntryDefaultHTTPSPort = 443
	// portalRuleMigratedSuffix marks the path the access migration moves a rule to
	// when a stored rule keeps the path it declared.
	portalRuleMigratedSuffix = "/migrated"
)

var portalRuleMigrationLogger = logger.New("vine.hub.migration")

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
	BuiltIn                 bool   `gorm:"column:built_in;not null;default:false"`
}

func (*PortalRule) TableName() string {
	return "portal_rule"
}

type PortalRuleDao struct {
	rdb.Dao[*PortalRule]
}

func (d *PortalRuleDao) InitSchema() {
	ex.PanicIfError(ensurePortalEntryTable(d.GormDB()))
	d.migrateAccessColumns()
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

// _LegacyPortalRuleAccess is one access stored on rules before entries existed.
type _LegacyPortalRuleAccess struct {
	Scheme string
	Host   string
	Port   int
}

// _PortalEntryId is the identity the migration reads back after writing an entry.
type _PortalEntryId struct {
	Id int
}

// migrateAccessColumns moves the access stored on every rule into portal_entry.
// Rules used to carry match_scheme, match_host, and match_port, which made an
// access change a rewrite of every rule that used it. The migration groups the
// stored access and gives each group one entry the rules then reference.
func (d *PortalRuleDao) migrateAccessColumns() {
	db := d.GormDB()
	if !db.Migrator().HasTable(&PortalRule{}) {
		return
	}
	columns, err := tableColumnNames(db, &PortalRule{})
	ex.PanicIfError(err)
	if !columns["match_scheme"] {
		return
	}
	if !columns["entry_id"] {
		ex.PanicIfError(db.Migrator().AddColumn(&PortalRule{}, "EntryId"))
	}

	groups := []_LegacyPortalRuleAccess{}
	ex.PanicIfError(db.Raw("SELECT DISTINCT match_scheme AS scheme, match_host AS host, match_port AS port FROM portal_rule").
		Scan(&groups).Error)
	for _, group := range groups {
		entryId := d.migrateAccessGroup(group)
		ex.PanicIfError(db.Exec(
			"UPDATE portal_rule SET entry_id = ? WHERE match_scheme = ? AND match_host = ? AND match_port = ?",
			entryId, group.Scheme, group.Host, group.Port,
		).Error)
	}

	d.resolveMigratedRulePaths()

	ex.PanicIfError(db.Exec("DROP INDEX IF EXISTS uk_portal_rule_match").Error)
	for _, column := range []string{"match_scheme", "match_host", "match_port"} {
		ex.PanicIfError(db.Exec("ALTER TABLE portal_rule DROP COLUMN " + column).Error)
	}
}

// migrateAccessGroup returns the entry that serves one stored access, creating
// it when the access is new.
func (d *PortalRuleDao) migrateAccessGroup(group _LegacyPortalRuleAccess) int {
	db := d.GormDB()
	scheme := strings.ToLower(strings.TrimSpace(group.Scheme))
	host := strings.TrimSpace(group.Host)
	port := group.Port
	// Rules could leave the port unset; entries store the port Portal serves.
	switch {
	case scheme == "http" && port == 0:
		port = portalEntryDefaultHTTPPort
	case scheme == "https" && port == 0:
		port = portalEntryDefaultHTTPSPort
	}

	id := _PortalEntryId{}
	ex.PanicIfError(db.Raw(
		"SELECT id FROM portal_entry WHERE scheme = ? AND host = ? AND port = ? AND built_in = ?",
		scheme, host, port, false,
	).Scan(&id).Error)
	if id.Id != 0 {
		return id.Id
	}

	ex.PanicIfError(db.Exec(
		"INSERT INTO portal_entry (scheme, host, port, built_in, created_at, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		scheme, host, port, false,
	).Error)
	ex.PanicIfError(db.Raw(
		"SELECT id FROM portal_entry WHERE scheme = ? AND host = ? AND port = ? AND built_in = ?",
		scheme, host, port, false,
	).Scan(&id).Error)
	return id.Id
}

// _MigratedPortalRule is one rule whose stored access the migration moves into an
// entry.
type _MigratedPortalRule struct {
	Id              int
	Name            string
	EntryId         int
	MatchPathPrefix string
	MatchPort       int
}

// _PortalRulePath identifies one match path inside one entry.
type _PortalRulePath struct {
	EntryId         int
	MatchPathPrefix string
}

// resolveMigratedRulePaths keeps one rule per entry path. Rules used to be kept
// apart by their stored port, so a rule that left the port unset and the rule
// that named the default port explicitly can land on the same entry path. An
// upgraded database cannot be corrected by editing stored data, so Hub keeps the
// rule with the explicit port on its path and moves the rule that used the
// default port to a /migrated path.
func (d *PortalRuleDao) resolveMigratedRulePaths() {
	rules := []_MigratedPortalRule{}
	ex.PanicIfError(d.GormDB().Raw(
		"SELECT id, name, entry_id, match_path_prefix, match_port FROM portal_rule ORDER BY id",
	).Scan(&rules).Error)

	groups := map[_PortalRulePath][]_MigratedPortalRule{}
	paths := make([]_PortalRulePath, 0, len(rules))
	for _, rule := range rules {
		path := _PortalRulePath{EntryId: rule.EntryId, MatchPathPrefix: rule.MatchPathPrefix}
		if _, ok := groups[path]; !ok {
			paths = append(paths, path)
		}
		groups[path] = append(groups[path], rule)
	}
	sort.Slice(paths, func(a int, b int) bool {
		if paths[a].EntryId != paths[b].EntryId {
			return paths[a].EntryId < paths[b].EntryId
		}
		return paths[a].MatchPathPrefix < paths[b].MatchPathPrefix
	})

	// The explicit rule of every group keeps its path; the rules that used the
	// default port move to a path that is still free.
	taken := map[_PortalRulePath]bool{}
	moved := []_MigratedPortalRule{}
	for _, path := range paths {
		group := groups[path]
		keeper := group[0]
		for _, rule := range group {
			if rule.MatchPort != 0 {
				keeper = rule
				break
			}
		}
		taken[path] = true
		for _, rule := range group {
			if rule.Id != keeper.Id {
				moved = append(moved, rule)
			}
		}
	}
	sort.Slice(moved, func(a int, b int) bool {
		return moved[a].Id < moved[b].Id
	})

	for _, rule := range moved {
		path := rule.MatchPathPrefix
		for {
			path = migratedPortalRulePath(path)
			migrated := _PortalRulePath{EntryId: rule.EntryId, MatchPathPrefix: path}
			if !taken[migrated] {
				taken[migrated] = true
				break
			}
		}
		ex.PanicIfError(d.GormDB().Exec("UPDATE portal_rule SET match_path_prefix = ? WHERE id = ?", path, rule.Id).Error)
		portalRuleMigrationLogger.Warn("portal rule moved to a migrated path",
			"rule", rule.Name, "from", rule.MatchPathPrefix, "to", path)
	}
}

// migratedPortalRulePath returns the path a rule that used the default port moves
// to. Repeating the suffix keeps the path free inside the entry.
func migratedPortalRulePath(matchPathPrefix string) string {
	trimmed := strings.TrimRight(matchPathPrefix, "/")
	if trimmed == "" {
		return portalRuleMigratedSuffix
	}
	return trimmed + portalRuleMigratedSuffix
}

// tableColumnNames returns the stored columns of a model's table.
func tableColumnNames(db *gorm.DB, model any) (map[string]bool, error) {
	columnTypes, err := db.Migrator().ColumnTypes(model)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(columnTypes))
	for _, column := range columnTypes {
		names[column.Name()] = true
	}
	return names, nil
}
