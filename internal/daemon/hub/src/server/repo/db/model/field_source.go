package model

import (
	"slices"

	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type _FieldSource struct {
	Kind     string `gorm:"column:kind;primaryKey"`
	EntityId int    `gorm:"column:entity_id;primaryKey;autoIncrement:false"`
	Fields   string `gorm:"column:fields"`
}

func (*_FieldSource) TableName() string { return "field_source" }

// FieldSourceDao reads stored field provenance for Hub entities.
type FieldSourceDao struct {
	db *gorm.DB
}

// Keep both the entity IDs and the kind within SQLite's bind parameter limit.
const fieldSourceBatchSize = 500

// ListByEntityIDs returns the stored fields for the requested IDs of one kind.
// Entities without a source row have no entry in the result.
func (d FieldSourceDao) ListByEntityIDs(kind string, ids []int) map[int]string {
	fieldsByID := make(map[int]string)
	db := d.db.Session(&gorm.Session{NewDB: true})
	for batch := range slices.Chunk(ids, fieldSourceBatchSize) {
		var sources []_FieldSource
		err := db.Select("entity_id", "fields").
			Where("kind = ? AND entity_id IN ?", kind, batch).Find(&sources).Error
		ex.PanicIfError(err)
		for _, source := range sources {
			fieldsByID[source.EntityId] = source.Fields
		}
	}
	return fieldsByID
}

// Load the ordered entities first, then fill their ignored FieldSources fields
// in bulk in place of the single-entity AfterFind hooks.
func listWithFieldSources[T schema.Tabler](query *gorm.DB, fieldTarget func(T) (int, *string)) []T {
	var rows []T
	ex.PanicIfError(query.Session(&gorm.Session{SkipHooks: true}).Find(&rows).Error)
	if len(rows) == 0 {
		return rows
	}

	ids := make([]int, len(rows))
	targets := make([]*string, len(rows))
	for i, row := range rows {
		ids[i], targets[i] = fieldTarget(row)
	}
	sources := FieldSourceDao{db: query}.ListByEntityIDs(rows[0].TableName(), ids)
	for i, id := range ids {
		*targets[i] = sources[id]
	}
	return rows
}

// Each entity initializes the shared source table before its DAO serves reads.
func ensureFieldSourceTable(db *gorm.DB) {
	ex.PanicIfError(db.Exec(`CREATE TABLE IF NOT EXISTS field_source (
  kind TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  fields TEXT NOT NULL,
  PRIMARY KEY (kind, entity_id)
 )`).Error)
}

func loadFieldSource(tx *gorm.DB, kind string, id int, fields *string) error {
	var row _FieldSource
	result := tx.Session(&gorm.Session{NewDB: true}).Where("kind = ? AND entity_id = ?", kind, id).Limit(1).Find(&row)
	*fields = row.Fields
	return result.Error
}

// GORM calls these hooks inside the entity write transaction, so a source write
// failure rolls back the entity mutation as well.
func saveFieldSource(tx *gorm.DB, kind string, id int, fields string) error {
	if fields == "" || fields == "{}" {
		return deleteFieldSource(tx, kind, id)
	}
	return tx.Session(&gorm.Session{NewDB: true}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kind"}, {Name: "entity_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"fields"}),
	}).Create(&_FieldSource{Kind: kind, EntityId: id, Fields: fields}).Error
}

func deleteFieldSource(tx *gorm.DB, kind string, id int) error {
	return tx.Session(&gorm.Session{NewDB: true}).Where("kind = ? AND entity_id = ?", kind, id).Delete(&_FieldSource{}).Error
}
