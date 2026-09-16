package seeder

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"gopkg.in/yaml.v3"
)

type _SettingsYAMLPayload struct {
	AppConfigs  []_AppConfig  `yaml:"appConfigs"`
	PortalSites []_PortalSite `yaml:"portalSites"`
	// PortalEntries declares named entries. Hub also creates an entry on its own
	// for the access of a rule that no entry serves yet.
	PortalEntries []_PortalEntry `yaml:"portalEntries"`
	PortalRules   []_PortalRule  `yaml:"portalRules"`
	PortalCerts   []_PortalCert  `yaml:"portalCerts"`
}

// seedStringFields lists the fields each seed section declares as plain strings.
// A seed variable that targets one of them is validated as a string; the fields
// with a structured or numeric type resolve through the cases in vars.go.
var seedStringFields = map[string]map[string]bool{
	"portalSites":   stringFieldsOf(_PortalSite{}),
	"portalEntries": stringFieldsOf(_PortalEntry{}),
	"portalRules":   stringFieldsOf(_PortalRule{}),
	"portalCerts":   stringFieldsOf(_PortalCert{}),
}

// stringFieldsOf derives the string-valued YAML fields of a seed payload struct,
// so the vocabulary of variable targets cannot drift from the contract itself.
func stringFieldsOf(payload any) map[string]bool {
	fields := map[string]bool{}
	payloadType := reflect.TypeOf(payload)
	for index := range payloadType.NumField() {
		field := payloadType.Field(index)
		name := field.Tag.Get("yaml")
		if field.Type.Kind() != reflect.String || name == "" || name == "-" {
			continue
		}
		fields[name] = true
	}
	return fields
}

// SeedEntities is a seed document expressed as the domain entities it declares.
// Startup seeding and Dashboard imports read the same YAML contract through it.
type SeedEntities struct {
	AppConfigs    []*core.AppConfig
	PortalSites   []*core.PortalSite
	PortalEntries []*core.PortalEntry
	PortalRules   []*core.PortalRule
	PortalCerts   []*core.PortalCert
}

// ParseSeedEntities decodes a seed document without resolving seed variables and
// returns the entities it declares. Callers validate the entities they apply.
func ParseSeedEntities(content string) (*SeedEntities, error) {
	payload, err := vcode.UnmarshalYaml[*_SettingsYAMLPayload]([]byte(content))
	if err != nil || payload == nil {
		return nil, err
	}
	if err := checkSeedRuleStyle(payload); err != nil {
		return nil, err
	}
	return payload.entities(), nil
}

func (p *_SettingsYAMLPayload) entities() *SeedEntities {
	entities := &SeedEntities{
		AppConfigs:    make([]*core.AppConfig, 0, len(p.AppConfigs)),
		PortalSites:   make([]*core.PortalSite, 0, len(p.PortalSites)),
		PortalEntries: make([]*core.PortalEntry, 0, len(p.PortalEntries)),
		PortalRules:   make([]*core.PortalRule, 0, len(p.PortalRules)),
		PortalCerts:   make([]*core.PortalCert, 0, len(p.PortalCerts)),
	}
	for _, item := range p.AppConfigs {
		entities.AppConfigs = append(entities.AppConfigs, item.toCoreAppConfig())
	}
	for _, site := range p.PortalSites {
		entities.PortalSites = append(entities.PortalSites, site.toCorePortalSite())
	}
	for _, entry := range p.PortalEntries {
		entities.PortalEntries = append(entities.PortalEntries, entry.toCorePortalEntry())
	}
	for _, rule := range p.PortalRules {
		entities.PortalRules = append(entities.PortalRules, rule.toCorePortalRule())
	}
	for _, cert := range p.PortalCerts {
		entities.PortalCerts = append(entities.PortalCerts, cert.toCorePortalCert())
	}
	return entities
}

func (p *_SettingsYAMLPayload) UnmarshalYAML(node *yaml.Node) error {
	if err := checkSeedYAMLSyntax(node); err != nil {
		return err
	}
	type _Plain _SettingsYAMLPayload
	return node.Decode((*_Plain)(p))
}

var plainYAMLNumber = regexp.MustCompile(`^[+-]?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// checkSeedYAMLSyntax rejects unsupported references and ambiguous numbers before seed decoding.
func checkSeedYAMLSyntax(node *yaml.Node) error {
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
	return decodeAppConfig(node, (*plain)(i))
}

func (i _AppConfig) toCoreAppConfig() *core.AppConfig {
	return &core.AppConfig{FieldSources: i.Sources, Name: i.Name, Value: i.Value}
}

// Portal entry

type _PortalEntry struct {
	Name   string `yaml:"name"`
	Scheme string `yaml:"scheme"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
}

func (e _PortalEntry) toCorePortalEntry() *core.PortalEntry {
	return &core.PortalEntry{
		Name:   e.Name,
		Scheme: e.Scheme,
		Host:   e.Host,
		Port:   e.Port,
	}
}

// Portal rule

type _PortalRule struct {
	Sources core.FieldSources `yaml:"-"`
	Name    string            `yaml:"name"`
	// EntryName names the entry the rule joins. A rule declares either the entry
	// name or the access the entry serves, never both.
	EntryName               string `yaml:"entryName"`
	MatchScheme             string `yaml:"matchScheme"`
	MatchHost               string `yaml:"matchHost"`
	MatchPort               int    `yaml:"matchPort"`
	MatchPathPrefix         string `yaml:"matchPathPrefix"`
	RouteType               string `yaml:"routeType"`
	RouteSiteName           string `yaml:"routeSiteName"`
	RouteRedirectionPattern string `yaml:"routeRedirectionPattern"`
	RoutePathPrefix         string `yaml:"routePathPrefix"`
}

func (r _PortalRule) toCorePortalRule() *core.PortalRule {
	return &core.PortalRule{FieldSources: r.Sources,
		Name:                    r.Name,
		EntryName:               r.EntryName,
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

func (s _PortalSite) toCorePortalSite() *core.PortalSite {
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

func (c _PortalCert) toCorePortalCert() *core.PortalCert {
	cert := &core.PortalCert{FieldSources: c.Sources,
		Name:             c.Name,
		PublicKeyBase64:  c.PublicKeyBase64,
		PrivateKeyBase64: c.PrivateKeyBase64,
	}
	return cert
}

func (r *_PortalRule) UnmarshalYAML(node *yaml.Node) error {
	// TODO: Remove legacy field decoding from startup seeds when old YAML support
	// is retired, together with decodePortalRule compatibility logic.
	type plain _PortalRule
	return decodePortalRule(node, (*plain)(r))
}
