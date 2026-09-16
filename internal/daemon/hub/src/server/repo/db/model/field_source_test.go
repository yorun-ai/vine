package model

import (
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
)

const testSourceFields = `{"/value/enabled":{"source":"app/default","define":"domain/base","override":"app/default"}}`

func sourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	raw, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	return db
}

func TestFieldSourceEntityLifecycle(t *testing.T) {
	for _, tc := range []struct {
		kind, sql string
		row       any
	}{
		{"app_config", createAppConfigSQLiteSQL, &AppConfig{Id: 1, Name: "existing", Value: "{}", Version: 1, FieldSources: testSourceFields}},
		{"portal_rule", createPortalRuleSQLiteSQL, &PortalRule{Id: 1, Name: "existing", FieldSources: testSourceFields}},
		{"portal_site", createPortalSiteSQLiteSQL, &PortalSite{Id: 1, Name: "existing", FieldSources: testSourceFields}},
		{"portal_cert", createPortalCertSQLiteSQL, &PortalCert{Id: 1, Name: "existing", Domains: "[]", FieldSources: testSourceFields}},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			db := sourceTestDB(t)
			require.NoError(t, db.Exec(tc.sql).Error)
			ensureFieldSourceTable(db)
			ensureFieldSourceTable(db)
			require.NoError(t, db.Create(tc.row).Error)
			require.False(t, db.Migrator().HasColumn(tc.kind, "field_sources"))
			var source _FieldSource
			require.NoError(t, db.First(&source, "kind = ? AND entity_id = 1", tc.kind).Error)
			require.Equal(t, testSourceFields, source.Fields)
			loaded := reflect.New(reflect.TypeOf(tc.row).Elem()).Interface()
			require.NoError(t, db.First(loaded, 1).Error)
			require.Equal(t, testSourceFields, reflect.ValueOf(loaded).Elem().FieldByName("FieldSources").String())
			require.NoError(t, db.Model(loaded).Update("name", "renamed").Error)
			require.NoError(t, db.First(&source, "kind = ? AND entity_id = 1", tc.kind).Error)
			require.Equal(t, testSourceFields, source.Fields)
			require.NoError(t, db.Delete(loaded).Error)
			var count int64
			require.NoError(t, db.Model(&_FieldSource{}).Where("kind = ?", tc.kind).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestFieldSourceSaveIsolationAndRollback(t *testing.T) {
	db := sourceTestDB(t)
	configs := &AppConfigDao{Dao: rdb.NewDao[*AppConfig](db)}
	sites := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	configs.InitSchema()
	sites.InitSchema()
	require.False(t, db.Migrator().HasColumn("app_config", "field_sources"))
	config := configs.Save(&AppConfig{Name: "config", Value: "{}", Version: 1, FieldSources: testSourceFields})
	site := sites.Save(&PortalSite{Id: 0, Name: "site", FieldSources: `{"/name":{"source":"domain/site","define":"domain/site"}}`})
	require.Equal(t, config.Id, site.Id)
	config.Name = "renamed"
	config.FieldSources = `{"/value/enabled":{"source":"hub","define":"domain/base","override":"hub"}}`
	configs.Save(config)
	loaded, ok := configs.ById(config.Id)
	require.True(t, ok)
	require.Equal(t, config.FieldSources, loaded.FieldSources)
	loadedSite, ok := sites.ById(site.Id)
	require.True(t, ok)
	require.Equal(t, site.FieldSources, loadedSite.FieldSources)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_source BEFORE INSERT ON field_source BEGIN SELECT RAISE(ABORT, 'source failure'); END`).Error)
	config.Name = "must-rollback"
	require.Panics(t, func() { configs.Save(config) })
	loaded, ok = configs.ById(config.Id)
	require.True(t, ok)
	require.Equal(t, "renamed", loaded.Name)
	require.Panics(t, func() {
		configs.Save(&AppConfig{Name: "failed-create", Value: "{}", Version: 1, FieldSources: testSourceFields})
	})
	_, ok = configs.LatestByName("failed-create")
	require.False(t, ok)
	require.NoError(t, db.Exec("DROP TRIGGER reject_source").Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_source_delete BEFORE DELETE ON field_source BEGIN SELECT RAISE(ABORT, 'source delete failure'); END`).Error)
	require.Panics(t, func() { configs.DeleteById(config.Id) })
	_, ok = configs.ById(config.Id)
	require.True(t, ok)
	require.NoError(t, db.Exec("DROP TRIGGER reject_source_delete").Error)
	config.FieldSources = "{}"
	configs.Save(config)
	loaded, ok = configs.ById(config.Id)
	require.True(t, ok)
	require.Empty(t, loaded.FieldSources)
	loadedSite, ok = sites.ById(site.Id)
	require.True(t, ok)
	require.Equal(t, site.FieldSources, loadedSite.FieldSources)
}
