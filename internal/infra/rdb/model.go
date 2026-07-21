package rdb

import (
	"gorm.io/gorm"
	"time"
)

// ModelConstraint is the generic model contract used by Dao and Query.
type ModelConstraint interface {
	getId() int
	setId(id int)
}

// Model is gorm.Model plus extra methods
type Model struct {
	Id        int            `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (m *Model) getId() int {
	return m.Id
}

func (m *Model) setId(id int) {
	m.Id = id
}

// DeletableModel can be delete permanently
type DeletableModel struct {
	Id        int       `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (m *DeletableModel) getId() int {
	return m.Id
}

func (m *DeletableModel) setId(id int) {
	m.Id = id
}
