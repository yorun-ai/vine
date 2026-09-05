package seeder

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vfile"
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

	if s.Flag.SeedYAMLPath == "" {
		if !s.MetadataRepo.IsSeeded() {
			s.Logger.Warn("mark hub seed as applied without seed yaml path")
			s.MetadataRepo.MarkSeeded()
		}
		return
	}

	s.loadSeedYAML()
	if s.MetadataRepo.IsSeeded() {
		if s.applyOverrideSeed() {
			s.Logger.Warn("apply override hub seed")
		}
		return
	}

	s.applySeed()
	s.MetadataRepo.MarkSeeded()
	s.Logger.Info("apply hub seed")
}

func (s *Seeder) loadSeedYAML() {
	payload, err := vfile.ReadAsYaml[*_SettingsYAMLPayload](s.Flag.SeedYAMLPath)
	ex.PanicIfError(err)

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

func (s *Seeder) applyOverrideSeed() bool {
	payload, toOverride := s.payload.Overridden()
	if !toOverride {
		return false
	}

	for _, item := range payload.AppConfigs {
		s.AppConfigCore.Save(*item.ToCoreAppConfig())
	}
	for _, site := range payload.PortalEntries {
		s.SiteCore.Save(*site.ToCorePortalSite())
	}
	for _, rule := range payload.PortalRules {
		s.RuleCore.Save(*rule.ToCorePortalRule())
	}
	for _, cert := range payload.PortalCerts {
		s.CertCore.Save(*cert.ToCorePortalCert())
	}

	return true
}
