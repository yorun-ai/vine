package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	configs.EnsureSchema()
	sites.EnsureSchema()
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

type fieldSourceSelectCounter struct {
	logger.Interface
	count int
}

func (c *fieldSourceSelectCounter) Trace(_ context.Context, _ time.Time, query func() (string, int64), _ error) {
	sql, _ := query()
	if strings.HasPrefix(sql, "SELECT ") {
		c.count++
	}
}

func TestListOrderedFieldSources(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) { testListOrderedFieldSources(t, sourceTestDB(t)) })
	t.Run("postgres", func(t *testing.T) { testListOrderedFieldSources(t, newModelPostgresTestDB(t)) })
}

func testListOrderedFieldSources(t *testing.T, db *gorm.DB) {
	counter := &fieldSourceSelectCounter{Interface: logger.Discard}
	db = db.Session(&gorm.Session{Logger: counter})
	configs := &AppConfigDao{Dao: rdb.NewDao[*AppConfig](db)}
	rules := &PortalRuleDao{Dao: rdb.NewDao[*PortalRule](db)}
	sites := &PortalSiteDao{Dao: rdb.NewDao[*PortalSite](db)}
	certs := &PortalCertDao{Dao: rdb.NewDao[*PortalCert](db)}
	configs.EnsureSchema()
	rules.EnsureSchema()
	sites.EnsureSchema()
	certs.EnsureSchema()
	require.NoError(t, db.Create(&_FieldSource{Kind: "other", EntityId: 1, Fields: "other kind"}).Error)

	t.Run("app_config", func(t *testing.T) {
		checkFieldSourceList(t, counter, &configs.Dao, configs.ListOrdered, "version", func(i int, name string, fields string) *AppConfig {
			return &AppConfig{Name: name, Version: i, FieldSources: fields}
		})
	})
	t.Run("portal_rule", func(t *testing.T) {
		checkFieldSourceList(t, counter, &rules.Dao, rules.ListOrdered, "name", func(_ int, name string, fields string) *PortalRule {
			return &PortalRule{Name: name, FieldSources: fields}
		})
	})
	t.Run("portal_site", func(t *testing.T) {
		checkFieldSourceList(t, counter, &sites.Dao, sites.ListOrdered, "name", func(_ int, name string, fields string) *PortalSite {
			return &PortalSite{Name: name, FieldSources: fields}
		})
	})
	t.Run("portal_cert", func(t *testing.T) {
		checkFieldSourceList(t, counter, &certs.Dao, certs.ListOrdered, "name", func(_ int, name string, fields string) *PortalCert {
			return &PortalCert{Name: name, FieldSources: fields}
		})
	})
}

func checkFieldSourceList[T rdb.ModelConstraint](t *testing.T, counter *fieldSourceSelectCounter, dao *rdb.Dao[T], list func() []T, order string, makeRow func(int, string, string) T) {
	t.Helper()
	counter.count = 0
	require.Empty(t, list())
	require.Equal(t, 1, counter.count)

	for i := range 32 {
		fields := ""
		if i%2 == 1 {
			fields = fmt.Sprintf(`{"/name":{"source":"%s/%d"}}`, t.Name(), i)
		}
		dao.Create(makeRow(i, fmt.Sprintf("row-%02d", 31-i), fields))
	}
	dao.Delete(dao.Create(makeRow(32, "deleted", testSourceFields)))

	// The ordinary read still runs AfterFind and serves as the reference for
	// field values, order, and soft deletion across both database backends.
	counter.count = 0
	want := dao.Query().Order(order).List()
	referenceReads := counter.count
	require.Len(t, want, 32)
	counter.count = 0
	require.Equal(t, want, list())
	require.Equal(t, 2, counter.count)
	t.Logf("%d rows: %d individual SELECTs, %d batched SELECTs", len(want), referenceReads, counter.count)
}

func TestFieldSourceDaoSplitsLargeIDLists(t *testing.T) {
	db := sourceTestDB(t)
	ensureFieldSourceTable(db)
	ids := make([]int, fieldSourceBatchSize+1)
	for i := range ids {
		ids[i] = i + 1
	}
	last := ids[len(ids)-1]
	require.NoError(t, db.Create([]_FieldSource{
		{Kind: "app_config", EntityId: 1, Fields: "first"},
		{Kind: "app_config", EntityId: last, Fields: "last"},
		{Kind: "app_config", EntityId: last + 1, Fields: "outside the list"},
	}).Error)
	counter := &fieldSourceSelectCounter{Interface: logger.Discard}
	dao := FieldSourceDao{db: db.Session(&gorm.Session{Logger: counter})}
	require.Empty(t, dao.ListByEntityIDs("app_config", nil))
	require.Zero(t, counter.count)
	require.Equal(t, map[int]string{1: "first", last: "last"}, dao.ListByEntityIDs("app_config", ids))
	require.Equal(t, 2, counter.count)
}
