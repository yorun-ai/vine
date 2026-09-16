package admin

import (
	"strconv"

	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
)

const (
	seedKindAppConfig   = "app_config"
	seedKindPortalSite  = "portal_site"
	seedKindPortalEntry = "portal_entry"
	seedKindPortalRule  = "portal_rule"
	seedKindPortalCert  = "portal_cert"
)

type MaintenanceApiServiceServerImpl struct {
	skeled.DefaultMaintenanceApiServiceServer

	AppConfigCore *core.AppConfigCore   `inject:""`
	SiteCore      *core.PortalSiteCore  `inject:""`
	EntryCore     *core.PortalEntryCore `inject:""`
	RuleCore      *core.PortalRuleCore  `inject:""`
	CertCore      *core.PortalCertCore  `inject:""`
	Access        *configaccess.Access  `inject:""`
}

// The fields each seed entity kind compares, in the order the Dashboard shows
// them. Both sides of the comparison use Core SeedFields, so the preview cannot
// drift from the values a seed applies.
var (
	seedAppConfigFields   = []string{"value"}
	seedPortalSiteFields  = []string{"type", "actorSkelName", "actorVia", "corsMode", "corsOrigins", "webName", "disabled"}
	seedPortalEntryFields = []string{"scheme", "host", "port", "disabled"}
	seedPortalRuleFields  = []string{
		"matchScheme", "matchHost", "matchPort", "matchPathPrefix",
		"routeType", "routeSiteName", "routeRedirectionPattern", "routePathPrefix", "disabled",
	}
	seedPortalCertFields = []string{
		"issuer", "domains", "publicKeyBase64", "privateKeyBase64", "validFrom", "validTo", "disabled",
	}
)

type _SeedSelectionKey struct {
	kind string
	name string
}

func (s *MaintenanceApiServiceServerImpl) PreviewSeedYaml(content string) skeled.SeedPreview {
	return s.preview(s.parseSeed(content))
}

func (s *MaintenanceApiServiceServerImpl) ApplySeedYaml(content string, selections []skeled.SeedItemSelection) skeled.SeedPreview {
	entities := s.parseSeed(content)
	if entities == nil {
		return newSeedPreview()
	}

	selected := map[_SeedSelectionKey]struct{}{}
	for _, selection := range selections {
		selected[_SeedSelectionKey{
			kind: selection.Kind,
			name: selection.Name,
		}] = struct{}{}
	}

	s.applyAppConfigs(entities.AppConfigs, selected)
	s.applyPortalSites(entities.PortalSites, selected)
	// Entries come before rules: a rule joins the entry that serves its access.
	s.applyPortalEntries(entities.PortalEntries, selected)
	s.applyPortalRules(entities.PortalRules, selected)
	s.applyPortalCerts(entities.PortalCerts, selected)
	return s.preview(entities)
}

// parseSeed reads the seed contract Seeder owns and validates every declared
// entity, so an invalid document fails before the Dashboard writes anything.
func (s *MaintenanceApiServiceServerImpl) parseSeed(content string) *seeder.SeedEntities {
	entities, err := seeder.ParseSeedEntities(content)
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, ex.F("parse seed yaml failed: %v", err))
	if entities == nil {
		return nil
	}
	for i, item := range entities.AppConfigs {
		validated := s.AppConfigCore.Validate(*item)
		entities.AppConfigs[i] = &validated
	}
	for i, site := range entities.PortalSites {
		validated := s.SiteCore.Validate(*site)
		entities.PortalSites[i] = &validated
	}
	for i, entry := range entities.PortalEntries {
		validated := s.EntryCore.Validate(*entry)
		entities.PortalEntries[i] = &validated
	}
	for _, rule := range entities.PortalRules {
		*rule.Rule = s.RuleCore.Validate(*rule.Rule)
		if rule.EntryName == "" {
			s.EntryCore.ValidateAccess(rule.Access)
		}
	}
	for i, cert := range entities.PortalCerts {
		validated := s.CertCore.Validate(*cert)
		entities.PortalCerts[i] = &validated
	}
	return entities
}

