package skel

import (
	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type DomainSchema struct {
	Domain      string            `json:"domain"`
	Description string            `json:"description,omitempty"`
	Hash        string            `json:"hash"`
	Full        bool              `json:"full"`
	Generated   *GeneratedInfo    `json:"generated,omitempty"`
	Enums       []*EnumSchema     `json:"enums,omitempty"`
	Data        []*DataSchema     `json:"data,omitempty"`
	Configs     []*ConfigSchema   `json:"configs,omitempty"`
	Webs        []*WebSchema      `json:"webs,omitempty"`
	Events      []*EventSchema    `json:"events,omitempty"`
	Actors      []*ActorSchema    `json:"actors,omitempty"`
	Resources   []*ResourceSchema `json:"resources,omitempty"`
	Services    []*ServiceSchema  `json:"services,omitempty"`
	Tasks       []*TaskSchema     `json:"tasks,omitempty"`
}

type EnumSchema struct {
	Name        string            `json:"name"`
	SkelName    string            `json:"skelName"`
	Description string            `json:"description,omitempty"`
	Hash        string            `json:"hash"`
	Items       []*EnumItemSchema `json:"items"`
}

type EnumItemSchema struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type DataSchema struct {
	Name           string          `json:"name"`
	SkelName       string          `json:"skelName"`
	Description    string          `json:"description,omitempty"`
	Hash           string          `json:"hash"`
	TypeParameters []string        `json:"typeParameters,omitempty"`
	Members        []*MemberSchema `json:"members,omitempty"`
}

type ConfigSchema struct {
	Name        string          `json:"name"`
	SkelName    string          `json:"skelName"`
	Description string          `json:"description,omitempty"`
	Hash        string          `json:"hash"`
	Pub         bool            `json:"pub"`
	Lifecycle   string          `json:"lifecycle"`
	Members     []*MemberSchema `json:"members,omitempty"`
}

type WebSchema struct {
	Name        string                 `json:"name"`
	SkelName    string                 `json:"skelName"`
	Description string                 `json:"description,omitempty"`
	Hash        string                 `json:"hash"`
	Audiences   []*ActorAudienceSchema `json:"audiences"`
}

type EventSchema struct {
	Name        string          `json:"name"`
	SkelName    string          `json:"skelName"`
	Description string          `json:"description,omitempty"`
	Hash        string          `json:"hash"`
	Pub         bool            `json:"pub"`
	Members     []*MemberSchema `json:"members,omitempty"`
}

type ActorSchema struct {
	Name           string         `json:"name"`
	SkelName       string         `json:"skelName"`
	Description    string         `json:"description,omitempty"`
	Hash           string         `json:"hash"`
	Vias           []ActorVia     `json:"vias"`
	AuthEnabled    bool           `json:"authEnabled"`
	AuthCredential *DataSchema    `json:"authCredential,omitempty"`
	AuthInfo       *DataSchema    `json:"authInfo,omitempty"`
	AuthService    *ServiceSchema `json:"authService,omitempty"`
	AuthMethod     *MethodSchema  `json:"authMethod,omitempty"`
	PermEnabled    bool           `json:"permEnabled"`
	PermService    *ServiceSchema `json:"permService,omitempty"`
	PermMethod     *MethodSchema  `json:"permMethod,omitempty"`
}

type ActorAudienceSchema struct {
	Name     string   `json:"name"`
	SkelName string   `json:"skelName"`
	Via      ActorVia `json:"via,omitempty"`
}

type ServiceSchema struct {
	Name        string                 `json:"name"`
	SkelName    string                 `json:"skelName"`
	Description string                 `json:"description,omitempty"`
	Hash        string                 `json:"hash"`
	Pub         bool                   `json:"pub"`
	AuthMode    AuthMode               `json:"authMode"`
	Audiences   []*ActorAudienceSchema `json:"audiences,omitempty"`

	Require *PermRequire    `json:"require,omitempty"`
	Methods []*MethodSchema `json:"methods"`
}

func (s *ServiceSchema) MethodByName(skelName string) (*MethodSchema, bool) {
	for _, method := range s.Methods {
		if method.SkelName == skelName {
			return method, true
		}
	}
	return nil, false
}

func (s *ServiceSchema) HasAudience(actorSkelName string, via ActorVia) bool {
	for _, audience := range s.Audiences {
		if audience.SkelName != actorSkelName {
			continue
		}
		if audience.Via == "" || audience.Via == via {
			return true
		}
	}
	return false
}

type MethodSchema struct {
	Name              string          `json:"name"`
	SkelName          string          `json:"skelName"`
	Description       string          `json:"description,omitempty"`
	Hash              string          `json:"hash"`
	Example           string          `json:"example,omitempty"`
	AuthMode          AuthMode        `json:"authMode"`
	Require           *PermRequire    `json:"require,omitempty"`
	InputDescription  string          `json:"inputDescription,omitempty"`
	OutputDescription string          `json:"outputDescription,omitempty"`
	OutputExample     string          `json:"outputExample,omitempty"`
	Arguments         []*MemberSchema `json:"arguments,omitempty"`
	ResultType        *TypeSchema     `json:"resultType,omitempty"`
}

type ResourceSchema struct {
	Name         string                  `json:"name"`
	SkelName     string                  `json:"skelName"`
	Description  string                  `json:"description,omitempty"`
	Hash         string                  `json:"hash"`
	Checks       []*ResourceCheckSchema  `json:"checks,omitempty"`
	Actions      []*ResourceActionSchema `json:"actions"`
	CheckService *ServiceSchema          `json:"checkService,omitempty"`
}

type ResourceActionSchema struct {
	Name           string                 `json:"name"`
	PermissionCode string                 `json:"permissionCode"`
	Description    string                 `json:"description,omitempty"`
	Checks         []*ResourceCheckSchema `json:"checks,omitempty"`
}

type ResourceCheckSchema struct {
	Name      string          `json:"name"`
	Method    *MethodSchema   `json:"method"`
	Arguments []*MemberSchema `json:"arguments,omitempty"`
}

type PermRequireMode string

const (
	PermRequireModeCode  PermRequireMode = "code"
	PermRequireModeCheck PermRequireMode = "check"
	PermRequireModeAll   PermRequireMode = "all"
	PermRequireModeAny   PermRequireMode = "any"
)

type PermRequire struct {
	Expr *PermExpr `json:"expr"`
}

type PermExpr struct {
	Mode     PermRequireMode      `json:"mode"`
	Code     string               `json:"code,omitempty"`
	Check    *PermCheckInvocation `json:"check,omitempty"`
	Children []*PermExpr          `json:"children,omitempty"`
}

type PermCheckInvocation struct {
	ResourceSkelName string               `json:"resourceSkelName"`
	ActionName       string               `json:"actionName"`
	CheckName        string               `json:"checkName"`
	ServiceSkelName  string               `json:"serviceSkelName"`
	MethodSkelName   string               `json:"methodSkelName"`
	Arguments        []*PermCheckArgument `json:"arguments,omitempty"`
}

type PermCheckArgument struct {
	Name     string      `json:"name"`
	JsonPath string      `json:"jsonPath"`
	Type     *TypeSchema `json:"type"`
}

type TaskSchema struct {
	Name        string           `json:"name"`
	SkelName    string           `json:"skelName"`
	Description string           `json:"description,omitempty"`
	Hash        string           `json:"hash"`
	Triggers    []*TriggerSchema `json:"triggers"`
}

type TriggerSchema struct {
	Name             string          `json:"name"`
	SkelName         string          `json:"skelName"`
	Description      string          `json:"description,omitempty"`
	Hash             string          `json:"hash"`
	Example          string          `json:"example,omitempty"`
	InputDescription string          `json:"inputDescription,omitempty"`
	Arguments        []*MemberSchema `json:"arguments,omitempty"`
}

type MemberSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Example     string      `json:"example,omitempty"`
	Type        *TypeSchema `json:"type"`
}

