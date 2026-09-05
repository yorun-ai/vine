package admin

import (
	"strconv"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/util/vcode"
	"gopkg.in/yaml.v3"
)

const (
	seedKindAppConfig  = "app_config"
	seedKindPortalSite = "portal_site"
	seedKindPortalRule = "portal_rule"
	seedKindPortalCert = "portal_cert"
)

type MaintenanceServiceServerImpl struct {
	skeled.DefaultMaintenanceServiceServer

	AppConfigRepo core.AppConfigRepo   `inject:""`
	EntryRepo     core.PortalSiteRepo  `inject:""`
	RuleRepo      core.PortalRuleRepo  `inject:""`
	RuleCore      *core.PortalRuleCore `inject:""`
	AppConfigCore *core.AppConfigCore  `inject:""`
	SiteCore      *core.PortalSiteCore `inject:""`
	CertCore      *core.PortalCertCore `inject:""`
	CertRepo      core.PortalCertRepo  `inject:""`
}

type _SeedYAMLPayload struct {
	AppConfigs    []_SeedAppConfig  `yaml:"appConfigs"`
	PortalEntries []_SeedPortalSite `yaml:"portalSites"`
	PortalRules   []_SeedPortalRule `yaml:"portalRules"`
	PortalCerts   []_SeedPortalCert `yaml:"portalCerts"`
}

type _SeedAppConfig struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type _SeedPortalSite struct {
	Name          string          `yaml:"name"`
	Type          string          `yaml:"type"`
	ActorSkelName string          `yaml:"actorSkelName"`
	ActorVia      string          `yaml:"actorVia"`
	Cors          _SeedPortalCors `yaml:"cors"`
	WebName       string          `yaml:"webName"`
}

type _SeedPortalCors struct {
	Mode           string   `yaml:"mode"`
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

type _SeedPortalRule struct {
	Name                    string `yaml:"name"`
	MatchScheme             string `yaml:"matchScheme"`
	MatchHost               string `yaml:"matchHost"`
	MatchPort               int    `yaml:"matchPort"`
	MatchPathPrefix         string `yaml:"matchPathPrefix"`
	RoutePathPrefix         string `yaml:"routePathPrefix"`
	RouteType               string `yaml:"routeType"`
	RouteSiteName           string `yaml:"routeSiteName"`
	RouteRedirectionPattern string `yaml:"routeRedirectionPattern"`
}

type _SeedPortalCert struct {
	Name             string    `yaml:"name"`
	Issuer           string    `yaml:"issuer"`
	Domains          []string  `yaml:"domains"`
	PublicKeyBase64  string    `yaml:"publicKeyBase64"`
	PrivateKeyBase64 string    `yaml:"privateKeyBase64"`
	ValidFrom        time.Time `yaml:"validFrom"`
	ValidTo          time.Time `yaml:"validTo"`
}

type _FieldValue struct {
	name  string
	value string
}

type _SeedSelectionKey struct {
	kind string
	name string
}

func (s *MaintenanceServiceServerImpl) PreviewSeedYaml(content string) skeled.SeedPreview {
	return s.preview(s.parseSeed(content))
}

func (s *MaintenanceServiceServerImpl) ApplySeedYaml(content string, selections []skeled.SeedItemSelection) skeled.SeedPreview {
	payload := s.parseSeed(content)
	selected := map[_SeedSelectionKey]struct{}{}
	for _, selection := range selections {
		selected[_SeedSelectionKey{
			kind: selection.Kind,
			name: selection.Name,
		}] = struct{}{}
	}

	s.applyAppConfigs(payload.AppConfigs, selected)
	s.applyPortalEntries(payload.PortalEntries, selected)
	s.applyPortalRules(payload.PortalRules, selected)
	s.applyPortalCerts(payload.PortalCerts, selected)
	return s.preview(payload)
}

func (s *MaintenanceServiceServerImpl) parseSeed(content string) *_SeedYAMLPayload {
	payload, err := vcode.UnmarshalYaml[*_SeedYAMLPayload]([]byte(content))
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, ex.F("parse seed yaml failed: %v", err))
	if payload != nil {
		for _, item := range payload.AppConfigs {
			s.AppConfigCore.Validate(item.toCore())
		}
		for i := range payload.PortalEntries {
			entity := s.SiteCore.Validate(payload.PortalEntries[i].toCore())
			payload.PortalEntries[i].Cors = _SeedPortalCors{Mode: string(entity.Cors.Mode), AllowedOrigins: entity.Cors.AllowedOrigins}
		}
		for i := range payload.PortalCerts {
			entity := s.CertCore.Validate(payload.PortalCerts[i].toCore())
			payload.PortalCerts[i].Issuer = entity.Issuer
			payload.PortalCerts[i].Domains = entity.Domains
			payload.PortalCerts[i].ValidFrom = entity.ValidFrom
			payload.PortalCerts[i].ValidTo = entity.ValidTo
		}
		for i := range payload.PortalRules {
			rule := &payload.PortalRules[i]
			entity := s.RuleCore.Validate(rule.toCore())
			rule.RoutePathPrefix = entity.RoutePathPrefix
		}
	}
	return payload
}