func (s *MaintenanceApiServiceServerImpl) preview(entities *seeder.SeedEntities) skeled.SeedPreview {
	if entities == nil {
		return newSeedPreview()
	}

	preview := newSeedPreview()
	for _, item := range entities.AppConfigs {
		preview.Items = append(preview.Items, s.previewAppConfig(item))
	}
	for _, site := range entities.PortalSites {
		preview.Items = append(preview.Items, s.previewPortalSite(site))
	}
	for _, entry := range entities.PortalEntries {
		preview.Items = append(preview.Items, s.previewPortalEntry(entry))
	}
	seedEntries := make(map[string]*core.PortalEntry, len(entities.PortalEntries))
	for _, entry := range entities.PortalEntries {
		seedEntries[entry.Name] = entry
	}
	for _, rule := range entities.PortalRules {
		preview.Items = append(preview.Items, s.previewPortalRule(rule, seedEntries))
	}
	for _, cert := range entities.PortalCerts {
		preview.Items = append(preview.Items, s.previewPortalCert(cert))
	}
	return preview
}

func newSeedPreview() skeled.SeedPreview {
	return skeled.SeedPreview{Items: make([]skeled.SeedEntityDiff, 0)}
}

func (s *MaintenanceApiServiceServerImpl) previewAppConfig(item *core.AppConfig) skeled.SeedEntityDiff {
	current, exists := s.AppConfigCore.FindByName(item.Name)
	return seedEntityDiff(seedKindAppConfig, item.Name, exists, current.SeedFields(), item.SeedFields(), seedAppConfigFields)
}

func (s *MaintenanceApiServiceServerImpl) previewPortalSite(site *core.PortalSite) skeled.SeedEntityDiff {
	current, exists := s.SiteCore.FindByName(site.Name)
	return seedEntityDiff(seedKindPortalSite, site.Name, exists, current.SeedFields(), site.SeedFields(), seedPortalSiteFields)
}

func (s *MaintenanceApiServiceServerImpl) previewPortalEntry(entry *core.PortalEntry) skeled.SeedEntityDiff {
	current, exists := s.EntryCore.FindByName(entry.Name)
	return seedEntityDiff(seedKindPortalEntry, entry.Name, exists, current.SeedFields(), entry.SeedFields(), seedPortalEntryFields)
}

// previewPortalRule compares a declared rule with the stored one. A rule carries
// no access of its own, so the preview compares the access of the entry each side
// joins as well: the values Portal matches are what an operator changes.
func (s *MaintenanceApiServiceServerImpl) previewPortalRule(rule *seeder.SeedRule, seedEntries map[string]*core.PortalEntry) skeled.SeedEntityDiff {
	current, exists := s.RuleCore.FindByName(rule.Rule.Name)
	declared := rule.Rule.SeedFields()
	if entry := s.seedRuleEntry(rule, seedEntries); entry != nil {
		addPortalRuleAccess(declared, entry)
	}
	currentFields := map[string]string{}
	if current != nil {
		currentFields = current.SeedFields()
		if entry, ok := s.EntryCore.FindById(current.EntryId); ok {
			addPortalRuleAccess(currentFields, entry)
		}
	}
	return seedEntityDiff(seedKindPortalRule, rule.Rule.Name, exists, currentFields, declared, seedPortalRuleFields)
}

// addPortalRuleAccess adds the access a rule matches to the fields a seed
// compares, under the field names a rule declares it with.
func addPortalRuleAccess(fields map[string]string, entry *core.PortalEntry) {
	fields["matchScheme"] = entry.Scheme
	fields["matchHost"] = entry.Host
	fields["matchPort"] = strconv.Itoa(entry.Port)
}

