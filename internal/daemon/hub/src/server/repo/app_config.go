package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
)

// DB Repo

type DBAppConfigRepo struct {
	Dao    *model.AppConfigDao `inject:""`
	Syncer *syncer.Syncer      `inject:""`
}

func (s *DBAppConfigRepo) ListItems() []*core.AppConfig {
	rows := s.Dao.ListOrdered()
	items := make([]*core.AppConfig, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAppConfig(row))
	}
	return items
}

func (s *DBAppConfigRepo) GetItemById(id int) (*core.AppConfig, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return mapAppConfig(row), true
	}
	return nil, false
}

func (s *DBAppConfigRepo) GetItemByName(name string) (*core.AppConfig, bool) {
	if row, ok := s.Dao.LatestByName(name); ok {
		return mapAppConfig(row), true
	}
	return nil, false
}

func (s *DBAppConfigRepo) SaveItem(item *core.AppConfig) {
	row := s.Dao.Save(&model.AppConfig{
		Model:   rdb.Model{Id: item.Id},
		Name:    item.Name,
		Value:   item.Value,
		Version: item.Version,
	})
	item.Id = row.Id

	s.Syncer.SyncAppConfig(item)
}

func (s *DBAppConfigRepo) RemoveItem(id int) bool {
	item, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemoveAppConfig(mapAppConfig(item))
	return true
}

func mapAppConfig(row *model.AppConfig) *core.AppConfig {
	return &core.AppConfig{
		Id:        row.Id,
		CreatedAt: row.CreatedAt,
		Name:      row.Name,
		Value:     row.Value,
		Version:   row.Version,
	}
}
