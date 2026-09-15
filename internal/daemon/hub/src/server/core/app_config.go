package core

import (
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
)

// Structs

type AppConfig struct {
	FieldSources FieldSources
	Id           int
	CreatedAt    time.Time
	Name         string
	Value        string
	Version      int
	// Definition is the declaration of the application that owns the config, nil
	// when no registered application declares it.
	Definition *AppConfigDefinition
	// Configured reports whether the Hub stores a value for the config. An
	// application can declare a config the Hub has no value for yet.
	Configured bool
	// Status and Lifecycle are derived from the value and the declaration.
	Status    AppConfigStatus
	Lifecycle string
}

type AppConfigCreation struct {
	Name  string
	Value string
}

type AppConfigUpdate struct {
	Value *string
}

// Repo

// AppConfigRepo stores configurations. List and the lookups return entities the
// caller owns, including configurations an application declared without a value.
type AppConfigRepo interface {
	List() []*AppConfig
	// ListSlots returns every configuration the Hub serves: the stored values and
	// the configurations registered applications declare without a value.
	ListSlots() []*AppConfig
	// FindByName returns the configuration with the name, stored or declared.
	FindByName(name string) (*AppConfig, bool)
	GetById(id int) (*AppConfig, bool)
	GetByName(name string) (*AppConfig, bool)
	Save(item *AppConfig)
	Remove(id int) bool
}

// Core

type AppConfigCore struct {
	AppConfigRepo AppConfigRepo `inject:""`
}

func (m *AppConfigCore) List() []*AppConfig {
	return m.AppConfigRepo.ListSlots()
}

// FindByName returns the stored configuration with the name.
func (m *AppConfigCore) FindByName(name string) (*AppConfig, bool) {
	return m.AppConfigRepo.GetByName(name)
}

func (m *AppConfigCore) Get(id int) *AppConfig {
	item, ok := m.AppConfigRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))
	return item
}

// GetSlot returns the configuration with the name, including a configuration an
// application declares without a stored value.
func (m *AppConfigCore) GetSlot(name string) *AppConfig {
	slot, ok := m.AppConfigRepo.FindByName(name)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %q not found", name))
	return slot
}

func (m *AppConfigCore) Create(creation AppConfigCreation) *AppConfig {
	_, ok := m.AppConfigRepo.GetByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("config %q already exists", creation.Name))

	item := &AppConfig{
		Name:      creation.Name,
		Value:     creation.Value,
		Version:   1,
		CreatedAt: time.Now(),
	}
	*item = m.Validate(*item)
	m.AppConfigRepo.Save(item)
	return item
}

func (m *AppConfigCore) Update(id int, update AppConfigUpdate) *AppConfig {
	item, ok := m.AppConfigRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))

	next := &AppConfig{
		FieldSources: cloneFieldSources(item.FieldSources),
		Id:           item.Id,
		CreatedAt:    item.CreatedAt,
		Name:         item.Name,
		Value:        item.Value,
		Version:      item.Version,
	}
	if update.Value != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/value")
		next.Value = *update.Value
	}
	*next = m.Validate(*next)
	if next.Value != item.Value {
		next.Version++
	}

	m.AppConfigRepo.Save(next)
	return next
}

func (m *AppConfigCore) Remove(id int) bool {
	item, ok := m.AppConfigRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config %d not found", id))
	return m.AppConfigRepo.Remove(item.Id)
}

// Validate checks configuration fields without accessing storage.
func (*AppConfigCore) Validate(item AppConfig) AppConfig {
	ex.PanicNewIfNot(strings.TrimSpace(item.Name) != "", ex.OperationFailed, "config name is required")
	return item
}

// Save creates or replaces a configuration by name. Only value changes advance its version.
func (m *AppConfigCore) Save(item AppConfig) *AppConfig {
	item = m.Validate(item)
	item.Id = 0
	item.Version = 1
	item.CreatedAt = time.Now()
	if current, ok := m.AppConfigRepo.GetByName(item.Name); ok {
		item.Id = current.Id
		item.CreatedAt = current.CreatedAt
		item.Version = current.Version
		if item.Value != current.Value {
			item.Version++
		}
	}
	m.AppConfigRepo.Save(&item)
	return &item
}
