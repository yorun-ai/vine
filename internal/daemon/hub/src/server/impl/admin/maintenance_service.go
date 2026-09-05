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
		if current, ok := s.AppConfigRepo.GetItemByName(item.Name); ok {
			next := *current
			next.Value = item.Value
			if next.Value != current.Value {
				next.Version++
			}
			s.AppConfigRepo.SaveItem(&next)
			continue
		}

		next := &core.AppConfig{
			Name:    item.Name,
			Value:   item.Value,
			Version: 1,
		}
		s.AppConfigRepo.SaveItem(next)
	}
}

func (s *MaintenanceServiceServerImpl) applyPortalEntries(entries []_SeedPortalSite, selected map[_SeedSelectionKey]struct{}) {
	for _, entry := range entries {
		if !hasSelection(selected, seedKindPortalSite, entry.Name) {
			continue
		}
		if current, ok := s.EntryRepo.GetEntryByName(entry.Name); ok {
			next := *current
			next.Type = core.PortalSiteType(entry.Type)
			next.ActorSkelName = entry.ActorSkelName
			next.ActorVia = entry.ActorVia
			next.Cors = core.NormalizePortalCors(core.PortalCors{
				Mode:           core.PortalCorsMode(entry.Cors.Mode),
				AllowedOrigins: append([]string{}, entry.Cors.AllowedOrigins...),
			})
			next.WebName = entry.WebName
			s.EntryRepo.SaveEntry(&next)
			continue
		}

		next := &core.PortalSite{
			Name:          entry.Name,
			Type:          core.PortalSiteType(entry.Type),
			ActorSkelName: entry.ActorSkelName,
			ActorVia:      entry.ActorVia,
			Cors: core.NormalizePortalCors(core.PortalCors{
				Mode:           core.PortalCorsMode(entry.Cors.Mode),
				AllowedOrigins: append([]string{}, entry.Cors.AllowedOrigins...),
			}),
			WebName: entry.WebName,
		}
		s.EntryRepo.SaveEntry(next)
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
		if current, ok := s.CertRepo.GetCertByName(cert.Name); ok {
			next := *current
			next.Issuer = cert.Issuer
			next.Domains = append([]string(nil), cert.Domains...)
			next.PublicKeyBase64 = cert.PublicKeyBase64
			next.PrivateKeyBase64 = cert.PrivateKeyBase64
			next.ValidFrom = cert.ValidFrom
			next.ValidTo = cert.ValidTo
			s.CertRepo.SaveCert(&next)
			continue
		}

		next := &core.PortalCert{
			Name:             cert.Name,
			Issuer:           cert.Issuer,
			Domains:          append([]string(nil), cert.Domains...),
			PublicKeyBase64:  cert.PublicKeyBase64,
			PrivateKeyBase64: cert.PrivateKeyBase64,
			ValidFrom:        cert.ValidFrom,
			ValidTo:          cert.ValidTo,
		}
		s.CertRepo.SaveCert(next)
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
