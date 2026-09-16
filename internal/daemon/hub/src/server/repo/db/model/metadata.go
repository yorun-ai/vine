package model

import (
	_ "embed"

	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/ex"
)

//go:embed sql/sqlite/create_metadata.sql
var createMetadataSQLiteSQL string

//go:embed sql/pgsql/create_metadata.sql
var createMetadataPgSQL string

type Metadata struct {
	rdb.Model
	Name  string `gorm:"column:name"`
	Value string `gorm:"column:value"`
}

func (*Metadata) TableName() string {
	return "metadata"
}

type MetadataDao struct {
	rdb.Dao[*Metadata]
}

func (d *MetadataDao) InitSchema() {
	sql := schemaSQL(d.GormDB(), createMetadataSQLiteSQL, createMetadataPgSQL)
	err := d.GormDB().Exec(sql).Error
	ex.PanicIfError(err)
}

func (d *MetadataDao) ByName(name string) (*Metadata, bool) {
	return d.First("name = ?", name)
}

func (d *MetadataDao) SaveByName(name string, value string) {
	row, ok := d.ByName(name)
	if !ok {
		d.Create(&Metadata{Name: name, Value: value})
		return
	}
	d.Update(row, rdb.Patch{"value": value})
}