func (s *MaintenanceServiceServerImpl) preview(payload *_SeedYAMLPayload) skeled.SeedPreview {
	if payload == nil {
		return newSeedPreview()
	}

	preview := newSeedPreview()
	for _, item := range payload.AppConfigs {
		preview.Items = append(preview.Items, s.previewAppConfig(item))
	}
	for _, entry := range payload.PortalEntries {
		preview.Items = append(preview.Items, s.previewPortalSite(entry))
	}
	for _, rule := range payload.PortalRules {
		preview.Items = append(preview.Items, s.previewPortalRule(rule))
	}
	for _, cert := range payload.PortalCerts {
		preview.Items = append(preview.Items, s.previewPortalCert(cert))
	}
	return preview
}

func newSeedPreview() skeled.SeedPreview {
	return skeled.SeedPreview{Items: make([]skeled.SeedEntityDiff, 0)}
}

func (s *MaintenanceServiceServerImpl) previewAppConfig(item _SeedAppConfig) skeled.SeedEntityDiff {
	current, exists := s.AppConfigRepo.GetItemByName(item.Name)
	return seedEntityDiff(seedKindAppConfig, item.Name, exists, currentConfigFields(current), []_FieldValue{
		{"value", item.Value},
	})
}

func (s *MaintenanceServiceServerImpl) previewPortalSite(entry _SeedPortalSite) skeled.SeedEntityDiff {
	current, exists := s.EntryRepo.GetEntryByName(entry.Name)
	return seedEntityDiff(seedKindPortalSite, entry.Name, exists, currentPortalSiteFields(current), []_FieldValue{
		{"type", entry.Type},
		{"actorSkelName", entry.ActorSkelName},
		{"actorVia", entry.ActorVia},
		{"webName", entry.WebName},
	})
}

func (s *MaintenanceServiceServerImpl) previewPortalRule(rule _SeedPortalRule) skeled.SeedEntityDiff {
	current, exists := s.RuleRepo.GetRuleByName(rule.Name)
	return seedEntityDiff(seedKindPortalRule, rule.Name, exists, currentPortalRuleFields(current), []_FieldValue{
		{"matchScheme", rule.MatchScheme},
		{"matchHost", rule.MatchHost},
		{"matchPort", intString(rule.MatchPort)},
		{"matchPathPrefix", rule.MatchPathPrefix},
		{"routePathPrefix", rule.RoutePathPrefix},
		{"routeType", rule.RouteType},
		{"routeSiteName", rule.RouteSiteName},
		{"routeRedirectionPattern", rule.RouteRedirectionPattern},
	})
}

func (s *MaintenanceServiceServerImpl) previewPortalCert(cert _SeedPortalCert) skeled.SeedEntityDiff {
	current, exists := s.CertRepo.GetCertByName(cert.Name)
	return seedEntityDiff(seedKindPortalCert, cert.Name, exists, currentPortalCertFields(current), []_FieldValue{
		{"issuer", cert.Issuer},
		{"domains", jsonString(cert.Domains)},
		{"publicKeyBase64", cert.PublicKeyBase64},
		{"privateKeyBase64", cert.PrivateKeyBase64},
		{"validFrom", timeString(cert.ValidFrom)},
		{"validTo", timeString(cert.ValidTo)},
	})
}

func seedEntityDiff(kind string, name string, exists bool, currentFields map[string]string, seedFields []_FieldValue) skeled.SeedEntityDiff {
	fields := make([]skeled.SeedFieldDiff, 0, len(seedFields))
	for _, field := range seedFields {
		currentValue := currentFields[field.name]
		fields = append(fields, skeled.SeedFieldDiff{
			Name:         field.name,
			CurrentValue: currentValue,
			SeedValue:    field.value,
			Changed:      !exists || currentValue != field.value,
		})
	}
	return skeled.SeedEntityDiff{
		Kind:   kind,
		Name:   name,
		Exists: exists,
		Fields: fields,
	}
}

func (s *MaintenanceServiceServerImpl) applyAppConfigs(items []_SeedAppConfig, selected map[_SeedSelectionKey]struct{}) {
	for _, item := range items {
		if !hasSelection(selected, seedKindAppConfig, item.Name) {
			continue
		}
		s.AppConfigCore.Save(item.toCore())
	}
}

