// Package legacy decodes schemas produced before runtime descriptors.
// Only registration adapters may depend on these input types.
package legacy

import "go.yorun.ai/skel/descriptor"

type AuthMode = descriptor.AuthMode

const ()

type DomainSchema struct {
	Domain      string            `json:"domain"`
	Description string            `json:"description,omitempty"`
	Hash        string            `json:"hash"`
	Full        bool              `json:"full"`
	Generated   *GeneratedInfo    `json:"generated"`
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

// GeneratedInfo records the compiler that generated a domain schema.
type GeneratedInfo struct {
	CompilerVersion string `json:"compilerVersion"`
}

type EnumSchema struct {
	Name             string            `json:"name"`
	SkelName         string            `json:"skelName"`
	Description      string            `json:"description,omitempty"`
	Deprecated       bool              `json:"deprecated,omitzero"`
	DeprecatedReason string            `json:"deprecatedReason,omitempty"`
	Hash             string            `json:"hash"`
	Items            []*EnumItemSchema `json:"items"`
}

type EnumItemSchema struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitzero"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
}

type DataSchema struct {
	Name             string          `json:"name"`
	SkelName         string          `json:"skelName"`
	Description      string          `json:"description,omitempty"`
	Deprecated       bool            `json:"deprecated,omitzero"`
	DeprecatedReason string          `json:"deprecatedReason,omitempty"`
	Hash             string          `json:"hash"`
	Sensitive        bool            `json:"sensitive,omitzero"`
	TypeParameters   []string        `json:"typeParameters,omitempty"`
	Members          []*MemberSchema `json:"members,omitempty"`
}

type ConfigSchema struct {
	Name             string          `json:"name"`
	SkelName         string          `json:"skelName"`
	Description      string          `json:"description,omitempty"`
	Deprecated       bool            `json:"deprecated,omitzero"`
	DeprecatedReason string          `json:"deprecatedReason,omitempty"`
	Hash             string          `json:"hash"`
	Pub              bool            `json:"pub"`
	Sensitive        bool            `json:"sensitive,omitzero"`
	Lifecycle        string          `json:"lifecycle"`
	Members          []*MemberSchema `json:"members,omitempty"`
}

type WebSchema struct {
	Name             string `json:"name"`
	SkelName         string `json:"skelName"`
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitzero"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
	Hash             string `json:"hash"`
	// AuthMode leaves legacy Web behavior intact when omitted.
	AuthMode  AuthMode               `json:"authMode,omitzero"`
	Audiences []*ActorAudienceSchema `json:"audiences"`
	MountPath string                 `json:"mountPath"`
}

type EventSchema struct {
	Name             string `json:"name"`
	SkelName         string `json:"skelName"`
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitzero"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
	Hash             string `json:"hash"`
	Pub              bool   `json:"pub"`
	// Ext exports the emitter contract for use by other domains.
	Ext       bool            `json:"ext,omitzero"`
	Sensitive bool            `json:"sensitive,omitzero"`
	Members   []*MemberSchema `json:"members,omitempty"`
}

type ActorSchema struct {
	Name             string                    `json:"name"`
	SkelName         string                    `json:"skelName"`
	Description      string                    `json:"description,omitempty"`
	Deprecated       bool                      `json:"deprecated,omitzero"`
	DeprecatedReason string                    `json:"deprecatedReason,omitempty"`
	Hash             string                    `json:"hash"`
	Vias             []descriptor.ActorViaKind `json:"vias"`
	AuthEnabled      bool                      `json:"authEnabled"`
	AuthCredential   *DataSchema               `json:"authCredential,omitempty"`
	AuthInfo         *DataSchema               `json:"authInfo,omitempty"`
	IdentifierField  string                    `json:"identifierField,omitempty"`
	AuthService      *ServiceSchema            `json:"authService,omitempty"`
	AuthMethod       *MethodSchema             `json:"authMethod,omitempty"`
	PermEnabled      bool                      `json:"permEnabled"`
	PermService      *ServiceSchema            `json:"permService,omitempty"`
	PermMethod       *MethodSchema             `json:"permMethod,omitempty"`
}

type ActorAudienceSchema struct {
	Name     string                  `json:"name"`
	SkelName string                  `json:"skelName"`
	Via      descriptor.ActorViaKind `json:"via,omitempty"`
}

type ServiceSchema struct {
	Name             string `json:"name"`
	SkelName         string `json:"skelName"`
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitzero"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
	Hash             string `json:"hash"`
	Pub              bool   `json:"pub"`
	// Api restricts calls to the Portal client path.
	Api bool `json:"api,omitzero"`
	// Ext exports the server contract for implementation by other domains.
	Ext       bool                   `json:"ext,omitzero"`
	AuthMode  AuthMode               `json:"authMode"`
	Audiences []*ActorAudienceSchema `json:"audiences,omitempty"`

	Require *PermRequire    `json:"require,omitempty"`
	Methods []*MethodSchema `json:"methods"`
}

