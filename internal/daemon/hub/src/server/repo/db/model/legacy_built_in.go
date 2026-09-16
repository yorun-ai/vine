package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

// _LegacyBuiltInAccess is one access an earlier release stored on a built-in
// rule.
type _LegacyBuiltInAccess struct {
	Scheme string
	Host   string
	Port   int
}

// removeLegacyBuiltInEntities deletes the entities an earlier release
// provisioned for Hub's own Dashboard, and returns the access those rules
// carried.
//
// Hub serves the Dashboard on the admin module's own listener now, so a stored
// built-in entity would keep publishing routes that Portal cannot serve. The
// built_in column tells Hub that a database still carries them: the work is a
// no-op once the rows are gone, and it never touches an entity an operator
// stores under a name Hub once used.
func removeLegacyBuiltInEntities(db *gorm.DB) []_LegacyBuiltInAccess {
	if !db.Migrator().HasTable("portal_rule") {
		return nil
	}
	columns, err := tableColumnNames(db, "portal_rule")
	ex.PanicIfError(err)
	if !columns["built_in"] {
		return nil
	}

	accesses := []_LegacyBuiltInAccess{}
	if columns["match_scheme"] {
		ex.PanicIfError(db.Raw(
			"SELECT DISTINCT match_scheme AS scheme, match_host AS host, match_port AS port FROM portal_rule WHERE built_in = TRUE",
		).Scan(&accesses).Error)
	}
	// The rules go before the migration groups stored access, so the migration
	// never creates an entry for the access Hub served itself.
	for _, table := range []string{"portal_rule", "portal_site", "portal_entry"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		ex.PanicIfError(db.Exec("DELETE FROM " + table + " WHERE built_in = TRUE").Error)
	}
	return accesses
}

// removeLegacyBuiltInAccessEntries deletes the entry the access migration
// created for an access the built-in rules served: the migration named the entry
// after that access, and no rule references it.
func removeLegacyBuiltInAccessEntries(db *gorm.DB, accesses []_LegacyBuiltInAccess) {
	if len(accesses) == 0 || !db.Migrator().HasTable("portal_entry") {
		return
	}
	for _, access := range accesses {
		ex.PanicIfError(db.Exec(`DELETE FROM portal_entry
			WHERE built_in = FALSE
			  AND scheme = ? AND host = ? AND port = ?
			  AND name = ?
			  AND NOT EXISTS (SELECT 1 FROM portal_rule WHERE portal_rule.entry_id = portal_entry.id)`,
			access.Scheme, access.Host, access.Port,
			portalEntryName(access.Scheme, access.Host, access.Port)).Error)
	}
}
