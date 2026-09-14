package seeder

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

type Seeder struct {
	app.BaseModule

	Flag   *flag.Flag     `inject:""`
	Logger *logger.Logger `inject:""`

	MetadataRepo  core.MetadataRepo    `inject:""`
	RuleRepo      core.PortalRuleRepo  `inject:""`
	RuleCore      *core.PortalRuleCore `inject:""`
	AppConfigCore *core.AppConfigCore  `inject:""`
	SiteCore      *core.PortalSiteCore `inject:""`
	CertCore      *core.PortalCertCore `inject:""`

	payload *_SettingsYAMLPayload
}

func (s *Seeder) DIInit() {
	// Keep built-in dashboard entry data current even when user seed has already run.
	s.seedDashboard()

	if s.MetadataRepo.IsSeeded() {
		s.Logger.Info("skip hub seed: all configuration is loaded from the database")
		return
	}

	if s.Flag.SeedHubDataFile == "" && s.Flag.SeedHubData == "" {
		s.Logger.Warn("mark hub seed as applied without seed yaml path")
		s.MetadataRepo.MarkSeeded()
		return
	}

	s.loadSeedYAML()

	s.applySeed()
	s.MetadataRepo.MarkSeeded()
	s.Logger.Info("apply hub seed")
}

func (s *Seeder) loadSeedYAML() {
	template, err := readSeedInput(s.Flag.SeedHubData, s.Flag.SeedHubDataFile)
	ex.PanicIfError(err)
	variables, err := readSeedInput("", s.Flag.SeedHubVarsFile)
	ex.PanicIfError(err)
	source, err := readSeedInput(s.Flag.SeedHubSource, s.Flag.SeedHubSourceFile)
	ex.PanicIfError(err)
	node, sources, err := resolveSeedInput(template, variables, source)
	ex.PanicIfError(err)
	payload := new(_SettingsYAMLPayload)
	ex.PanicIfError(node.Decode(payload))
	for i := range payload.AppConfigs {
		payload.AppConfigs[i].Sources = entityFieldSources(sources, "appConfigs", i)
	}
	for i := range payload.PortalEntries {
		payload.PortalEntries[i].Sources = entityFieldSources(sources, "portalSites", i)
	}
	for i := range payload.PortalRules {
		payload.PortalRules[i].Sources = entityFieldSources(sources, "portalRules", i)
	}
	for i := range payload.PortalCerts {
		payload.PortalCerts[i].Sources = entityFieldSources(sources, "portalCerts", i)
	}

	for _, item := range payload.AppConfigs {
		s.AppConfigCore.Validate(*item.ToCoreAppConfig())
	}
	for _, site := range payload.PortalEntries {
		s.SiteCore.Validate(*site.ToCorePortalSite())
	}
	for _, rule := range payload.PortalRules {
		s.RuleCore.Validate(*rule.ToCorePortalRule())
	}
	for _, cert := range payload.PortalCerts {
		s.CertCore.Validate(*cert.ToCorePortalCert())
	}

	s.payload = payload
}

func (s *Seeder) applySeed() {
	for _, item := range s.payload.AppConfigs {
		s.AppConfigCore.Save(*item.ToCoreAppConfig())
	}
	for _, site := range s.payload.PortalEntries {
		s.SiteCore.Save(*site.ToCorePortalSite())
	}
	for _, rule := range s.payload.PortalRules {
		s.RuleCore.Save(*rule.ToCorePortalRule())
	}
	for _, cert := range s.payload.PortalCerts {
		s.CertCore.Save(*cert.ToCorePortalCert())
	}
}
