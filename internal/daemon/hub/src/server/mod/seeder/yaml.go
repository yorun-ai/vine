package seeder

import (
	"fmt"
	"regexp"
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

type _SettingsYAMLPayload struct {
	AppConfigs    []_AppConfig  `yaml:"appConfigs"`
	PortalEntries []_PortalSite `yaml:"portalSites"`
	PortalRules   []_PortalRule `yaml:"portalRules"`
	PortalCerts   []_PortalCert `yaml:"portalCerts"`
}

func (p *_SettingsYAMLPayload) UnmarshalYAML(node *yaml.Node) error {
	if err := CheckSeedYAMLSyntax(node); err != nil {
		return err
	}
	type _Plain _SettingsYAMLPayload
	return node.Decode((*_Plain)(p))
}

var plainYAMLNumber = regexp.MustCompile(`^[+-]?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// CheckSeedYAMLSyntax rejects unsupported references and ambiguous numbers before seed decoding.
func CheckSeedYAMLSyntax(node *yaml.Node) error {
	return checkSeedYAMLNode(node)
}

func checkSeedYAMLNode(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return fmt.Errorf("line %d, column %d: YAML anchors and aliases are not supported", node.Line, node.Column)
	}
	if node.ShortTag() == "!!merge" {
		return fmt.Errorf("line %d, column %d: YAML merge keys are not supported", node.Line, node.Column)
	}
	if (node.ShortTag() == "!!int" || node.ShortTag() == "!!float") && !plainYAMLNumber.MatchString(node.Value) {
		return fmt.Errorf("line %d, column %d: unsupported YAML number %q; use plain decimal notation without separators, leading zeros, or exponents", node.Line, node.Column, node.Value)
	}
	for _, child := range node.Content {
		if err := checkSeedYAMLNode(child); err != nil {
			return err
		}
	}
	return nil
}

// App config

type _AppConfig struct {
	Sources core.FieldSources `yaml:"-"`
	Name    string            `yaml:"name"`
	Value   string            `yaml:"value"`
}

func (i *_AppConfig) UnmarshalYAML(node *yaml.Node) error {
	type plain _AppConfig
	return DecodeAppConfig(node, (*plain)(i))
}

func (i _AppConfig) ToCoreAppConfig() *core.AppConfig {
	return &core.AppConfig{FieldSources: i.Sources, Name: i.Name, Value: i.Value}
}

// Portal rule

type _PortalRule struct {
	Sources                 core.FieldSources `yaml:"-"`
	Name                    string            `yaml:"name"`
	MatchScheme             string            `yaml:"matchScheme"`
	MatchHost               string            `yaml:"matchHost"`
	MatchPort               int               `yaml:"matchPort"`
	MatchPathPrefix         string            `yaml:"matchPathPrefix"`
	RouteType               string            `yaml:"routeType"`
	RouteSiteName           string            `yaml:"routeSiteName"`
	RouteRedirectionPattern string            `yaml:"routeRedirectionPattern"`
	RoutePathPrefix         string            `yaml:"routePathPrefix"`
}

func (r _PortalRule) ToCorePortalRule() *core.PortalRule {
	return &core.PortalRule{FieldSources: r.Sources,
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
	Sources       core.FieldSources `yaml:"-"`
	Name          string            `yaml:"name"`
	Type          string            `yaml:"type"`
	ActorSkelName string            `yaml:"actorSkelName"`
	ActorVia      string            `yaml:"actorVia"`
	Cors          _PortalCors       `yaml:"cors"`
	WebName       string            `yaml:"webName"`
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
	site := &core.PortalSite{FieldSources: s.Sources,
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
	Sources          core.FieldSources `yaml:"-"`
	Name             string            `yaml:"name"`
	Issuer           string            `yaml:"issuer"`
	Domains          []string          `yaml:"domains"`
	PublicKeyBase64  string            `yaml:"publicKeyBase64"`
	PrivateKeyBase64 string            `yaml:"privateKeyBase64"`
	ValidFrom        time.Time         `yaml:"validFrom"`
	ValidTo          time.Time         `yaml:"validTo"`
}

func (c _PortalCert) ToCorePortalCert() *core.PortalCert {
	cert := &core.PortalCert{FieldSources: c.Sources,
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
