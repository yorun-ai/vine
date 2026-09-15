package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
)

const (
	seedKindAppConfig  = "app_config"
	seedKindPortalSite = "portal_site"
	seedKindPortalRule = "portal_rule"
	seedKindPortalCert = "portal_cert"
)

type MaintenanceApiServiceServerImpl struct {
	skeled.DefaultMaintenanceApiServiceServer

	AppConfigCore *core.AppConfigCore  `inject:""`
	SiteCore      *core.PortalSiteCore `inject:""`
	RuleCore      *core.PortalRuleCore `inject:""`
	CertCore      *core.PortalCertCore `inject:""`
	Access        *configaccess.Access `inject:""`
}

// The fields each seed entity kind compares, in the order the Dashboard shows
// them. Both sides of the comparison use Core SeedFields, so the preview cannot
// drift from the values a seed applies.
var (
	seedAppConfigFields  = []string{"value"}
	seedPortalSiteFields = []string{"type", "actorSkelName", "actorVia", "corsMode", "corsOrigins", "webName"}
	seedPortalRuleFields = []string{
		"matchScheme", "matchHost", "matchPort", "matchPathPrefix",
		"routeType", "routeSiteName", "routeRedirectionPattern", "routePathPrefix",
	}
	seedPortalCertFields = []string{
		"issuer", "domains", "publicKeyBase64", "privateKeyBase64", "validFrom", "validTo",
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
	for i, rule := range entities.PortalRules {
		validated := s.RuleCore.Validate(*rule)
		entities.PortalRules[i] = &validated
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
	for _, rule := range entities.PortalRules {
		preview.Items = append(preview.Items, s.previewPortalRule(rule))
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

func (s *MaintenanceApiServiceServerImpl) previewPortalRule(rule *core.PortalRule) skeled.SeedEntityDiff {
	current, exists := s.RuleCore.FindByName(rule.Name)
	return seedEntityDiff(seedKindPortalRule, rule.Name, exists, current.SeedFields(), rule.SeedFields(), seedPortalRuleFields)
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

func (s *MaintenanceApiServiceServerImpl) applyPortalRules(rules []*core.PortalRule, selected map[_SeedSelectionKey]struct{}) {
	for _, rule := range rules {
		if !hasSelection(selected, seedKindPortalRule, rule.Name) {
			continue
		}
		s.RuleCore.Save(*rule)
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
