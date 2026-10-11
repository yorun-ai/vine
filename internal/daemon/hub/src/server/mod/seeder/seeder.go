package seeder

import (
	"strings"

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
	node, sources, err := resolveSeedInput(template, variables, source, s.Flag.SeedHubVars...)
	ex.PanicIfError(err)
	payload := new(_SettingsYAMLPayload)
	ex.PanicIfError(node.Decode(payload))
	ex.PanicIfError(validateSeedRuleEntries(payload))
	for i := range payload.AppConfigs {
		payload.AppConfigs[i].Sources = entityFieldSources(sources, "appConfigs", i)
	}
	for i := range payload.PortalSites {
		payload.PortalSites[i].Sources = entityFieldSources(sources, "portalSites", i)
		if payload.PortalSites[i].actorViaPresent {
			s.Logger.Warn("hub seed actorVia is deprecated and ignored; remove this field", "site", payload.PortalSites[i].Name)
		}
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
	certsByName := map[string]*core.PortalCert{}
	for _, cert := range s.CertCore.List() {
		certsByName[cert.Name] = cert
	}
	for _, cert := range entities.PortalCerts {
		validated := s.CertCore.Validate(*cert)
		certsByName[validated.Name] = &validated
	}
	certs := make([]*core.PortalCert, 0, len(certsByName))
	for _, cert := range certsByName {
		certs = append(certs, cert)
	}
	entries := map[string]*core.PortalEntry{}
	listenerEntries := []core.PortalEntry{}
	for _, entry := range entities.PortalEntries {
		normalized := s.EntryCore.Validate(*entry)
		core.ValidatePortalEntryCertificates(normalized, certs)
		_, duplicateName := entries[normalized.Name]
		ex.PanicNewIfNot(!duplicateName, ex.OperationFailed, ex.F("portal entry %q is declared more than once in the seed", normalized.Name))
		// Save rejects duplicate access even for disabled entries or different
		// listener IPs. Check the same normalized identity before any seed writes.
		for _, other := range listenerEntries {
			if normalized.Protocol == other.Protocol && normalized.Http.Equal(*other.Http) {
				ex.PanicNew(ex.OperationFailed, ex.F("portal entry %q duplicates entry %q in the seed", normalized.Name, other.Name))
			}
		}
		entries[normalized.Name] = &normalized
		listenerEntries = append(listenerEntries, normalized)
	}
	sites := map[string]*core.PortalSite{}
	for _, site := range entities.PortalSites {
		sites[site.Name] = site
	}
	for _, rule := range entities.PortalRules {
		s.RuleCore.Validate(*rule.Rule)
		entry := *entries[rule.EntryName]
		if strings.HasPrefix(strings.TrimSpace(entry.Http.Host), "*.") {
			entry = s.EntryCore.Normalize(entry)
			site, ok := sites[rule.Rule.RouteSiteName]
			if !ok {
				site, _ = s.SiteCore.FindByName(rule.Rule.RouteSiteName)
			}
			rule.Rule.ValidateWildcardTarget(entry, site)
		}
	}
	core.ValidatePortalEntryListeners(listenerEntries)

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
	// Certificates come before entries so explicit HTTPS can be validated.
	for _, cert := range entities.PortalCerts {
		s.CertCore.Save(*cert)
	}
	// Entries come before rules: a rule joins the entry that serves its access,
	// and an entry the seed named keeps that name.
	for _, entry := range entities.PortalEntries {
		s.EntryCore.Save(*entry)
	}
	for _, rule := range entities.PortalRules {
		s.RuleCore.Save(*resolveSeedRule(s.EntryCore, rule))
	}
}
