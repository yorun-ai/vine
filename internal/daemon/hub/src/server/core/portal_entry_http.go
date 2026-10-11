package core

import (
	"encoding/json/v2"
	"fmt"
	"net"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
)

// PortalSwitchMode selects explicit or automatically derived transport behavior.
type PortalSwitchMode string

const (
	PortalSwitchModeAuto     PortalSwitchMode = "AUTO"
	PortalSwitchModeEnabled  PortalSwitchMode = "ENABLED"
	PortalSwitchModeDisabled PortalSwitchMode = "DISABLED"
)

// Valid reports whether the value is an explicit supported mode.
func (m PortalSwitchMode) Valid() bool {
	return m == PortalSwitchModeAuto || m == PortalSwitchModeEnabled || m == PortalSwitchModeDisabled
}

// UnmarshalJSON rejects null and invalid explicit modes; omission keeps the zero value.
func (m *PortalSwitchMode) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	mode := PortalSwitchMode(value)
	if !mode.Valid() {
		return fmt.Errorf("expected ENABLED, DISABLED or AUTO, got %s", data)
	}
	*m = mode
	return nil
}

func (m PortalSwitchMode) normalized(field string) PortalSwitchMode {
	if m == "" {
		return PortalSwitchModeAuto
	}
	ex.PanicNewIfNot(m.Valid(), ex.OperationFailed, ex.F("portal entry http.%s must be ENABLED, DISABLED or AUTO", field))
	return m
}

// PortalEntryHTTP stores configuration modes; omitted switches become AUTO.
type PortalEntryHTTP struct {
	Host      string           `json:"host"`
	Http      PortalSwitchMode `json:"http"`
	HttpPort  int              `json:"httpPort"`
	Https     PortalSwitchMode `json:"https"`
	HttpsPort int              `json:"httpsPort"`
	AutoHTTPS PortalSwitchMode `json:"autohttps"`
}

// PortalEntryEffectiveHTTP is Hub's complete transport decision.
type PortalEntryEffectiveHTTP struct {
	Host             string
	HttpEnabled      bool
	HttpPort         int
	HttpsEnabled     bool
	HttpsPort        int
	AutoHTTPSEnabled bool
	// HTTPSReason explains the decision without considering certificate validity.
	HTTPSReason string
}

// DefaultPortalEntryHTTP leaves switches automatic and ports at their defaults.
func DefaultPortalEntryHTTP() PortalEntryHTTP {
	return PortalEntryHTTP{Http: PortalSwitchModeAuto, Https: PortalSwitchModeAuto, AutoHTTPS: PortalSwitchModeAuto}
}

// Effective resolves switches using certificate presence, never validity dates.
func (h PortalEntryHTTP) Effective(hasCertificate bool) PortalEntryEffectiveHTTP {
	domain := h.Host != "" && net.ParseIP(h.Host) == nil
	httpEnabled := h.Http != PortalSwitchModeDisabled
	httpsEnabled := domain && hasCertificate && (h.Https != PortalSwitchModeDisabled)
	reason := "ENABLED"
	if !httpsEnabled {
		switch {
		case h.Https == PortalSwitchModeDisabled:
			reason = "DISABLED"
		case !domain:
			reason = "NO_DOMAIN"
		default:
			reason = "NO_MATCHING_CERTIFICATE"
		}
	}
	return PortalEntryEffectiveHTTP{Host: h.Host, HttpEnabled: httpEnabled, HttpPort: defaultPort(h.HttpPort, 80), HttpsEnabled: httpsEnabled, HttpsPort: defaultPort(h.HttpsPort, 443), AutoHTTPSEnabled: domain && httpEnabled && httpsEnabled && (h.AutoHTTPS != PortalSwitchModeDisabled), HTTPSReason: reason}
}
func defaultPort(port int, fallback int) int {
	if port == 0 {
		return fallback
	}
	return port
}

// Equal compares configuration intent, including automatic versus explicit switches.
func (h PortalEntryHTTP) Equal(other PortalEntryHTTP) bool {
	return h == other
}

// ValidatePortalEntryCertificates rejects explicit HTTPS or redirect activation
// without an enabled matching certificate. Automatic switches may wait for one.
// Certificate validity dates are deliberately excluded from this validation.
func ValidatePortalEntryCertificates(entry PortalEntry, certs []*PortalCert) {
	h := entry.Http
	if h.Https != PortalSwitchModeEnabled && h.AutoHTTPS != PortalSwitchModeEnabled {
		return
	}
	ex.PanicNewIfNot(HasPortalCertificate(h.Host, certs), ex.OperationFailed,
		ex.F("portal entry %q: explicit HTTPS or autohttps requires an enabled matching certificate for host %q", entry.Name, h.Host))
}

