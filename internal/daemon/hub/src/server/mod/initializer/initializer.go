package initializer

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
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
	RuleCore        *core.PortalRuleCore    `inject:""`
	SchemaRepo      core.SchemaRepo         `inject:""`
	RegistryCore    *core.RegistryCore      `inject:""`
	Seeder          *seeder.Seeder          `inject:""`
	Syncer          *syncer.Syncer          `inject:""`
	Access          *configaccess.Access    `inject:""`
	InprocFlag      *app.InternalInprocFlag `inject:""`
	Flag            *hubflag.Flag           `inject:""`
	Identity        *mtls.Identity          `inject:""`
	Logger          *logger.Logger          `inject:""`
}

const (
	inprocSchemaAppName    = "vine.hub.inproc"
	inprocSchemaInstanceId = "registered"
)

func (i *Initializer) DIInit() {
	i.RegistryCore.RegisterSchemas(inprocSchemaAppName, inprocSchemaInstanceId, skel.RegisteredDomainSchemas())

	i.removeLegacyDashboard()

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
	i.checkPortalRuleConflicts()
	if i.Flag.NoDB {
		i.Access.Lock()
	}
}

// checkPortalRuleConflicts reports rules that match the same request now that the
// schemas decide the prefix of a rule bound to a Web mount path. Hub applies a
// seed before applications register their schemas, so the write that declared the
// rules cannot answer this question. A read-only configuration has no surface an
// operator could fix, so Hub refuses to serve it; a stored configuration keeps
// running and Hub reports what the Dashboard has to resolve.
func (i *Initializer) checkPortalRuleConflicts() {
	conflicts := i.RuleCore.Conflicts()
	if len(conflicts) == 0 {
		return
	}
	first := conflicts[0]
	if i.Flag.NoDB {
		ex.PanicNew(ex.OperationFailed, ex.F(
			"portal rule %q and portal rule %q both match %s: give each request one rule in the seed Hub loads",
			first.Rule, first.Conflict, first.MatchText()))
	}
	for _, conflict := range conflicts {
		i.Logger.Error("two portal rules match the same request",
			"rule", conflict.Rule,
			"conflict", conflict.Conflict,
			"entry", conflict.Access.Name,
			"match", conflict.MatchText())
	}
}