type TypeKind string

const (
	TypeKindScalar TypeKind = "scalar"
	TypeKindList   TypeKind = "list"
	TypeKindMap    TypeKind = "map"
	TypeKindEnum   TypeKind = "enum"
	TypeKindData   TypeKind = "data"
	TypeKindConfig TypeKind = "config"
	TypeKindEvent  TypeKind = "event"

	TypeKindTypeParameter      TypeKind = "typeParameter"
	TypeKindSkelPermissionCode TypeKind = "permissionCode"
)

type Scalar string

const (
	ScalarString        Scalar = "string"
	ScalarBool          Scalar = "bool"
	ScalarInt           Scalar = "int"
	ScalarLong          Scalar = "long"
	ScalarFloat         Scalar = "float"
	ScalarDouble        Scalar = "double"
	ScalarDecimal       Scalar = "decimal"
	ScalarJson          Scalar = "json"
	ScalarUuid          Scalar = "uuid"
	ScalarTimestamp     Scalar = "timestamp"
	ScalarDuration      Scalar = "duration"
	ScalarLocalDate     Scalar = "localdate"
	ScalarLocalTime     Scalar = "localtime"
	ScalarLocalDateTime Scalar = "localdatetime"
	ScalarBinary        Scalar = "binary"
)

type TypeSchema struct {
	Kind          TypeKind      `json:"kind"`
	Nullable      bool          `json:"nullable,omitempty"`
	Scalar        Scalar        `json:"scalar,omitempty"`
	Name          string        `json:"name,omitempty"`
	SkelName      string        `json:"skelName,omitempty"`
	TypeArguments []*TypeSchema `json:"typeArguments,omitempty"`
	Element       *TypeSchema   `json:"element,omitempty"`
	Key           *TypeSchema   `json:"key,omitempty"`
	Value         *TypeSchema   `json:"value,omitempty"`
}

var schemasByDomain = map[string]*DomainSchema{}

func RegisterDomainSchema(schema *DomainSchema) {
	schema.checkCompilerVersion()

	registered, ok := schemasByDomain[schema.Domain]
	if !ok {
		schemasByDomain[schema.Domain] = schema
		return
	}

	// Split Go generation registers the pub package first through the regular
	// package import chain. The later regular full schema is the only duplicate
	// registration that should replace an existing schema for the same domain.
	if !registered.Full && schema.Full {
		schemasByDomain[schema.Domain] = schema
		return
	}
	panic("domain schema already registered: " + schema.Domain)
}

func (s *DomainSchema) checkCompilerVersion() {
	vpre.Check(s.Generated != nil && s.Generated.CompilerVersion != "",
		"domain schema %s missing generated compiler version; Vine requires skelc version %s or higher",
		s.Domain, MinSkelcVersion())

	if s.Generated.CompilerVersion == buildinfo.DevVersion {
		return
	}

	compilerVersion, err := semver.NewVersion(s.Generated.CompilerVersion)
	vpre.CheckNilError(err, "parse generated compiler version %s failed", s.Generated.CompilerVersion)

	minVersion := semver.MustParse(MinSkelcVersion())
	vpre.Check(compilerVersion.Compare(minVersion) >= 0,
		"domain schema %s generated by skelc %s is lower than Vine required skelc version %s",
		s.Domain, s.Generated.CompilerVersion, minVersion.Original())
}

func RegisteredDomainSchemas() []*DomainSchema {
	return vmap.SortedValues(schemasByDomain)
}