func (s *MaintenanceServiceServerImpl) applyPortalEntries(entries []_SeedPortalSite, selected map[_SeedSelectionKey]struct{}) {
	for _, entry := range entries {
		if !hasSelection(selected, seedKindPortalSite, entry.Name) {
			continue
		}
		s.SiteCore.Save(entry.toCore())
	}
}

func (s *MaintenanceServiceServerImpl) applyPortalRules(rules []_SeedPortalRule, selected map[_SeedSelectionKey]struct{}) {
	for _, rule := range rules {
		if !hasSelection(selected, seedKindPortalRule, rule.Name) {
			continue
		}
		s.RuleCore.Save(rule.toCore())
	}
}

func (s *MaintenanceServiceServerImpl) applyPortalCerts(certs []_SeedPortalCert, selected map[_SeedSelectionKey]struct{}) {
	for _, cert := range certs {
		if !hasSelection(selected, seedKindPortalCert, cert.Name) {
			continue
		}
		s.CertCore.Save(cert.toCore())
	}
}

func currentConfigFields(item *core.AppConfig) map[string]string {
	if item == nil {
		return map[string]string{}
	}
	return map[string]string{
		"value": item.Value,
	}
}

func currentPortalSiteFields(entry *core.PortalSite) map[string]string {
	if entry == nil {
		return map[string]string{}
	}
	return map[string]string{
		"type":          string(entry.Type),
		"actorSkelName": entry.ActorSkelName,
		"actorVia":      entry.ActorVia,
		"corsMode":      string(entry.Cors.Mode),
		"webName":       entry.WebName,
	}
}

func currentPortalRuleFields(rule *core.PortalRule) map[string]string {
	if rule == nil {
		return map[string]string{}
	}
	return map[string]string{
		"matchScheme":             rule.MatchScheme,
		"matchHost":               rule.MatchHost,
		"matchPort":               intString(rule.MatchPort),
		"matchPathPrefix":         rule.MatchPathPrefix,
		"routePathPrefix":         rule.RoutePathPrefix,
		"routeType":               rule.RouteType,
		"routeSiteName":           rule.RouteSiteName,
		"routeRedirectionPattern": rule.RouteRedirectionPattern,
	}
}

func currentPortalCertFields(cert *core.PortalCert) map[string]string {
	if cert == nil {
		return map[string]string{}
	}
	return map[string]string{
		"issuer":           cert.Issuer,
		"domains":          jsonString(cert.Domains),
		"publicKeyBase64":  cert.PublicKeyBase64,
		"privateKeyBase64": cert.PrivateKeyBase64,
		"validFrom":        timeString(cert.ValidFrom),
		"validTo":          timeString(cert.ValidTo),
	}
}

func hasSelection(selected map[_SeedSelectionKey]struct{}, kind string, name string) bool {
	_, ok := selected[_SeedSelectionKey{kind: kind, name: name}]
	return ok
}

func jsonString(value []string) string {
	if value == nil {
		value = []string{}
	}
	bytes, err := vcode.MarshalJson(value)
	ex.PanicIfError(err)
	return string(bytes)
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func timeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (r *_SeedPortalRule) UnmarshalYAML(node *yaml.Node) error {
	// TODO: Remove legacy field decoding from Dashboard imports when old YAML
	// support is retired, together with seeder.DecodePortalRule compatibility logic.
	type plain _SeedPortalRule
	return seeder.DecodePortalRule(node, (*plain)(r))
}

func (r _SeedPortalRule) toCore() core.PortalRule {
	return core.PortalRule{
		Name: r.Name, MatchScheme: r.MatchScheme, MatchHost: r.MatchHost, MatchPort: r.MatchPort,
		MatchPathPrefix: r.MatchPathPrefix, RouteType: r.RouteType, RouteSiteName: r.RouteSiteName,
		RoutePathPrefix: r.RoutePathPrefix, RouteRedirectionPattern: r.RouteRedirectionPattern,
	}
}

func (i _SeedAppConfig) toCore() core.AppConfig {
	return core.AppConfig{Name: i.Name, Value: i.Value}
}
func (s _SeedPortalSite) toCore() core.PortalSite {
	return core.PortalSite{
		Name:          s.Name,
		Type:          core.PortalSiteType(s.Type),
		ActorSkelName: s.ActorSkelName,
		ActorVia:      s.ActorVia,
		WebName:       s.WebName,
		Cors: core.PortalCors{
			Mode:           core.PortalCorsMode(s.Cors.Mode),
			AllowedOrigins: append([]string{}, s.Cors.AllowedOrigins...),
		},
	}
}
func (c _SeedPortalCert) toCore() core.PortalCert {
	return core.PortalCert{Name: c.Name, PublicKeyBase64: c.PublicKeyBase64, PrivateKeyBase64: c.PrivateKeyBase64}
}
