package model

import (
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type _FieldSource struct {
	Kind     string `gorm:"column:kind;primaryKey"`
	EntityId int    `gorm:"column:entity_id;primaryKey;autoIncrement:false"`
	Fields   string `gorm:"column:fields"`
}

func (*_FieldSource) TableName() string { return "field_source" }

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
