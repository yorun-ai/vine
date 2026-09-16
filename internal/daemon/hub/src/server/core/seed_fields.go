package core

import (
	"strconv"
	"time"

	"go.yorun.ai/vine/util/vcode"
)

// SeedFields returns the entity values a seed compares against, keyed by the
// field names seeds use.
func (a *AppConfig) SeedFields() map[string]string {
	if a == nil {
		return map[string]string{}
	}
	return map[string]string{
		"value": a.Value,
	}
}

// SeedFields returns the entity values a seed compares against, keyed by the
// field names seeds use.
func (s *PortalSite) SeedFields() map[string]string {
	if s == nil {
		return map[string]string{}
	}
	return map[string]string{
		"type":          string(s.Type),
		"actorSkelName": s.ActorSkelName,
		"actorVia":      s.ActorVia,
		"corsMode":      string(s.Cors.Mode),
		"corsOrigins":   SeedJSONString(s.Cors.AllowedOrigins),
		"webName":       s.WebName,
		"disabled":      strconv.FormatBool(!s.Enabled),
	}
}

// SeedFields returns the entity values a seed compares against, keyed by the
// field names seeds use.
func (e *PortalEntry) SeedFields() map[string]string {
	if e == nil {
		return map[string]string{}
	}
	return map[string]string{
		"scheme":   e.Scheme,
		"host":     e.Host,
		"port":     strconv.Itoa(e.Port),
		"disabled": strconv.FormatBool(!e.Enabled),
	}
}

// SeedFields returns the entity values a seed compares against, keyed by the
// field names seeds use.
func (r *PortalRule) SeedFields() map[string]string {
	if r == nil {
		return map[string]string{}
	}
	return map[string]string{
		"matchPathPrefix":         r.MatchPathPrefix,
		"routeType":               r.RouteType,
		"routeSiteName":           r.RouteSiteName,
		"routeRedirectionPattern": r.RouteRedirectionPattern,
		"routePathPrefix":         r.RoutePathPrefix,
		"disabled":                strconv.FormatBool(!r.Enabled),
	}
}

// SeedFields returns the entity values a seed compares against, keyed by the
// field names seeds use.
func (c *PortalCert) SeedFields() map[string]string {
	if c == nil {
		return map[string]string{}
	}
	return map[string]string{
		"issuer":           c.Issuer,
		"domains":          SeedJSONString(c.Domains),
		"publicKeyBase64":  c.PublicKeyBase64,
		"privateKeyBase64": c.PrivateKeyBase64,
		"validFrom":        SeedTimeString(c.ValidFrom),
		"validTo":          SeedTimeString(c.ValidTo),
		"disabled":         strconv.FormatBool(!c.Enabled),
	}
}

// SeedJSONString renders a string list the way seeds carry it.
func SeedJSONString(value []string) string {
	if value == nil {
		value = []string{}
	}
	return vcode.MustMarshalJsonS(value)
}

// SeedTimeString renders a timestamp the way seeds carry it.
func SeedTimeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
