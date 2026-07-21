package seeder

import (
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
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

func (i _AppConfig) ToCoreAppConfig(current *core.AppConfig) *core.AppConfig {
	version := 1
	if current != nil {
		version = current.Version
		if current.Value != i.Value {
			version++
		}
	}
	config := &core.AppConfig{
		Name:    i.Name,
		Value:   i.Value,
		Version: version,
	}
	if current != nil {
		config.Id = current.Id
	}
	return config
}

// Portal rule

type _PortalRule struct {
	Name               string `yaml:"name"`
	Scheme             string `yaml:"scheme"`
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	PathPrefix         string `yaml:"pathPrefix"`
	TargetType         string `yaml:"targetType"`
	SiteName           string `yaml:"siteName"`
	RedirectionPattern string `yaml:"redirectionPattern"`
	Override           bool   `yaml:"override"`
}

func (r _PortalRule) ToCorePortalRule(current *core.PortalRule) *core.PortalRule {
	rule := &core.PortalRule{
		Name:               r.Name,
		Scheme:             r.Scheme,
		Host:               r.Host,
		Port:               r.Port,
		PathPrefix:         r.PathPrefix,
		TargetType:         r.TargetType,
		SiteName:           r.SiteName,
		RedirectionPattern: r.RedirectionPattern,
	}
	if current != nil {
		rule.Id = current.Id
		rule.BuiltIn = current.BuiltIn
	}
	return rule
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

func (s _PortalSite) ToCorePortalSite(current *core.PortalSite) *core.PortalSite {
	cors := core.NormalizePortalCors(core.PortalCors{
		Mode:           core.PortalCorsMode(s.Cors.Mode),
		AllowedOrigins: append([]string{}, s.Cors.AllowedOrigins...),
	})
	site := &core.PortalSite{
		Name:          s.Name,
		Type:          core.PortalSiteType(s.Type),
		ActorSkelName: s.ActorSkelName,
		ActorVia:      s.ActorVia,
		Cors:          cors,
		WebName:       s.WebName,
	}
	if current != nil {
		site.Id = current.Id
		site.BuiltIn = current.BuiltIn
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

func (c _PortalCert) ToCorePortalCert(current *core.PortalCert) *core.PortalCert {
	cert := &core.PortalCert{
		Name:             c.Name,
		Issuer:           c.Issuer,
		Domains:          append([]string(nil), c.Domains...),
		PublicKeyBase64:  c.PublicKeyBase64,
		PrivateKeyBase64: c.PrivateKeyBase64,
		ValidFrom:        c.ValidFrom,
		ValidTo:          c.ValidTo,
	}
	if current != nil {
		cert.Id = current.Id
	}
	return cert
}
