package redised

import (
	"fmt"
	"time"
)

// Entry rules

const (
	portalRulePrefix    = "portal:rule"
	portalRuleKeyFormat = portalRulePrefix + ":%s"
)

type PortalRule struct {
	Name               string `json:"name"`
	Scheme             string `json:"scheme"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	PathPrefix         string `json:"pathPrefix"`
	TargetType         string `json:"targetType"`
	SiteName           string `json:"siteName"`
	RedirectionPattern string `json:"redirectionPattern"`
}

func FormatPortalRuleKey(name string) string {
	return fmt.Sprintf(portalRuleKeyFormat, name)
}

func FormatPortalRulePrefix() string {
	return portalRulePrefix
}

// Entries

const (
	portalSitePrefix    = "portal:site"
	portalSiteKeyFormat = portalSitePrefix + ":%s"
)

type PortalSite struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	ActorVia PortalActorVia `json:"actorVia"`
	Cors     PortalCors     `json:"cors"`

	RpcgwConfig *PortalRpcgwConfig `json:"rpcgwConfig,omitempty"`
	WebgwConfig *PortalWebgwConfig `json:"webgwConfig,omitempty"`
}

type PortalCorsMode string

const (
	PortalCorsModeDisabled   PortalCorsMode = "DISABLED"
	PortalCorsModeSameDomain PortalCorsMode = "SAME_DOMAIN"
	PortalCorsModeStrict     PortalCorsMode = "STRICT"
)

type PortalCors struct {
	Mode           PortalCorsMode `json:"mode"`
	AllowedOrigins []string       `json:"allowedOrigins"`
}

type PortalActorVia struct {
	ActorSkelName string `json:"actorSkelName"`
	ActorVia      string `json:"actorVia"`
}

type PortalRpcgwConfig struct {
	Services []PortalRpcgwService `json:"services"`
}

type PortalRpcgwService struct {
	SkelName string `json:"skelName"`
}

type PortalWebgwConfig struct {
	WebName string `json:"webName"`
}

func FormatPortalSiteKey(name string) string {
	return fmt.Sprintf(portalSiteKeyFormat, name)
}

func FormatPortalSitePrefix() string {
	return portalSitePrefix
}

// Certificates

const (
	portalCertPrefix    = "portal:cert"
	portalCertKeyFormat = portalCertPrefix + ":%s"
)

type PortalCert struct {
	Name             string    `json:"name"`
	Issuer           string    `json:"issuer"`
	PublicKeyBase64  string    `json:"publicKeyBase64"`
	PrivateKeyBase64 string    `json:"privateKeyBase64"`
	ValidFrom        time.Time `json:"validFrom"`
	ValidTo          time.Time `json:"validTo"`
}

func FormatPortalCertKey(name string) string {
	return fmt.Sprintf(portalCertKeyFormat, name)
}

func FormatPortalCertPrefix() string {
	return portalCertPrefix
}
