package model

import (
	_ "embed"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
	"gorm.io/gorm"
)

//go:embed sql/sqlite/create_app_config.sql
var createAppConfigSQLiteSQL string

//go:embed sql/pgsql/create_app_config.sql
var createAppConfigPgSQL string

type AppConfig struct {
	FieldSources string `gorm:"-"`
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

func (d *AppConfigDao) EnsureSchema() {
	sql := schemaSQL(d.GormDB(), createAppConfigSQLiteSQL, createAppConfigPgSQL)
	err := d.GormDB().Exec(sql).Error
	ex.PanicIfError(err)
	ensureFieldSourceTable(d.GormDB())
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
	row.FieldSources = item.FieldSources
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

func (row *AppConfig) AfterFind(tx *gorm.DB) error {
	return loadFieldSource(tx, "app_config", row.Id, &row.FieldSources)
}

func (row *AppConfig) AfterSave(tx *gorm.DB) error {
	return saveFieldSource(tx, "app_config", row.Id, row.FieldSources)
}

func (row *AppConfig) AfterDelete(tx *gorm.DB) error {
	return deleteFieldSource(tx, "app_config", row.Id)
}
