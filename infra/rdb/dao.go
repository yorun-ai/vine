package rdb

import (
	"go.yorun.ai/vine/core/ex"
	"go.yorun.ai/vine/infra/rdb/adapter"
	"go.yorun.ai/vine/util/vpre"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type _GormDBSetter interface {
	setGormDB(*gorm.DB)
}

// Dao provides typed create, read, update, and delete operations for M.
type Dao[M ModelConstraint] struct {
	gormDB *gorm.DB
}

// Patch maps GORM field or column names to update values.
// Nil and zero values are written explicitly; GORM expressions are supported.
type Patch map[string]any

// NewDao creates a typed data access object backed by gdb and registers UUID generation.
// Initialize DAOs before using an externally managed connection concurrently.
func NewDao[M ModelConstraint](gdb *gorm.DB) Dao[M] {
	ex.PanicIfError(adapter.RegisterCreateCallbacks(gdb))
	return Dao[M]{
		gormDB: gdb,
	}
}

func (d *Dao[M]) setGormDB(gdb *gorm.DB) {
	d.gormDB = gdb
}

// GormDB returns the connection bound to this DAO, including its context and logger.
// Use it for transactions or operations not covered by the typed API.
func (d *Dao[M]) GormDB() *gorm.DB {
	return d.gormDB
}

// EnsureSchema initializes the schema owned by this DAO. The default
// implementation does nothing; a concrete DAO may override it.
func (d *Dao[M]) EnsureSchema() {}

// Query builds a read query without executing it.
// Conditions follow GORM query forms, with UUID primary keys and map values normalized.
func (d *Dao[M]) Query(conditions ...any) *Query[M] {
	return &Query[M]{
		gormDB:     d.gormDB,
		conditions: adapter.NormalizeConditions(conditions),
	}
}

// Filter selects records for conditional updates and deletion without loading them.
// Conditions use the same forms and UUID normalization as Query.
// Filter panics when no conditions are supplied.
func (d *Dao[M]) Filter(conditions ...any) *Filtered[M] {
	vpre.Check(len(conditions) > 0, "filter requires conditions")
	return new(Filtered[M]{
		gormDB:     d.gormDB,
		conditions: adapter.NormalizeConditions(conditions),
	})
}

// One selects records for a write that must affect exactly one row.
// It accepts the same conditions as Filter and panics when none are supplied.
// Update and Delete run in a transaction, return 1 on success, and roll back
// and panic when the affected row count differs from 1.
func (d *Dao[M]) One(conditions ...any) *Filtered[M] {
	filtered := d.Filter(conditions...)
	filtered.one = true
	return filtered
}

// First returns the first matching model and true, or the zero value and false
// when no record matches. Database failures panic.
func (d *Dao[M]) First(conditions ...any) (M, bool) {
	return d.Query(conditions...).First()
}

// Exists reports whether a matching record exists without loading a model.
// Conditions use the same forms and UUID normalization as Query.
// Soft-deleted records are excluded by default. Database failures panic.
func (d *Dao[M]) Exists(conditions ...any) bool {
	return d.Query(conditions...).Exists()
}

// List returns all matching models. An empty result may be nil.
// Database failures panic.
func (d *Dao[M]) List(conditions ...any) []M {
	return d.Query(conditions...).List()
}

// Create inserts model and returns the candidate model. Database failures panic.
// With OnConflict DoNothing, it may retain a generated ID even when no row was
// inserted. Use GORM directly and inspect RowsAffected when the insertion outcome
// is required.
func (d *Dao[M]) Create(model M) M {
	result := d.gormDB.Clauses(clause.Returning{}).Create(model)
	ex.PanicIfError(result.Error)
	return model
}

// Update applies patch to model and returns the same model.
// Patch preserves nil and zero values and supports GORM expressions.
// Database failures panic; use Filter to update by conditions and inspect affected rows.
func (d *Dao[M]) Update(model M, patch Patch) M {
	patchMap := map[string]any(patch)
	result := d.gormDB.Model(model).Clauses(clause.Returning{}).Updates(patchMap)
	ex.PanicIfError(result.Error)
	return model
}

// Delete deletes model according to its GORM primary key and scopes.
// Models with DeletedAt use soft deletion; other models are physically deleted.
// Database failures panic.
func (d *Dao[M]) Delete(model M) {
	result := d.gormDB.Delete(&model)
	ex.PanicIfError(result.Error)
}
