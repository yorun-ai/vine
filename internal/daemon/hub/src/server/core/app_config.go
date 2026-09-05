package core

import (
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
)

// Structs

type AppConfig struct {
	Id        int
	CreatedAt time.Time
	Name      string
	Value     string
	Version   int
}

type AppConfigCreation struct {
	Name  string
	Value string
}

type AppConfigUpdate struct {
	Value *string
}

// Repo

type AppConfigRepo interface {
	ListItems() []*AppConfig
	GetItemById(id int) (*AppConfig, bool)
	GetItemByName(name string) (*AppConfig, bool)
	SaveItem(item *AppConfig)
	RemoveItem(id int) bool
}

// Core

type AppConfigCore struct {
	AppConfigRepo AppConfigRepo `inject:""`
}

func (m *AppConfigCore) List() []*AppConfig {
	return m.AppConfigRepo.ListItems()
}

func (m *AppConfigCore) Get(id int) *AppConfig {
	item, ok := m.AppConfigRepo.GetItemById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))
	return item
}

func (m *AppConfigCore) Create(creation AppConfigCreation) *AppConfig {
	_, ok := m.AppConfigRepo.GetItemByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("config %q already exists", creation.Name))

	item := &AppConfig{
		Name:      creation.Name,
		Value:     creation.Value,
		Version:   1,
		CreatedAt: time.Now(),
	}
	*item = m.Validate(*item)
	m.AppConfigRepo.SaveItem(item)
	return item
}

func (m *AppConfigCore) Update(id int, update AppConfigUpdate) *AppConfig {
	item, ok := m.AppConfigRepo.GetItemById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))

	next := &AppConfig{
		Id:        item.Id,
		CreatedAt: item.CreatedAt,
		Name:      item.Name,
		Value:     item.Value,
		Version:   item.Version,
	}
	if update.Value != nil {
		next.Value = *update.Value
	}
	*next = m.Validate(*next)
	if next.Value != item.Value {
		next.Version++
	}

	m.AppConfigRepo.SaveItem(next)
	return next
}

func (m *AppConfigCore) Remove(id int) bool {
	item, ok := m.AppConfigRepo.GetItemById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))
	return m.AppConfigRepo.RemoveItem(item.Id)
}

// Validate checks configuration fields without accessing storage.
func (*AppConfigCore) Validate(item AppConfig) AppConfig {
	ex.PanicNewIfNot(strings.TrimSpace(item.Name) != "", ex.OperationFailed, "config name is required")
	return item
}

// Save creates or replaces a configuration by name. Only value changes advance its version.
func (m *AppConfigCore) Save(item AppConfig) AppConfig {
	item = m.Validate(item)
	item.Id = 0
	item.Version = 1
	item.CreatedAt = time.Now()
	if current, ok := m.AppConfigRepo.GetItemByName(item.Name); ok {
		item.Id = current.Id
		item.CreatedAt = current.CreatedAt
		item.Version = current.Version
		if item.Value != current.Value {
			item.Version++
		}
	}
	m.AppConfigRepo.SaveItem(&item)
	return item
}
