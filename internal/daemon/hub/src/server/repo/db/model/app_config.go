package model

import (
	_ "embed"
	"sync"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/infra/rdb"
)

//go:embed sql/sqlite/create_app_config.sql
var createAppConfigSQLiteSQL string

//go:embed sql/pgsql/create_app_config.sql
var createAppConfigPgSQL string

var configItemSchemaOnce sync.Once

type AppConfig struct {
	rdb.Model
	Name    string `gorm:"column:name"`
	Value   string `gorm:"column:value"`
	Version int    `gorm:"column:version"`
}

func (*AppConfig) TableName() string {
	return "app_config"
}

type AppConfigDao struct {
	rdb.Dao[*AppConfig]
}

func (d *AppConfigDao) DIInit() {
	d.ensureSchema()
}

func (d *AppConfigDao) ensureSchema() {
	configItemSchemaOnce.Do(func() {
		sql := schemaSQL(d.GormDB(), createAppConfigSQLiteSQL, createAppConfigPgSQL)
		err := d.GormDB().Exec(sql).Error
		ex.PanicIfError(err)
	})
}

func (d *AppConfigDao) ListOrdered() []*AppConfig {
	return d.Query().Order("name").Order("version").List()
}

func (d *AppConfigDao) LatestByName(name string) (*AppConfig, bool) {
	return d.Query("name = ?", name).Order("version desc").First()
}

func (d *AppConfigDao) ById(id int) (*AppConfig, bool) {
	return d.First("id = ?", id)
}

func (d *AppConfigDao) Save(item *AppConfig) *AppConfig {
	if item.Id == 0 {
		d.Create(item)
		return item
	}

	row, ok := d.ById(item.Id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("config item %d not found", item.Id))
	d.Update(row, rdb.Patch{
		"name":    item.Name,
		"value":   item.Value,
		"version": item.Version,
	})
	return row
}

func (d *AppConfigDao) DeleteById(id int) (*AppConfig, bool) {
	row, ok := d.ById(id)
	if !ok {
		return nil, false
	}
	d.Delete(row)
	return row, true
}
