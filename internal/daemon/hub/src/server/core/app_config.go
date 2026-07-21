package core

import (
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
	next.Version++

	m.AppConfigRepo.SaveItem(next)
	return next
}

func (m *AppConfigCore) Remove(id int) bool {
	item, ok := m.AppConfigRepo.GetItemById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))
	return m.AppConfigRepo.RemoveItem(item.Id)
}
