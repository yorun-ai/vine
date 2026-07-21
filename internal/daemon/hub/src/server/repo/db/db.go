package db

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
)

type HubDatabase struct {
	rdb.Database

	Flag *flag.Flag `inject:""`
}

func (d *HubDatabase) InitOption(option *rdb.Option) {
	switch d.Flag.SourceType {
	case flag.SourceSQLite:
		option.ConnURL = "sqlite://" + d.Flag.DBSQLiteFile
	case flag.SourcePostgreSQL:
		option.ConnURL = d.Flag.DBPostgresURL
	}
}

func (*HubDatabase) InitDao(addDao rdb.TypeAdder) {
	addDao(rdb.T[*model.AppConfigDao]())
	addDao(rdb.T[*model.PortalCertDao]())
	addDao(rdb.T[*model.PortalRuleDao]())
	addDao(rdb.T[*model.MetadataDao]())
	addDao(rdb.T[*model.PortalSiteDao]())
}