// HasPortalCertificate matches enabled certificate names; validity belongs to Portal.
// A wildcard entry requires a certificate covering its entire wildcard namespace.
func HasPortalCertificate(host string, certs []*PortalCert) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" || net.ParseIP(host) != nil {
		return false
	}
	for _, cert := range certs {
		if !cert.Enabled {
			continue
		}
		for _, domain := range cert.Domains {
			domain = strings.ToLower(strings.TrimSuffix(domain, "."))
			if domain == host {
				return true
			}
			if !strings.HasPrefix(host, "*.") && strings.HasPrefix(domain, "*.") {
				label, ok := strings.CutSuffix(host, domain[1:])
				if ok && label != "" && !strings.Contains(label, ".") {
					return true
				}
			}
		}
	}
	return false
}

// PortalEntryAccess describes one transport's scheme and listener port.
type PortalEntryAccess struct {
	Scheme string
	Port   int
}

// EffectiveAccesses returns only the transports enabled by Hub's decision.
// The entry must have EffectiveHTTP populated by its repository.
func (e PortalEntry) EffectiveAccesses() []PortalEntryAccess {
	return e.EffectiveHTTP.accesses()
}

// ReservedAccesses returns potential transports for static conflict checks,
// including automatic HTTPS before a certificate is available. It ignores
// EffectiveHTTP, so certificate changes cannot release a reserved listener.
func (e PortalEntry) ReservedAccesses() []PortalEntryAccess {
	return normalizePortalEntry(e).Http.Effective(true).accesses()
}

func (h PortalEntryEffectiveHTTP) accesses() []PortalEntryAccess {
	accesses := []PortalEntryAccess{}
	if h.HttpEnabled {
		accesses = append(accesses, PortalEntryAccess{Scheme: "http", Port: h.HttpPort})
	}
	if h.HttpsEnabled {
		accesses = append(accesses, PortalEntryAccess{Scheme: "https", Port: h.HttpsPort})
	}
	return accesses
}

// NormalizePortalEntry validates static configuration without consulting certificates.
func NormalizePortalEntry(entry PortalEntry) PortalEntry { return normalizePortalEntry(entry) }
func normalizePortalEntryHTTP(entry PortalEntry) PortalEntry {
	entry.Protocol = strings.ToLower(strings.TrimSpace(entry.Protocol))
	ex.PanicNewIfNot(entry.Protocol == "http", ex.OperationFailed, ex.F("unknown portal entry protocol: %s", entry.Protocol))
	if entry.Http == nil {
		entry.Http = new(DefaultPortalEntryHTTP())
	} else {
		entry.Http = new(*entry.Http)
	}
	h := entry.Http
	h.Http = h.Http.normalized("http")
	h.Https = h.Https.normalized("https")
	h.AutoHTTPS = h.AutoHTTPS.normalized("autohttps")
	host := strings.TrimSpace(h.Host)
	h.Host = strings.ToLower(strings.TrimSuffix(host, "."))
	// Only an omitted host may become empty. Reject multiple trailing dots so
	// repeated normalization cannot change the entry's identity again.
	ex.PanicNewIfNot((host == "" || h.Host != "") && !strings.HasSuffix(h.Host, ".") && portalEntryHostAccepted(h.Host), ex.OperationFailed, "portal entry host must be a hostname, IP or leading *. wildcard without a port")
	ex.PanicNewIfNot(h.HttpPort >= 0 && h.HttpPort <= 65535 && h.HttpsPort >= 0 && h.HttpsPort <= 65535, ex.OperationFailed, "portal entry port must be between 0 and 65535")
	h.HttpPort = defaultPort(h.HttpPort, 80)
	h.HttpsPort = defaultPort(h.HttpsPort, 443)
	domain := h.Host != "" && net.ParseIP(h.Host) == nil
	ex.PanicNewIfNot(h.Https != PortalSwitchModeEnabled || domain, ex.OperationFailed, "HTTPS requires a domain host")
	// Reserve potential HTTPS bindings even before a certificate is uploaded.
	effective := h.Effective(true)
	ex.PanicNewIfNot(effective.HttpEnabled || effective.HttpsEnabled, ex.OperationFailed, "portal entry must enable HTTP or HTTPS")
	ex.PanicNewIfNot(h.AutoHTTPS != PortalSwitchModeEnabled || (domain && effective.HttpEnabled && effective.HttpsEnabled), ex.OperationFailed, "autohttps requires a domain host and both HTTP and HTTPS")
	ex.PanicNewIfNot(!effective.HttpEnabled || !effective.HttpsEnabled || effective.HttpPort != effective.HttpsPort, ex.OperationFailed, "HTTP and HTTPS cannot listen on the same port")
	return entry
}
