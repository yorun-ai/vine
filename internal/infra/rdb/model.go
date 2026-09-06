package rdb

import (
	"time"
	"uuid"

	"gorm.io/gorm"
)

// Constraint

// ModelConstraint is the generic model contract used by Dao and Query.
type ModelConstraint interface {
	mustBeModel()
}

// Model

// Model provides an integer identifier, timestamps, and soft deletion.
// Prefer UModel for new models; Model remains available for existing integer-key tables.
type Model struct {
	Id        int            `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (*Model) mustBeModel() {}

// DeletableModel provides an integer identifier and timestamps for physical deletion.
// Prefer UDeletableModel for new models that require physical deletion.
type DeletableModel struct {
	Id        int       `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (*DeletableModel) mustBeModel() {}

// UUID Model

// UModel provides a UUIDv7 identifier, timestamps, and soft deletion.
// It is the recommended base for new models.
type UModel struct {
	Id        uuid.UUID      `gorm:"column:id;primaryKey;type:uuid;serializer:uuid"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (*UModel) mustBeModel() {}

// BeforeCreate assigns a UUIDv7 identifier when one has not been supplied.
func (m *UModel) BeforeCreate(_ *gorm.DB) error {
	if m.Id == uuid.Nil() {
		m.Id = uuid.NewV7()
	}
	return nil
}

// UDeletableModel provides a UUIDv7 identifier and timestamps for physical deletion.
// It is the recommended base for new models that require physical deletion.
type UDeletableModel struct {
	Id        uuid.UUID `gorm:"column:id;primaryKey;type:uuid;serializer:uuid"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (*UDeletableModel) mustBeModel() {}

// BeforeCreate assigns a UUIDv7 identifier when one has not been supplied.
func (m *UDeletableModel) BeforeCreate(_ *gorm.DB) error {
	if m.Id == uuid.Nil() {
		m.Id = uuid.NewV7()
	}
	return nil
}
