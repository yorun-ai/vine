package initializer

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
)

type Initializer struct {
	app.BaseModule

	AppConfigRepo core.AppConfigRepo      `inject:""`
	RuleRepo      core.PortalRuleRepo     `inject:""`
	CertRepo      core.PortalCertRepo     `inject:""`
	EntryRepo     core.PortalSiteRepo     `inject:""`
	SchemaRepo    core.SchemaRepo         `inject:""`
	Seeder        *seeder.Seeder          `inject:""`
	Syncer        *syncer.Syncer          `inject:""`
	InprocFlag    *app.InternalInprocFlag `inject:""`
	Flag          *hubflag.Flag           `inject:""`
}

const (
	inprocSchemaAppName    = "vine.hub.inproc"
	inprocSchemaInstanceId = "registered"
)

func (i *Initializer) DIInit() {
	i.SchemaRepo.SaveDomainSchemas(inprocSchemaAppName, inprocSchemaInstanceId, skel.RegisteredDomainSchemas())

	i.initDashboard()

	domainViews := i.SchemaRepo.ListDomainSchemaViews()
	i.Syncer.SyncSchemas(domainViews)
	for _, item := range i.AppConfigRepo.ListItems() {
		i.Syncer.SyncAppConfig(item)
	}
	for _, rule := range i.RuleRepo.ListRules() {
		i.Syncer.SyncPortalRule(&rule)
	}
	for _, site := range i.EntryRepo.ListEntries() {
		i.Syncer.SyncPortalSiteWithRpcgwServices(&site, core.MatchPortalSiteRpcgwServicesInDomainViews(site, domainViews))
	}
	for _, cert := range i.CertRepo.ListCerts() {
		i.Syncer.SyncPortalCert(cert)
	}
}
