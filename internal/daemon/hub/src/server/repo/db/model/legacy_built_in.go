package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

// TODO: Delete this file, the built_in column it reads, and the built_in filter
// in migrateAccessGroup once the upgrade window closes: nothing provisions a
// built-in entity any more, so the cleanup only serves a database that predates
// this release.

// removeLegacyBuiltInEntities deletes the entities an earlier release
// provisioned for Hub's own Dashboard.
//
// Hub serves the Dashboard on the admin module's own listener now, so a stored
// built-in entity would keep publishing routes that Portal cannot serve. The
// built_in column tells Hub that a database still carries them: the work is a
// no-op once the rows are gone, and it never touches an entity an operator
// stores under a name Hub once used.
func removeLegacyBuiltInEntities(db *gorm.DB) {
	if !db.Migrator().HasTable("portal_rule") {
		return
	}
	columns, err := tableColumnNames(db, "portal_rule")
	ex.PanicIfError(err)
	if !columns["built_in"] {
		return
	}

	// The rules go before the migration groups stored access, so the migration
	// never creates an entry for the access Hub served itself.
	for _, table := range []string{"portal_rule", "portal_site"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		ex.PanicIfError(db.Exec("DELETE FROM " + table + " WHERE built_in = TRUE").Error)
	}
}
