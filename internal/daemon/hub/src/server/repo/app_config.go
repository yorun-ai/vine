package repo

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
)

type AppConfigRepo struct {
	Dao        *model.AppConfigDao  `inject:""`
	SchemaRepo core.SchemaRepo      `inject:""`
	Syncer     *syncer.Syncer       `inject:""`
	Access     *configaccess.Access `inject:""`
}

func (s *AppConfigRepo) List() []*core.AppConfig {
	schemas := s.schemas()
	enumSchemas := s.SchemaRepo.ListEnumSchemas()
	return s.listItems(schemas, enumSchemas)
}

// ListSlots returns the configuration surface: the stored values plus the
// configurations registered applications declare without a value.
func (s *AppConfigRepo) ListSlots() []*core.AppConfig {
	schemas := s.schemas()
	enumSchemas := s.SchemaRepo.ListEnumSchemas()
	slots := s.listItems(schemas, enumSchemas)
	declared := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		declared[slot.Name] = struct{}{}
	}
	for _, schema := range schemas {
		if _, ok := declared[schema.SkelName]; ok {
			continue
		}
		slots = append(slots, s.toCoreAppConfig(&core.AppConfig{Name: schema.SkelName}, schemas, enumSchemas))
	}
	return slots
}

func (s *AppConfigRepo) listItems(schemas []*skel.ConfigSchema, enumSchemas []*skel.EnumSchema) []*core.AppConfig {
	rows := s.Dao.ListOrdered()
	items := make([]*core.AppConfig, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.toCoreAppConfig(mapAppConfig(row), schemas, enumSchemas))
	}
	return items
}

// FindByName returns the configuration with the name, stored or declared.
func (s *AppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	schemas := s.schemas()
	enumSchemas := s.SchemaRepo.ListEnumSchemas()
	if row, ok := s.Dao.LatestByName(name); ok {
		return s.toCoreAppConfig(mapAppConfig(row), schemas, enumSchemas), true
	}
	schema := findAppConfigSchema(name, schemas)
	if schema == nil {
		return nil, false
	}
	return s.toCoreAppConfig(&core.AppConfig{Name: name}, schemas, enumSchemas), true
}

func (s *AppConfigRepo) GetById(id int) (*core.AppConfig, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return s.toCoreAppConfig(mapAppConfig(row), s.schemas(), s.SchemaRepo.ListEnumSchemas()), true
	}
	return nil, false
}

func (s *AppConfigRepo) GetByName(name string) (*core.AppConfig, bool) {
	if row, ok := s.Dao.LatestByName(name); ok {
		return s.toCoreAppConfig(mapAppConfig(row), s.schemas(), s.SchemaRepo.ListEnumSchemas()), true
	}
	return nil, false
}

func (s *AppConfigRepo) Save(item *core.AppConfig) {
	s.Access.CheckWrite()
	row := s.Dao.Save(&model.AppConfig{
		FieldSources: encodeFieldSources(item.FieldSources),
		Id:           item.Id,
		Name:         item.Name,
		Value:        item.Value,
		Version:      item.Version,
	})

	saved := s.toCoreAppConfig(mapAppConfig(row), s.schemas(), s.SchemaRepo.ListEnumSchemas())
	*item = *saved
	s.Syncer.SyncAppConfig(saved)
}

func (s *AppConfigRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	item, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemoveAppConfig(s.toCoreAppConfig(mapAppConfig(item), s.schemas(), s.SchemaRepo.ListEnumSchemas()))
	return true
}

func (s *AppConfigRepo) schemas() []*skel.ConfigSchema {
	return s.SchemaRepo.ListAppConfigSchemas()
}

// toCoreAppConfig assembles a configuration slot from its stored value and the
// declaration of the application that owns it.
func (s *AppConfigRepo) toCoreAppConfig(item *core.AppConfig, schemas []*skel.ConfigSchema, enumSchemas []*skel.EnumSchema) *core.AppConfig {
	schema := findAppConfigSchema(item.Name, schemas)
	item.Definition = core.NewAppConfigDefinition(schema, enumSchemas)
	item.Configured = item.Id != 0
	item.Lifecycle = core.AppConfigLifecycleFor(schema)
	if item.Configured {
		item.Status = core.AppConfigStatusFor(schema, item.Value, enumSchemas)
	} else {
		item.Status = core.AppConfigStatusUnconfigured
	}
	return item
}

func findAppConfigSchema(name string, schemas []*skel.ConfigSchema) *skel.ConfigSchema {
	for _, schema := range schemas {
		if schema.SkelName == name {
			return schema
		}
	}
	return nil
}

func mapAppConfig(row *model.AppConfig) *core.AppConfig {
	return &core.AppConfig{
		FieldSources: decodeFieldSources(row.FieldSources),
		Id:           row.Id,
		CreatedAt:    row.CreatedAt,
		Name:         row.Name,
		Value:        row.Value,
		Version:      row.Version,
	}
}
