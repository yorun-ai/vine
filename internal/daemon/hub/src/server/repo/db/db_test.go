package db

import (
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/infra/rdb/adapter"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"gorm.io/gorm"
	"testing"
)

func TestNoDBCreatesIsolatedDatabasesAndSchemas(t *testing.T) {
	var previous string
	for range 2 {
		component := &HubDatabase{Flag: &flag.Flag{Store: flag.StoreMemory}}
		option := new(rdb.Option)
		component.InitOption(option)
		require.NotEqual(t, previous, option.ConnURL)
		previous = option.ConnURL
		db, err := gorm.Open(adapter.NewDialector(option.ConnURL), &gorm.Config{})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = pool.Close() })
		config := &model.AppConfigDao{Dao: rdb.NewDao[*model.AppConfig](db)}
		config.InitSchema()
		(&model.PortalSiteDao{Dao: rdb.NewDao[*model.PortalSite](db)}).InitSchema()
		(&model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](db)}).InitSchema()
		(&model.PortalRuleDao{Dao: rdb.NewDao[*model.PortalRule](db)}).InitSchema()
		(&model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](db)}).InitSchema()
		(&model.MetadataDao{Dao: rdb.NewDao[*model.Metadata](db)}).InitSchema()
		require.Empty(t, config.ListOrdered())
		config.Save(&model.AppConfig{Name: "demo.Config", Value: "{}", Version: 1})
		require.Len(t, config.ListOrdered(), 1)
	}
}