// seedRuleEntry returns the entry a declared rule joins without creating one:
// the entry the seed declares or Hub stores, or the entry that serves the access
// the rule declares.
func (s *MaintenanceApiServiceServerImpl) seedRuleEntry(rule *seeder.SeedRule, seedEntries map[string]*core.PortalEntry) *core.PortalEntry {
	if rule.EntryName != "" {
		return s.seedPortalEntry(rule.EntryName, seedEntries)
	}
	if entry, ok := s.EntryCore.FindByAccess(rule.Access.Scheme, rule.Access.Host, rule.Access.Port); ok {
		return entry
	}
	access := rule.Access
	access.Name = core.PortalEntryName(access.Scheme, access.Host, access.Port)
	access.Enabled = true
	return &access
}

// seedPortalEntry resolves an entry a seed document declares or Hub stores.
func (s *MaintenanceApiServiceServerImpl) seedPortalEntry(name string, seedEntries map[string]*core.PortalEntry) *core.PortalEntry {
	if entry, ok := seedEntries[name]; ok {
		return entry
	}
	entry, ok := s.EntryCore.FindByName(name)
	if !ok {
		return nil
	}
	return entry
}

func (s *MaintenanceApiServiceServerImpl) previewPortalCert(cert *core.PortalCert) skeled.SeedEntityDiff {
	current, exists := s.CertCore.FindByName(cert.Name)
	return seedEntityDiff(seedKindPortalCert, cert.Name, exists, current.SeedFields(), cert.SeedFields(), seedPortalCertFields)
}

func seedEntityDiff(
	kind string,
	name string,
	exists bool,
	currentFields map[string]string,
	seedFields map[string]string,
	fieldNames []string,
) skeled.SeedEntityDiff {
	fields := make([]skeled.SeedFieldDiff, 0, len(fieldNames))
	for _, fieldName := range fieldNames {
		currentValue := currentFields[fieldName]
		seedValue := seedFields[fieldName]
		fields = append(fields, skeled.SeedFieldDiff{
			Name:         fieldName,
			CurrentValue: currentValue,
			SeedValue:    seedValue,
			Changed:      !exists || currentValue != seedValue,
		})
	}
	return skeled.SeedEntityDiff{
		Kind:   kind,
		Name:   name,
		Exists: exists,
		Fields: fields,
	}
}

func (s *MaintenanceApiServiceServerImpl) applyAppConfigs(items []*core.AppConfig, selected map[_SeedSelectionKey]struct{}) {
	for _, item := range items {
		if !hasSelection(selected, seedKindAppConfig, item.Name) {
			continue
		}
		s.AppConfigCore.Save(*item)
	}
}

func (s *MaintenanceApiServiceServerImpl) applyPortalSites(sites []*core.PortalSite, selected map[_SeedSelectionKey]struct{}) {
	for _, site := range sites {
		if !hasSelection(selected, seedKindPortalSite, site.Name) {
			continue
		}
		s.SiteCore.Save(*site)
	}
}

func (s *MaintenanceApiServiceServerImpl) applyPortalEntries(entries []*core.PortalEntry, selected map[_SeedSelectionKey]struct{}) {
	for _, entry := range entries {
		if !hasSelection(selected, seedKindPortalEntry, entry.Name) {
			continue
		}
		s.EntryCore.Save(*entry)
	}
}

func (s *MaintenanceApiServiceServerImpl) applyPortalRules(rules []*seeder.SeedRule, selected map[_SeedSelectionKey]struct{}) {
	for _, rule := range rules {
		if !hasSelection(selected, seedKindPortalRule, rule.Rule.Name) {
			continue
		}
		s.RuleCore.Save(*seeder.ResolveSeedRule(s.EntryCore, rule))
	}
}

func (s *MaintenanceApiServiceServerImpl) applyPortalCerts(certs []*core.PortalCert, selected map[_SeedSelectionKey]struct{}) {
	for _, cert := range certs {
		if !hasSelection(selected, seedKindPortalCert, cert.Name) {
			continue
		}
		s.CertCore.Save(*cert)
	}
}

func hasSelection(selected map[_SeedSelectionKey]struct{}, kind string, name string) bool {
	_, ok := selected[_SeedSelectionKey{kind: kind, name: name}]
	return ok
}

func (s *MaintenanceApiServiceServerImpl) ConfigReadOnly() bool {
	return s.Access.ReadOnly()
}
