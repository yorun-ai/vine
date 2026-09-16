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

	MetadataRepo  core.MetadataRepo     `inject:""`
	EntryCore     *core.PortalEntryCore `inject:""`
	RuleCore      *core.PortalRuleCore  `inject:""`
	AppConfigCore *core.AppConfigCore   `inject:""`
	SiteCore      *core.PortalSiteCore  `inject:""`
	CertCore      *core.PortalCertCore  `inject:""`

	payload *_SettingsYAMLPayload
}

func (s *Seeder) DIInit() {
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
	ex.PanicIfError(checkSeedRuleStyle(payload))
	for i := range payload.AppConfigs {
		payload.AppConfigs[i].Sources = entityFieldSources(sources, "appConfigs", i)
	}
	for i := range payload.PortalSites {
		payload.PortalSites[i].Sources = entityFieldSources(sources, "portalSites", i)
	}
	for i := range payload.PortalRules {
		payload.PortalRules[i].Sources = entityFieldSources(sources, "portalRules", i)
	}
	for i := range payload.PortalCerts {
		payload.PortalCerts[i].Sources = entityFieldSources(sources, "portalCerts", i)
	}

	entities := payload.entities()
	for _, item := range entities.AppConfigs {
		s.AppConfigCore.Validate(*item)
	}
	for _, site := range entities.PortalSites {
		s.SiteCore.Validate(*site)
	}
	for _, entry := range entities.PortalEntries {
		s.EntryCore.Validate(*entry)
	}
	for _, rule := range entities.PortalRules {
		s.RuleCore.Validate(*rule.Rule)
		if rule.EntryName == "" {
			s.EntryCore.ValidateAccess(rule.Access)
		}
	}
	for _, cert := range entities.PortalCerts {
		s.CertCore.Validate(*cert)
	}

	s.payload = payload
}

func (s *Seeder) applySeed() {
	entities := s.payload.entities()
	for _, item := range entities.AppConfigs {
		s.AppConfigCore.Save(*item)
	}
	for _, site := range entities.PortalSites {
		s.SiteCore.Save(*site)
	}
	// Entries come before rules: a rule joins the entry that serves its access,
	// and an entry the seed named keeps that name.
	for _, entry := range entities.PortalEntries {
		s.EntryCore.Save(*entry)
	}
	for _, rule := range entities.PortalRules {
		s.RuleCore.Save(*resolveSeedRule(s.EntryCore, rule))
	}
	for _, cert := range entities.PortalCerts {
		s.CertCore.Save(*cert)
	}
}
