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

	AppConfigRepo core.AppConfigRepo  `inject:""`
	MetadataRepo  core.MetadataRepo   `inject:""`
	RuleRepo      core.PortalRuleRepo `inject:""`
	CertRepo      core.PortalCertRepo `inject:""`
	EntryRepo     core.PortalSiteRepo `inject:""`

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

	for _, site := range payload.PortalEntries {
		current, ok := s.EntryRepo.GetEntryByName(site.Name)
		ex.PanicNewIfNot(!ok || !current.BuiltIn,
			ex.OperationFailed,
			ex.F("portal site %q conflicts with built-in site", site.Name))
	}
	for _, rule := range payload.PortalRules {
		current, ok := s.RuleRepo.GetRuleByName(rule.Name)
		ex.PanicNewIfNot(!ok || !current.BuiltIn,
			ex.OperationFailed,
			ex.F("portal rule %q conflicts with built-in rule", rule.Name))
	}

	s.payload = payload
}

func (s *Seeder) applySeed() {
	for _, item := range s.payload.AppConfigs {
		s.AppConfigRepo.SaveItem(item.ToCoreAppConfig(nil))
	}
	for _, site := range s.payload.PortalEntries {
		s.EntryRepo.SaveEntry(site.ToCorePortalSite(nil))
	}
	for _, rule := range s.payload.PortalRules {
		s.RuleRepo.SaveRule(rule.ToCorePortalRule(nil))
	}
	for _, cert := range s.payload.PortalCerts {
		s.CertRepo.SaveCert(cert.ToCorePortalCert(nil))
	}
}

func (s *Seeder) applyOverrideSeed() bool {
	payload, toOverride := s.payload.Overridden()
	if !toOverride {
		return false
	}

	for _, item := range payload.AppConfigs {
		current, _ := s.AppConfigRepo.GetItemByName(item.Name)
		s.AppConfigRepo.SaveItem(item.ToCoreAppConfig(current))
	}
	for _, site := range payload.PortalEntries {
		current, _ := s.EntryRepo.GetEntryByName(site.Name)
		s.EntryRepo.SaveEntry(site.ToCorePortalSite(current))
	}
	for _, rule := range payload.PortalRules {
		current, _ := s.RuleRepo.GetRuleByName(rule.Name)
		s.RuleRepo.SaveRule(rule.ToCorePortalRule(current))
	}
	for _, cert := range payload.PortalCerts {
		current, _ := s.CertRepo.GetCertByName(cert.Name)
		s.CertRepo.SaveCert(cert.ToCorePortalCert(current))
	}

	return true
}
