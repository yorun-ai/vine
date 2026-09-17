package seeder

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"gopkg.in/yaml.v3"
)

type _SettingsYAMLPayload struct {
	AppConfigs    []_AppConfig   `yaml:"appConfigs"`
	PortalSites   []_PortalSite  `yaml:"portalSites"`
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

// seedSectionFields lists every field a seed section declares, so a field Hub
// does not know fails instead of silently leaving the entity at its default.
// Portal sections are fully typed: unlike an app config value, nothing in them
// is free-form. The rule section also accepts the legacy names it warns about.
var seedSectionFields = map[string]map[string]bool{
	"portalSites":    yamlFieldsOf(_PortalSite{}),
	"portalEntries":  yamlFieldsOf(_PortalEntry{}),
	"portalRules":    yamlFieldsOf(_PortalRule{}, portalRuleAliases),
	"portalCerts":    yamlFieldsOf(_PortalCert{}),
	"portalSiteCors": yamlFieldsOf(_PortalCors{}),
}

// yamlFieldsOf derives every YAML field a seed payload struct declares. Extra
// name sets carry field names the contract still accepts under another name.
func yamlFieldsOf(payload any, extra ...map[string]string) map[string]bool {
	fields := map[string]bool{}
	payloadType := reflect.TypeOf(payload)
	for index := range payloadType.NumField() {
		field := payloadType.Field(index)
		name := field.Tag.Get("yaml")
		if name == "" || name == "-" {
			continue
		}
		fields[name] = true
	}
	for _, names := range extra {
		for name := range names {
			fields[name] = true
		}
	}
	return fields
}

// seedEntities is a seed document expressed as the domain entities it declares.
type seedEntities struct {
	AppConfigs    []*core.AppConfig
	PortalSites   []*core.PortalSite
	PortalEntries []*core.PortalEntry
	PortalRules   []*seedRule
	PortalCerts   []*core.PortalCert
}

// seedRule is one Portal rule a seed declares together with the entry it joins:
// the entry the document names, or the scheme, host, and port the rule declares
// and Hub ensures when it applies the rule.
type seedRule struct {
	Rule *core.PortalRule
	// EntryName names a Portal entry the same document declares. Empty means the
	// rule declares the scheme, host, and port of the entry it joins.
	EntryName string
	// Entry holds the scheme, host, and port the rule declares, used when
	// EntryName is empty.
	Entry core.PortalEntry
}

// resolveSeedRule points a seed rule at the entry it joins: the entry the seed
// names, or the entry that serves the scheme, host, and port the rule declares.
// Hub ensures the latter, the way it does for any rule whose fields no entry
// serves yet.
func resolveSeedRule(entryCore *core.PortalEntryCore, rule *seedRule) *core.PortalRule {
	resolved := *rule.Rule
	if rule.EntryName != "" {
		entry, ok := entryCore.FindByName(rule.EntryName)
		ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("portal entry %s not found", rule.EntryName))
		resolved.EntryId = entry.Id
		return &resolved
	}
	resolved.EntryId = entryCore.EnsureEntry(rule.Entry.Scheme, rule.Entry.Host, rule.Entry.Port).Id
	return &resolved
}

// parseSeedEntities decodes a seed document without resolving seed variables and
// returns the entities it declares. Callers validate the entities they apply.
func parseSeedEntities(content string) (*seedEntities, error) {
	payload, err := vcode.UnmarshalYaml[*_SettingsYAMLPayload]([]byte(content))
	if err != nil || payload == nil {
		return nil, err
	}
	if err := checkSeedRuleStyle(payload); err != nil {
		return nil, err
	}
	return payload.entities(), nil
}

