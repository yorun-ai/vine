package initializer

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
)

type Initializer struct {
	app.BaseModule

	AppConfigRepo   core.AppConfigRepo      `inject:""`
	PortalEntryRepo core.PortalEntryRepo    `inject:""`
	PortalRuleRepo  core.PortalRuleRepo     `inject:""`
	PortalCertRepo  core.PortalCertRepo     `inject:""`
	PortalSiteRepo  core.PortalSiteRepo     `inject:""`
	SchemaRepo      core.SchemaRepo         `inject:""`
	RegistryCore    *core.RegistryCore      `inject:""`
	Seeder          *seeder.Seeder          `inject:""`
	Syncer          *syncer.Syncer          `inject:""`
	Access          *configaccess.Access    `inject:""`
	InprocFlag      *app.InternalInprocFlag `inject:""`
	Flag            *hubflag.Flag           `inject:""`
	Identity        *mtls.Identity          `inject:""`
}

const (
	inprocSchemaAppName    = "vine.hub.inproc"
	inprocSchemaInstanceId = "registered"
)

func (i *Initializer) DIInit() {
	i.RegistryCore.RegisterSchemas(inprocSchemaAppName, inprocSchemaInstanceId, skel.RegisteredDomainSchemas())

	i.initDashboard()

	domainViews := i.SchemaRepo.ListDomainSchemaViews()
	i.Syncer.SyncSchemas(domainViews)
	for _, item := range i.AppConfigRepo.List() {
		i.Syncer.SyncAppConfig(item)
	}
	// Entries and sites come before rules: Hub decides whether it publishes a rule
	// from the entry it belongs to and the site it targets.
	for _, entry := range i.PortalEntryRepo.List() {
		i.Syncer.SyncPortalEntry(entry)
	}
	for _, site := range i.PortalSiteRepo.List() {
		i.Syncer.SyncPortalSite(site)
	}
	for _, rule := range i.PortalRuleRepo.List() {
		i.Syncer.SyncPortalRule(rule)
	}
	for _, cert := range i.PortalCertRepo.List() {
		i.Syncer.SyncPortalCert(cert)
	}
	if i.Flag.NoDB {
		i.Access.Lock()
	}
}