type MethodSchema struct {
	Name               string          `json:"name"`
	SkelName           string          `json:"skelName"`
	Description        string          `json:"description,omitempty"`
	Deprecated         bool            `json:"deprecated,omitzero"`
	DeprecatedReason   string          `json:"deprecatedReason,omitempty"`
	Hash               string          `json:"hash"`
	Example            string          `json:"example,omitempty"`
	AuthMode           AuthMode        `json:"authMode"`
	Require            *PermRequire    `json:"require,omitempty"`
	InputDescription   string          `json:"inputDescription,omitempty"`
	ArgumentsSensitive bool            `json:"argumentsSensitive,omitzero"`
	OutputDescription  string          `json:"outputDescription,omitempty"`
	OutputExample      string          `json:"outputExample,omitempty"`
	ResultSensitive    bool            `json:"resultSensitive,omitzero"`
	Arguments          []*MemberSchema `json:"arguments,omitempty"`
	ResultType         *TypeSchema     `json:"resultType,omitempty"`
}

type ResourceSchema struct {
	Name             string                  `json:"name"`
	SkelName         string                  `json:"skelName"`
	Description      string                  `json:"description,omitempty"`
	Deprecated       bool                    `json:"deprecated,omitzero"`
	DeprecatedReason string                  `json:"deprecatedReason,omitempty"`
	Hash             string                  `json:"hash"`
	Checks           []*ResourceCheckSchema  `json:"checks,omitempty"`
	Actions          []*ResourceActionSchema `json:"actions"`
	CheckService     *ServiceSchema          `json:"checkService,omitempty"`
}

type ResourceActionSchema struct {
	Name             string                 `json:"name"`
	PermissionCode   string                 `json:"permissionCode"`
	Description      string                 `json:"description,omitempty"`
	Deprecated       bool                   `json:"deprecated,omitzero"`
	DeprecatedReason string                 `json:"deprecatedReason,omitempty"`
	Checks           []*ResourceCheckSchema `json:"checks,omitempty"`
}

type ResourceCheckSchema struct {
	Name             string          `json:"name"`
	Deprecated       bool            `json:"deprecated,omitzero"`
	DeprecatedReason string          `json:"deprecatedReason,omitempty"`
	Method           *MethodSchema   `json:"method"`
	Arguments        []*MemberSchema `json:"arguments,omitempty"`
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
	ResourceSkelName string `json:"resourceSkelName"`
	ActionName       string `json:"actionName"`
	CheckName        string `json:"checkName"`
	ServiceSkelName  string `json:"serviceSkelName"`
	MethodSkelName   string `json:"methodSkelName"`
	// CodeArgumentName names the parameter receiving the injected permission code.
	CodeArgumentName string               `json:"codeArgumentName,omitempty"`
	Arguments        []*PermCheckArgument `json:"arguments,omitempty"`
}

type PermCheckArgument struct {
	Name     string      `json:"name"`
	JsonPath string      `json:"jsonPath"`
	Type     *TypeSchema `json:"type"`
}

type TaskSchema struct {
	Name             string           `json:"name"`
	SkelName         string           `json:"skelName"`
	Description      string           `json:"description,omitempty"`
	Deprecated       bool             `json:"deprecated,omitzero"`
	DeprecatedReason string           `json:"deprecatedReason,omitempty"`
	Hash             string           `json:"hash"`
	Triggers         []*TriggerSchema `json:"triggers"`
}

type TriggerSchema struct {
	Name               string          `json:"name"`
	SkelName           string          `json:"skelName"`
	Description        string          `json:"description,omitempty"`
	Deprecated         bool            `json:"deprecated,omitzero"`
	DeprecatedReason   string          `json:"deprecatedReason,omitempty"`
	Hash               string          `json:"hash"`
	InputDescription   string          `json:"inputDescription,omitempty"`
	ArgumentsSensitive bool            `json:"argumentsSensitive,omitzero"`
	Arguments          []*MemberSchema `json:"arguments,omitempty"`
}

type MemberSchema struct {
	Name             string      `json:"name"`
	Description      string      `json:"description,omitempty"`
	Deprecated       bool        `json:"deprecated,omitzero"`
	DeprecatedReason string      `json:"deprecatedReason,omitempty"`
	Example          string      `json:"example,omitempty"`
	Sensitive        bool        `json:"sensitive,omitzero"`
	Type             *TypeSchema `json:"type"`
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

	TypeKindTypeParameter TypeKind = "typeParameter"
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
	Nullable      bool          `json:"nullable,omitzero"`
	Scalar        Scalar        `json:"scalar,omitempty"`
	Name          string        `json:"name,omitempty"`
	SkelName      string        `json:"skelName,omitempty"`
	TypeArguments []*TypeSchema `json:"typeArguments,omitempty"`
	Element       *TypeSchema   `json:"element,omitempty"`
	Key           *TypeSchema   `json:"key,omitempty"`
	Value         *TypeSchema   `json:"value,omitempty"`
}