func (p *_SettingsYAMLPayload) entities() *seedEntities {
	entities := &seedEntities{
		AppConfigs:    make([]*core.AppConfig, 0, len(p.AppConfigs)),
		PortalSites:   make([]*core.PortalSite, 0, len(p.PortalSites)),
		PortalEntries: make([]*core.PortalEntry, 0, len(p.PortalEntries)),
		PortalRules:   make([]*seedRule, 0, len(p.PortalRules)),
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
		entities.PortalRules = append(entities.PortalRules, rule.toSeedRule())
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
	// Disabled is optional and defaults to false. A seed declares the exception,
	// so Hub keeps the positive spelling of the switch it stores: enabled.
	Disabled bool `yaml:"disabled"`
}

func (e *_PortalEntry) UnmarshalYAML(node *yaml.Node) error {
	fields, err := seedMappingFields(node, "portal entry")
	if err != nil {
		return err
	}
	if err := checkSeedFields(fields, "portalEntries"); err != nil {
		return err
	}
	type plain _PortalEntry
	return node.Decode((*plain)(e))
}

func (e _PortalEntry) toCorePortalEntry() *core.PortalEntry {
	return &core.PortalEntry{
		Name:    e.Name,
		Scheme:  e.Scheme,
		Host:    e.Host,
		Port:    e.Port,
		Enabled: !e.Disabled,
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
	// Disabled is optional and defaults to false. A seed declares the exception,
	// so Hub keeps the positive spelling of the switch it stores: enabled.
	Disabled bool `yaml:"disabled"`
}

// seedMappingFields decodes the YAML mapping of one seed entity.
func seedMappingFields(node *yaml.Node, entity string) (map[string]yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s must be a YAML mapping", entity)
	}
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// checkSeedFields rejects a field the section does not declare, and names the
// switch Hub stores as enabled under the name a seed declares instead. Decoding
// ignores a field the payload does not declare, so a typo or a renamed field
// would silently leave the entity at its default.
func checkSeedFields(fields map[string]yaml.Node, section string) error {
	return checkSeedEntityFields(fields, section, fields["name"].Value)
}

// checkSeedEntityFields checks one entity mapping, naming it with the label the
// caller passes: a nested mapping such as cors carries no name of its own.
func checkSeedEntityFields(fields map[string]yaml.Node, section string, name string) error {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if seedSectionFields[section][key] {
			continue
		}
		if key == "enabled" {
			return fmt.Errorf("%s declares \"enabled\"; a seed turns configuration off with \"disabled: true\"",
				seedEntityLabel(section, name))
		}
		if key == "builtIn" {
			// TODO: Report builtIn as an unknown field once the built-in entities
			// are a release behind.
			return fmt.Errorf("%s declares \"builtIn\"; Hub owns the built-in entities, so a seed cannot declare it",
				seedEntityLabel(section, name))
		}
		return fmt.Errorf("%s declares unknown field %q", seedEntityLabel(section, name), key)
	}
	return nil
}

// seedEntityLabel names the entity an error is about.
func seedEntityLabel(section string, name string) string {
	kind := map[string]string{
		"portalSites":    "portal site",
		"portalEntries":  "portal entry",
		"portalRules":    "portal rule",
		"portalCerts":    "portal certificate",
		"portalSiteCors": "portal site cors",
	}[section]
	return fmt.Sprintf("%s %q", kind, name)
}

// toSeedRule carries the rule itself and the entry it joins: a seed expresses
// the entry either by naming one it declares or by declaring the access Hub
// ensures when it applies the rule.
func (r _PortalRule) toSeedRule() *seedRule {
	return &seedRule{
		Rule: &core.PortalRule{FieldSources: ruleFieldSources(r.Sources),
			Name:                    r.Name,
			MatchPathPrefix:         r.MatchPathPrefix,
			RouteType:               r.RouteType,
			RouteSiteName:           r.RouteSiteName,
			RouteRedirectionPattern: r.RouteRedirectionPattern,
			RoutePathPrefix:         r.RoutePathPrefix,
			Enabled:                 !r.Disabled,
		},
		EntryName: r.EntryName,
		Entry: core.PortalEntry{
			Scheme: r.MatchScheme,
			Host:   r.MatchHost,
			Port:   r.MatchPort,
		},
	}
}

// ruleFieldSources drops the fields a seed declares for the entry a rule joins:
// the entry owns the access, and an entry carries no field sources of its own.
func ruleFieldSources(sources core.FieldSources) core.FieldSources {
	if len(sources) == 0 {
		return nil
	}
	ret := core.FieldSources{}
	for path, source := range sources {
		switch path {
		case "/matchScheme", "/matchHost", "/matchPort":
			continue
		}
		ret[path] = source
	}
	return ret
}

// toCorePortalRule returns the rule the seed declares, without resolving the
// entry it belongs to.
func (r _PortalRule) toCorePortalRule() *core.PortalRule {
	return r.toSeedRule().Rule
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
	// Disabled is optional and defaults to false. A seed declares the exception,
	// so Hub keeps the positive spelling of the switch it stores: enabled.
	Disabled bool `yaml:"disabled"`
}

func (s *_PortalSite) UnmarshalYAML(node *yaml.Node) error {
	fields, err := seedMappingFields(node, "portal site")
	if err != nil {
		return err
	}
	if err := checkSeedFields(fields, "portalSites"); err != nil {
		return err
	}
	if cors, ok := fields["cors"]; ok && cors.Kind == yaml.MappingNode {
		corsFields, err := seedMappingFields(&cors, "portal site cors")
		if err != nil {
			return err
		}
		if err := checkSeedEntityFields(corsFields, "portalSiteCors", fields["name"].Value); err != nil {
			return err
		}
	}
	type plain _PortalSite
	return node.Decode((*plain)(s))
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
		Enabled:       !s.Disabled,
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
	// Disabled is optional and defaults to false. A seed declares the exception,
	// so Hub keeps the positive spelling of the switch it stores: enabled.
	Disabled bool `yaml:"disabled"`
}

func (c *_PortalCert) UnmarshalYAML(node *yaml.Node) error {
	fields, err := seedMappingFields(node, "portal certificate")
	if err != nil {
		return err
	}
	if err := checkSeedFields(fields, "portalCerts"); err != nil {
		return err
	}
	type plain _PortalCert
	return node.Decode((*plain)(c))
}

func (c _PortalCert) toCorePortalCert() *core.PortalCert {
	cert := &core.PortalCert{FieldSources: c.Sources,
		Name:             c.Name,
		PublicKeyBase64:  c.PublicKeyBase64,
		PrivateKeyBase64: c.PrivateKeyBase64,
		Enabled:          !c.Disabled,
	}
	return cert
}

func (r *_PortalRule) UnmarshalYAML(node *yaml.Node) error {
	// TODO: Remove legacy field decoding from startup seeds when old YAML support
	// is retired, together with decodePortalRule compatibility logic.
	type plain _PortalRule
	return decodePortalRule(node, (*plain)(r))
}
