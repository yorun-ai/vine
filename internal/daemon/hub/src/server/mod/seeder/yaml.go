package seeder

import (
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
	"gopkg.in/yaml.v3"
)

type _SettingsYAMLPayload struct {
	AppConfigs    []_AppConfig  `yaml:"appConfigs"`
	PortalEntries []_PortalSite `yaml:"portalSites"`
	PortalRules   []_PortalRule `yaml:"portalRules"`
	PortalCerts   []_PortalCert `yaml:"portalCerts"`
}

func (p *_SettingsYAMLPayload) Overridden() (*_SettingsYAMLPayload, bool) {
	overridden := &_SettingsYAMLPayload{
		AppConfigs: vslice.Filter(p.AppConfigs, func(item _AppConfig) bool {
			return item.Override
		}),
		PortalEntries: vslice.Filter(p.PortalEntries, func(item _PortalSite) bool {
			return item.Override
		}),
		PortalRules: vslice.Filter(p.PortalRules, func(item _PortalRule) bool {
			return item.Override
		}),
		PortalCerts: vslice.Filter(p.PortalCerts, func(item _PortalCert) bool {
			return item.Override
		}),
	}

	hasOverridden := len(overridden.AppConfigs) > 0 ||
		len(overridden.PortalEntries) > 0 ||
		len(overridden.PortalRules) > 0 ||
		len(overridden.PortalCerts) > 0

	return overridden, hasOverridden
}

// App config

type _AppConfig struct {
	Name     string `yaml:"name"`
	Value    string `yaml:"value"`
	Override bool   `yaml:"override"`
}

func (i _AppConfig) ToCoreAppConfig() *core.AppConfig {
	return &core.AppConfig{Name: i.Name, Value: i.Value}
}

// Portal rule

type _PortalRule struct {
	Name                    string `yaml:"name"`
	MatchScheme             string `yaml:"matchScheme"`
	MatchHost               string `yaml:"matchHost"`
	MatchPort               int    `yaml:"matchPort"`
	MatchPathPrefix         string `yaml:"matchPathPrefix"`
	RouteType               string `yaml:"routeType"`
	RouteSiteName           string `yaml:"routeSiteName"`
	RouteRedirectionPattern string `yaml:"routeRedirectionPattern"`
	RoutePathPrefix         string `yaml:"routePathPrefix"`
	Override                bool   `yaml:"override"`
}

func (r _PortalRule) ToCorePortalRule() *core.PortalRule {
	return &core.PortalRule{
		Name:                    r.Name,
		MatchScheme:             r.MatchScheme,
		MatchHost:               r.MatchHost,
		MatchPort:               r.MatchPort,
		MatchPathPrefix:         r.MatchPathPrefix,
		RouteType:               r.RouteType,
		RouteSiteName:           r.RouteSiteName,
		RouteRedirectionPattern: r.RouteRedirectionPattern,
		RoutePathPrefix:         r.RoutePathPrefix,
	}
}

// Portal site

type _PortalSite struct {
	Name          string      `yaml:"name"`
	Type          string      `yaml:"type"`
	ActorSkelName string      `yaml:"actorSkelName"`
	ActorVia      string      `yaml:"actorVia"`
	Cors          _PortalCors `yaml:"cors"`
	WebName       string      `yaml:"webName"`
	Override      bool        `yaml:"override"`
}

type _PortalCors struct {
	Mode           string   `yaml:"mode"`
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

func (s _PortalSite) ToCorePortalSite() *core.PortalSite {
	cors := core.PortalCors{
		Mode:           core.PortalCorsMode(s.Cors.Mode),
		AllowedOrigins: append([]string{}, s.Cors.AllowedOrigins...),
	}
	site := &core.PortalSite{
		Name:          s.Name,
		Type:          core.PortalSiteType(s.Type),
		ActorSkelName: s.ActorSkelName,
		ActorVia:      s.ActorVia,
		Cors:          cors,
		WebName:       s.WebName,
	}
	return site
}

// Portal cert

type _PortalCert struct {
	Name             string    `yaml:"name"`
	Issuer           string    `yaml:"issuer"`
	Domains          []string  `yaml:"domains"`
	PublicKeyBase64  string    `yaml:"publicKeyBase64"`
	PrivateKeyBase64 string    `yaml:"privateKeyBase64"`
	ValidFrom        time.Time `yaml:"validFrom"`
	ValidTo          time.Time `yaml:"validTo"`
	Override         bool      `yaml:"override"`
}

func (c _PortalCert) ToCorePortalCert() *core.PortalCert {
	cert := &core.PortalCert{
		Name:             c.Name,
		PublicKeyBase64:  c.PublicKeyBase64,
		PrivateKeyBase64: c.PrivateKeyBase64,
	}
	return cert
}

func (r *_PortalRule) UnmarshalYAML(node *yaml.Node) error {
	// TODO: Remove legacy field decoding from startup seeds when old YAML support
	// is retired, together with DecodePortalRule compatibility logic.
	type plain _PortalRule
	return DecodePortalRule(node, (*plain)(r))
}
