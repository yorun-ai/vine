package repo

import (
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
)

type AppConfigRepo struct {
	Dao            *model.AppConfigDao  `inject:""`
	DescriptorRepo core.DescriptorRepo  `inject:""`
	Syncer         *syncer.Syncer       `inject:""`
	Access         *configaccess.Access `inject:""`
}

func (s *AppConfigRepo) List() []*core.AppConfig {
	descriptors, enumDescriptors, dataDescriptors := s.DescriptorRepo.ListAppConfigTypeDescriptors()
	return s.listItems(descriptors, enumDescriptors, dataDescriptors)
}

// ListSlots returns the configuration surface: the stored values plus the
// configurations registered applications declare without a value.
func (s *AppConfigRepo) ListSlots() []*core.AppConfig {
	descriptors, enumDescriptors, dataDescriptors := s.DescriptorRepo.ListAppConfigTypeDescriptors()
	slots := s.listItems(descriptors, enumDescriptors, dataDescriptors)
	declared := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		declared[slot.Name] = struct{}{}
	}
	for _, descriptor := range descriptors {
		if _, ok := declared[descriptor.SkelName]; ok {
			continue
		}
		slots = append(slots, s.toCoreAppConfig(&core.AppConfig{
			Name: descriptor.SkelName,
		}, descriptors, enumDescriptors, dataDescriptors))
	}
	return slots
}

func (s *AppConfigRepo) listItems(descriptors []*skeldesc.Config, enumDescriptors []*skeldesc.Enum, dataDescriptors []*skeldesc.Data) []*core.AppConfig {
	rows := s.Dao.ListOrdered()
	items := make([]*core.AppConfig, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.toCoreAppConfig(mapAppConfig(row), descriptors, enumDescriptors, dataDescriptors))
	}
	return items
}

// FindByName returns the configuration with the name, stored or declared.
func (s *AppConfigRepo) FindByName(name string) (*core.AppConfig, bool) {
	descriptors, enumDescriptors, dataDescriptors := s.DescriptorRepo.ListAppConfigTypeDescriptors()
	if row, ok := s.Dao.LatestByName(name); ok {
		return s.toCoreAppConfig(mapAppConfig(row), descriptors, enumDescriptors, dataDescriptors), true
	}
	descriptor := findAppConfigDescriptor(name, descriptors)
	if descriptor == nil {
		return nil, false
	}
	return s.toCoreAppConfig(&core.AppConfig{
		Name: name,
	}, descriptors, enumDescriptors, dataDescriptors), true
}

func (s *AppConfigRepo) GetById(id int) (*core.AppConfig, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return s.resolveAppConfig(mapAppConfig(row)), true
	}
	return nil, false
}

func (s *AppConfigRepo) GetByName(name string) (*core.AppConfig, bool) {
	if row, ok := s.Dao.LatestByName(name); ok {
		return s.resolveAppConfig(mapAppConfig(row)), true
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

	saved := s.resolveAppConfig(mapAppConfig(row))
	*item = *saved
	s.Syncer.SyncAppConfig(saved)
}

func (s *AppConfigRepo) Remove(id int) bool {
	s.Access.CheckWrite()
	item, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemoveAppConfig(s.resolveAppConfig(mapAppConfig(item)))
	return true
}

func (s *AppConfigRepo) resolveAppConfig(item *core.AppConfig) *core.AppConfig {
	descriptors, enums, data := s.DescriptorRepo.ListAppConfigTypeDescriptors()
	return s.toCoreAppConfig(item, descriptors, enums, data)
}

// toCoreAppConfig assembles a configuration slot from its stored value and the
// declaration of the application that owns it.
func (s *AppConfigRepo) toCoreAppConfig(item *core.AppConfig, descriptors []*skeldesc.Config, enumDescriptors []*skeldesc.Enum, dataDescriptors []*skeldesc.Data) *core.AppConfig {
	descriptor := findAppConfigDescriptor(item.Name, descriptors)
	item.Definition = core.NewAppConfigDefinition(descriptor, enumDescriptors, dataDescriptors)
	item.Configured = item.Id != 0
	item.Lifecycle = core.AppConfigLifecycleFor(descriptor)
	if item.Configured {
		item.Status = core.AppConfigStatusFor(descriptor, item.Value, enumDescriptors, dataDescriptors)
	} else {
		item.Status = core.AppConfigStatusUnconfigured
	}
	return item
}

func findAppConfigDescriptor(name string, descriptors []*skeldesc.Config) *skeldesc.Config {
	for _, descriptor := range descriptors {
		if descriptor.SkelName == name {
			return descriptor
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
